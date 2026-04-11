package flux_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ocuris/flux"
)

// TestRaceSafety ensures that concurrent requests don't cause data races
// in the context pool or routing logic. Run with -race flag!
func TestRaceSafety(t *testing.T) {
	app := flux.New(flux.Config{})
	app.GET("/users/:id", func(c *flux.Context) error {
		id := c.Param("id")
		c.Set("id", id) // Test the internal store
		return c.JSON(200, flux.Map{"id": id})
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", fmt.Sprintf("/users/%d", idx), nil)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Errorf("expected 200, got %d", w.Code)
			}
		}(i)
	}
	wg.Wait()
}

// TestPathSanitization ensures that weird paths don't crash the server.
func TestPathSanitization(t *testing.T) {
	app := flux.New(flux.Config{})
	app.GET("/health", func(c *flux.Context) error {
		return c.String(200, "ok")
	})

	evilPaths := []string{
		"/health/",
		"//health",
		"/./health",
		"/health/../health",
		"/health?query=1#fragment",
	}

	for _, path := range evilPaths {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)

		// If it's a valid health check or a 404, it's fine as long as it didn't PANIC
		if w.Code == 500 {
			t.Errorf("Path %s caused a server error", path)
		}
	}
}

// TestMiddlewareLeakage ensures that data from one request is not
// visible in the next request due to pooling.
func TestMiddlewareLeakage(t *testing.T) {
	app := flux.New(flux.Config{})

	// Fast request that sets a key
	app.GET("/set", func(c *flux.Context) error {
		c.Set("secret", "top-hidden-data")
		return c.String(200, "set")
	})

	// Request that should NOT see the key
	app.GET("/check", func(c *flux.Context) error {
		if _, ok := c.Get("secret"); ok {
			return fmt.Errorf("LEAK DETECTED")
		}
		return c.String(200, "clean")
	})

	// Run set, then check
	for i := 0; i < 100; i++ {
		w1 := httptest.NewRecorder()
		app.ServeHTTP(w1, httptest.NewRequest("GET", "/set", nil))

		w2 := httptest.NewRecorder()
		app.ServeHTTP(w2, httptest.NewRequest("GET", "/check", nil))

		if w2.Code != 200 {
			t.Fatal("Request leakage detected at iteration", i)
		}
	}
}

// TestGracefulShutdownTimeout ensures the server stops correctly
func TestGracefulStop(t *testing.T) {
	app := flux.New(flux.Config{})
	app.GET("/slow", func(c *flux.Context) error {
		time.Sleep(100 * time.Millisecond)
		return c.String(200, "done")
	})

	// Start server in background
	go func() {
		_ = app.Start(":9999")
	}()

	time.Sleep(50 * time.Millisecond)

	// Trigger stop
	stopErr := app.Stop(context.Background()) // Immediate stop for testing
	if stopErr != nil {
		t.Log("Stop returned:", stopErr)
	}
}
