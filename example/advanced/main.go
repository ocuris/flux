package main

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ocuris/flux"
)

// ── Structured error types ────────────────────────────────────────────────────

// AppError is a machine-readable, structured error response body.
// Clients should use Code to branch logic and Message for display.
type AppError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Status    int            `json:"status"`
	Timestamp time.Time      `json:"timestamp"`
	RequestID string         `json:"request_id,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

// Application-level error codes (use instead of raw HTTP status text).
const (
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeNotFound         = "RESOURCE_NOT_FOUND"
	CodeConflict         = "RESOURCE_CONFLICT"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeBadRequest       = "BAD_REQUEST"
	CodeInternalError    = "INTERNAL_ERROR"
)

// sendError writes an AppError JSON body and returns nil so Flux does
// not try to run its own error handler.
func sendError(c *flux.Context, code, message string, status int, details map[string]any) error {
	rid := ""
	if id, ok := c.Get("request_id"); ok {
		rid, _ = id.(string)
	}
	// Fall back to the X-Request-ID header if not in context store.
	if rid == "" {
		rid = c.Header("X-Request-ID")
	}
	return c.JSON(status, AppError{
		Code:      code,
		Message:   message,
		Status:    status,
		Timestamp: time.Now().UTC(),
		RequestID: rid,
		Details:   details,
	})
}

// ── Domain types ──────────────────────────────────────────────────────────────

// Article is the blog-post domain model.
type Article struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Author    string    `json:"author"`
	Status    string    `json:"status"` // "draft" | "published" | "archived"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateArticleRequest is the payload for POST /articles.
type CreateArticleRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Author  string `json:"author"`
}

// UpdateArticleRequest is the payload for PUT /articles/:id.
type UpdateArticleRequest struct {
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
}

// ── Thread-safe article store ─────────────────────────────────────────────────

type articleStore struct {
	mu       sync.RWMutex
	articles map[int]*Article
	nextID   int
}

func newArticleStore() *articleStore {
	s := &articleStore{articles: make(map[int]*Article), nextID: 2}
	s.articles[1] = &Article{
		ID:        1,
		Title:     "Getting Started with Flux",
		Content:   "Flux is a high-performance Go web framework designed for simplicity and production readiness...",
		Author:    "Jane Smith",
		Status:    "published",
		CreatedAt: time.Now().UTC().Add(-48 * time.Hour),
		UpdatedAt: time.Now().UTC().Add(-24 * time.Hour),
	}
	return s
}

func (s *articleStore) list() []*Article {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Article, 0, len(s.articles))
	for _, a := range s.articles {
		out = append(out, a)
	}
	return out
}

func (s *articleStore) get(id int) (*Article, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.articles[id]
	return a, ok
}

func (s *articleStore) create(req CreateArticleRequest) *Article {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	a := &Article{
		ID:        s.nextID,
		Title:     strings.TrimSpace(req.Title),
		Content:   strings.TrimSpace(req.Content),
		Author:    strings.TrimSpace(req.Author),
		Status:    "draft",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.articles[s.nextID] = a
	s.nextID++
	return a
}

func (s *articleStore) update(id int, req UpdateArticleRequest) (*Article, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.articles[id]
	if !ok {
		return nil, false
	}
	if req.Title != "" {
		a.Title = strings.TrimSpace(req.Title)
	}
	if req.Content != "" {
		a.Content = strings.TrimSpace(req.Content)
	}
	a.UpdatedAt = time.Now().UTC()
	return a, true
}

var store = newArticleStore()

// ── Application entry point ───────────────────────────────────────────────────

func main() {
	app := flux.New(flux.Config{
		Title:       "Flux Advanced Error Handling Example",
		Version:     "1.0.0",
		Description: "Demonstrates machine-readable structured error responses, conflict detection, and panic recovery.",
		Debug:       false,
	})

	app.Use(flux.Recover())
	app.Use(flux.RequestID())
	app.Use(flux.SecurityHeaders())

	// ── Article CRUD ──────────────────────────────────────────────────────────

	// Articles Group
	art := app.Group("/articles", "Articles")

	art.GET("", listArticles, flux.Info{
		Summary:     "List Articles",
		Description: "Returns all published articles.",
	})

	art.GET("/:id", getArticle, flux.Info{
		Summary:     "Get Article",
		Description: "Returns a single article with 404 error handling.",
	}.Param("id", "path", "Article ID", "integer", true))

	art.POST("", createArticle, flux.Doc("Create Article", "Publishes a new article").
		RequestBody("Publication payload", CreateArticleRequest{}))

	art.PUT("/:id", updateArticle, flux.Doc("Update Article", "Modifies title/content").
		Param("id", "path", "Article ID", "integer", true))

	// Demo Group (illustrate error shapes)
	demo := app.Group("/demo", "Demo Examples")

	demo.GET("/validation", demoValidation, flux.Info{Summary: "Trigger Validation Error"})
	demo.GET("/not-found", demoNotFound, flux.Info{Summary: "Trigger Not Found"})
	demo.GET("/conflict", demoConflict, flux.Info{Summary: "Trigger Conflict"})
	demo.GET("/panic", demoPanic, flux.Info{Summary: "Trigger Panic Recovery"})

	if err := app.Start(":8004"); err != nil {
		panic(err)
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func listArticles(c *flux.Context) error {
	articles := store.list()
	return c.JSON(200, flux.Map{
		"articles": articles,
		"count":    len(articles),
	})
}

func getArticle(c *flux.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return sendError(c, CodeBadRequest, "Invalid article ID — must be a positive integer", 400,
			map[string]any{"field": "id"})
	}

	article, ok := store.get(id)
	if !ok {
		return sendError(c, CodeNotFound, "Article not found", 404,
			map[string]any{"id": id})
	}

	return c.JSON(200, article)
}

func createArticle(c *flux.Context) error {
	var req CreateArticleRequest
	if err := c.BindJSON(&req); err != nil {
		return sendError(c, CodeBadRequest, "Invalid JSON payload", 400, nil)
	}

	details := make(map[string]any)

	if strings.TrimSpace(req.Title) == "" {
		details["title"] = "Title is required"
	} else if len(req.Title) < 5 || len(req.Title) > 200 {
		details["title"] = "Title must be 5–200 characters"
	}
	if strings.TrimSpace(req.Content) == "" {
		details["content"] = "Content is required"
	} else if len(req.Content) < 10 {
		details["content"] = "Content must be at least 10 characters"
	}
	if strings.TrimSpace(req.Author) == "" {
		details["author"] = "Author is required"
	}

	if len(details) > 0 {
		return sendError(c, CodeValidationFailed, "Request validation failed", 400, details)
	}

	article := store.create(req)
	return c.JSON(201, article)
}

func updateArticle(c *flux.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return sendError(c, CodeBadRequest, "Invalid article ID", 400, nil)
	}

	article, ok := store.get(id)
	if !ok {
		return sendError(c, CodeNotFound, "Article not found", 404,
			map[string]any{"id": id})
	}

	// Archived articles (status="archived") cannot be edited.
	if article.Status == "archived" {
		return sendError(c, CodeConflict, "Cannot update an archived article", 409,
			map[string]any{
				"reason":     "Article status is 'archived'",
				"article_id": id,
			})
	}

	var req UpdateArticleRequest
	if err := c.BindJSON(&req); err != nil {
		return sendError(c, CodeBadRequest, "Invalid JSON payload", 400, nil)
	}

	updated, _ := store.update(id, req) // ok is guaranteed since we checked above
	return c.JSON(200, updated)
}

// ── Demo handlers ─────────────────────────────────────────────────────────────

func demoValidation(c *flux.Context) error {
	return sendError(c, CodeValidationFailed, "Request validation failed", 400,
		map[string]any{
			"email":    "Invalid email format",
			"password": "Password must be at least 8 characters",
			"age":      "Age must be between 18 and 120",
		})
}

func demoNotFound(c *flux.Context) error {
	return sendError(c, CodeNotFound, "The requested resource does not exist", 404,
		map[string]any{"path": c.Path()})
}

func demoConflict(c *flux.Context) error {
	return sendError(c, CodeConflict, "A conflict occurred while processing the request", 409,
		map[string]any{
			"reason":     "Email address already registered",
			"suggestion": "Visit /auth/forgot-password to reset your credentials",
		})
}

func demoPanic(c *flux.Context) error {
	panic("intentional panic — demonstrating Recover() middleware")
}
