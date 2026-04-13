package main

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ocuris/flux"
)

// User is the domain model returned to API clients.
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserRequest is the payload for POST /users.
type CreateUserRequest struct {
	Name  string `json:"name"  validate:"required,min=3,max=50"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age"   validate:"gte=0,lte=120"`
}

// UpdateUserRequest is the payload for PUT /users/:id.
// All fields are optional — only non-empty / non-zero values are applied.
type UpdateUserRequest struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Age   int    `json:"age,omitempty"`
}

// --- in-memory store (protected by a RWMutex for concurrent safety) ---

type userStore struct {
	mu     sync.RWMutex
	users  map[int]*User
	nextID int
}

func newUserStore() *userStore {
	return &userStore{users: make(map[int]*User), nextID: 1}
}

func (s *userStore) list() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	return out
}

func (s *userStore) get(id int) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	return u, ok
}

func (s *userStore) create(req CreateUserRequest) *User {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := &User{
		ID:        s.nextID,
		Name:      strings.TrimSpace(req.Name),
		Email:     strings.ToLower(strings.TrimSpace(req.Email)),
		Age:       req.Age,
		CreatedAt: time.Now().UTC(),
	}
	s.users[s.nextID] = u
	s.nextID++
	return u
}

func (s *userStore) update(id int, req UpdateUserRequest) (*User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return nil, false
	}
	if req.Name != "" {
		u.Name = strings.TrimSpace(req.Name)
	}
	if req.Email != "" {
		u.Email = strings.ToLower(strings.TrimSpace(req.Email))
	}
	if req.Age != 0 {
		u.Age = req.Age
	}
	return u, true
}

func (s *userStore) delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[id]; !ok {
		return false
	}
	delete(s.users, id)
	return true
}

func (s *userStore) search(q string) []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q = strings.ToLower(q)
	var results []*User
	for _, u := range s.users {
		if strings.Contains(strings.ToLower(u.Name), q) ||
			strings.Contains(strings.ToLower(u.Email), q) {
			results = append(results, u)
		}
	}
	return results
}

var store = newUserStore()

func main() {
	app := flux.New(flux.Config{
		Title:       "Flux Basic CRUD Example",
		Version:     "1.0.0",
		Description: "A minimal user-management REST API demonstrating Flux core patterns.",
		Debug:       true,
	})

	// Middleware — order matters: recover first, then enrich, then log.
	app.Use(flux.Recover())
	app.Use(flux.RequestID())
	app.Use(flux.SecurityHeaders())
	app.Use(flux.Logger())
	app.Use(flux.CORS(flux.CORSConfig{
		// Restrict origins to your domain in production.
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
	}))

	// Meta Group
	meta := app.Group("/", "System")

	
	meta.GET("/", handleRoot)         // Auto-summary: "Handle Root"
	meta.GET("/health", handleHealth) // Auto-summary: "Handle Health"

	// Users Group - Automatically tagged as "Users"
	users := app.Group("/users", "Users")

	users.GET("", listUsers, flux.Doc("List Users", "Returns all users. Supports optional search").
		Param("q", "query", "Search term", "string", false))

	users.GET("/:id", getUser, flux.Doc("Get User Details", "Returns a single user by numeric ID").
		Param("id", "path", "User ID", "integer", true))

	users.POST("", createUser, flux.Doc("Create User", "Creates a new user record").
		RequestBody("User payload", CreateUserRequest{}))

	users.PUT("/:id", updateUser, flux.Doc("Update User", "Modifies an existing user").
		Param("id", "path", "User ID", "integer", true))

	users.DELETE("/:id", deleteUser, flux.Doc("Delete User", "Removes a user record").
		Param("id", "path", "User ID", "integer", true))

	if err := app.Start(":8000"); err != nil {
		panic(err)
	}
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleRoot(c *flux.Context) error {
	return c.JSON(200, flux.Map{
		"service": "flux-basic-example",
		"version": "1.0.0",
		"docs":    "http://localhost:8000/docs",
		"health":  "http://localhost:8000/health",
	})
}

func handleHealth(c *flux.Context) error {
	return c.JSON(200, flux.Map{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func listUsers(c *flux.Context) error {
	// Optional full-text search
	if q := c.Query("q"); q != "" {
		results := store.search(q)
		return c.JSON(200, flux.Map{
			"users": results,
			"count": len(results),
			"query": q,
		})
	}

	users := store.list()
	return c.JSON(200, flux.Map{
		"users": users,
		"count": len(users),
	})
}

func getUser(c *flux.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return flux.NewHTTPError(400, "Invalid user ID — must be a positive integer")
	}

	user, ok := store.get(id)
	if !ok {
		return flux.NewHTTPError(404, "User not found")
	}

	return c.JSON(200, user)
}

func createUser(c *flux.Context) error {
	var req CreateUserRequest
	if err := c.BindJSON(&req); err != nil {
		return err // BindJSON already returns a structured HTTPError
	}

	user := store.create(req)
	return c.JSON(201, user)
}

func updateUser(c *flux.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return flux.NewHTTPError(400, "Invalid user ID — must be a positive integer")
	}

	var req UpdateUserRequest
	if err := c.BindJSON(&req); err != nil {
		return err
	}

	user, ok := store.update(id, req)
	if !ok {
		return flux.NewHTTPError(404, "User not found")
	}

	return c.JSON(200, user)
}

func deleteUser(c *flux.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return flux.NewHTTPError(400, "Invalid user ID — must be a positive integer")
	}

	if !store.delete(id) {
		return flux.NewHTTPError(404, "User not found")
	}

	return c.NoContent() // 204 — no body
}
