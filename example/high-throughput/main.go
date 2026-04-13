package main

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ocuris/flux"
)

// ── Cache ─────────────────────────────────────────────────────────────────────

// cacheItem is a single cached entry with an expiry time.
type cacheItem struct {
	value     any
	expiresAt time.Time
}

// Cache is a thread-safe in-memory key→value store with per-item TTL.
// Expired items are evicted lazily on read. For production, replace with Redis.
type Cache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
}

func newCache() *Cache { return &Cache{items: make(map[string]*cacheItem)} }

func (c *Cache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	c.items[key] = &cacheItem{value: value, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

// Get returns the cached value and true, or nil and false if missing/expired.
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(item.expiresAt) {
		return nil, false
	}
	return item.value, true
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// ── Metrics ───────────────────────────────────────────────────────────────────

// Metrics tracks request and cache statistics using atomic counters for lock-free
// increments and a mutex-protected sample ring for latency averaging.
type Metrics struct {
	requests    atomic.Int64
	cacheHits   atomic.Int64
	cacheMisses atomic.Int64

	mu      sync.Mutex
	samples []time.Duration // ring buffer — last 1000 entries
}

func (m *Metrics) Record(latency time.Duration, hit bool) {
	m.requests.Add(1)
	if hit {
		m.cacheHits.Add(1)
	} else {
		m.cacheMisses.Add(1)
	}

	m.mu.Lock()
	if len(m.samples) >= 1000 {
		m.samples = m.samples[1:]
	}
	m.samples = append(m.samples, latency)
	m.mu.Unlock()
}

func (m *Metrics) Snapshot() map[string]any {
	req := m.requests.Load()
	hits := m.cacheHits.Load()
	misses := m.cacheMisses.Load()

	var hitRate float64
	if req > 0 {
		hitRate = float64(hits) / float64(req) * 100
	}

	m.mu.Lock()
	var totalNs int64
	for _, s := range m.samples {
		totalNs += s.Nanoseconds()
	}
	n := int64(len(m.samples))
	m.mu.Unlock()

	var avgLatencyMs float64
	if n > 0 {
		avgLatencyMs = float64(totalNs) / float64(n) / 1e6
	}

	return map[string]any{
		"total_requests": req,
		"cache_hits":     hits,
		"cache_misses":   misses,
		"hit_rate_pct":   hitRate,
		"avg_latency_ms": avgLatencyMs,
		"sample_window":  n,
	}
}

// ── Domain ────────────────────────────────────────────────────────────────────

// UserProfile is the data we cache and serve.
type UserProfile struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Tier  string `json:"tier"`
}

// simulateDB mimics a 1 ms database round-trip.
func simulateDB(userID int) (*UserProfile, bool) {
	time.Sleep(time.Millisecond) // remove in production
	profiles := map[int]*UserProfile{
		1: {ID: 1, Name: "Alice", Email: "alice@example.com", Tier: "premium"},
		2: {ID: 2, Name: "Bob", Email: "bob@example.com", Tier: "free"},
		3: {ID: 3, Name: "Carol", Email: "carol@example.com", Tier: "business"},
	}
	p, ok := profiles[userID]
	return p, ok
}

// ── Globals ───────────────────────────────────────────────────────────────────

var (
	cache   = newCache()
	metrics = &Metrics{}
)

// ── Application entry point ───────────────────────────────────────────────────

func main() {
	app := flux.New(flux.Config{
		Title:       "Flux High-Throughput Example",
		Version:     "1.0.0",
		Description: "Demonstrates in-memory caching with TTL and real-time performance metrics.",
		Debug:       false,
	})

	// Keep middleware minimal: only Recover and RequestID.
	// Logger adds per-request I/O; skip it when throughput is the priority.
	app.Use(flux.Recover())
	app.Use(flux.RequestID())
	app.Use(flux.SecurityHeaders())
	// app.Use(flux.Logger()) // uncomment during debugging

	app.GET("/users/:id", flux.Doc(
		"Get User (cached)",
		"Returns a user profile. Served from cache after the first request. Cache TTL is 5 seconds.",
		"users",
	).
		Param("id", "path", "Numeric user ID (1–3 in this demo)", "integer", true).
		Response(200, "User profile with cache metadata", "application/json", nil).
		Response(400, "Invalid ID", "application/json", nil).
		Response(404, "User not found", "application/json", nil),
		getUser,
	)

	app.GET("/metrics", flux.Doc(
		"Performance Metrics",
		"Returns live cache hit rate, request count, and average latency.",
		"observability",
	).Response(200, "Metrics snapshot", "application/json", nil),
		getMetrics,
	)

	app.DELETE("/cache/:id", flux.Doc(
		"Invalidate Cache Entry",
		"Removes the cached entry for a user, forcing the next request to hit the database.",
		"observability",
	).
		Param("id", "path", "User ID whose cache entry to evict", "integer", true).
		Response(200, "Eviction confirmation", "application/json", nil),
		invalidateCache,
	)

	if err := app.Start(":8005"); err != nil {
		panic(err)
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func getUser(c *flux.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return flux.NewHTTPError(400, "Invalid user ID — must be a positive integer")
	}

	key := "user:" + strconv.Itoa(id)
	start := time.Now()

	if cached, ok := cache.Get(key); ok {
		lat := time.Since(start)
		metrics.Record(lat, true)
		return c.JSON(200, flux.Map{
			"user":       cached,
			"cached":     true,
			"latency_ms": float64(lat.Nanoseconds()) / 1e6,
		})
	}

	profile, found := simulateDB(id)
	if !found {
		return flux.NewHTTPError(404, "User not found")
	}

	cache.Set(key, profile, 5*time.Second)

	lat := time.Since(start)
	metrics.Record(lat, false)

	return c.JSON(200, flux.Map{
		"user":       profile,
		"cached":     false,
		"latency_ms": float64(lat.Nanoseconds()) / 1e6,
	})
}

func getMetrics(c *flux.Context) error {
	snap := metrics.Snapshot()
	snap["cache_size"] = cache.Size()
	return c.JSON(200, snap)
}

func invalidateCache(c *flux.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return flux.NewHTTPError(400, "Invalid user ID — must be a positive integer")
	}
	key := "user:" + strconv.Itoa(id)
	cache.Delete(key)
	return c.JSON(200, flux.Map{
		"message": "Cache entry evicted",
		"key":     key,
	})
}
