package main

import (
	"fmt"
	"net/http"

	"github.com/ocuris/flux"
)

// User represents a simple data model for the example.
type User struct {
	ID    int    `json:"id" validate:"required"`
	Name  string `json:"name" validate:"required,min=2"`
	Email string `json:"email" validate:"required,email"`
}

func main() {
	// 1. Initialize Flux with Metadata for OpenAPI/Scalar AI
	app := flux.New(flux.Config{
		Title:       "Flux Grand Tour API",
		Description: "A comprehensive showcase of Flux framework capabilities",
		Version:     "1.0.0",
		Debug:       true,
	})

	// 2. Global Middleware (Security, Observability, Stability)
	app.Use(flux.Recover())         // Panic safety (Crucial since we removed built-in defer for speed)
	app.Use(flux.Logger())          // Request logging
	app.Use(flux.RequestID())       // Traceable Request IDs
	app.Use(flux.SecurityHeaders()) // Security best practices (XSS, Sniffing, etc.)
	app.Use(flux.CORS(flux.CORSConfig{
		AllowOrigins: []string{"*"},
	}))

	// -------------------------------------------------------------------------
	// 3. BASIC ROUTING & PARAMS
	// -------------------------------------------------------------------------

	// Simple GET with String response
	app.GET("/", func(c *flux.Context) error {
		return c.String(http.StatusOK, "Welcome to the Flux Grand Tour!")
	})

	// Path Parameters (:id)
	app.GET("/users/:id",
		flux.Doc("Get User", "Get a user by ID", "users").
			Param("id", "path", "The ID of the user", "integer", true),
		func(c *flux.Context) error {
			id := c.Param("id")
			return c.JSON(http.StatusOK, flux.Map{"id": id, "status": "found"})
		},
	)

	// Wildcards (*)
	app.GET("/static/*", func(c *flux.Context) error {
		filepath := c.Param("*")
		return c.String(http.StatusOK, fmt.Sprintf("Serving file: %s", filepath))
	})

	// -------------------------------------------------------------------------
	// 4. REQUEST DATA (JSON, Query, Headers, Cookies)
	// -------------------------------------------------------------------------

	// POST with JSON Body & Auto-Validation
	app.POST("/users",
		flux.Doc("Create User", "Validate and create a new user", "users").
			Response(http.StatusCreated, "User created successfully", "application/json", User{}),
		func(c *flux.Context) error {
			var u User
			if err := c.BindJSON(&u); err != nil {
				return err // Returns 400/422 with validation details automatically
			}
			return c.StatusJSON(http.StatusCreated, u)
		},
	)

	// Query Parameters
	app.GET("/search", func(c *flux.Context) error {
		query := c.Query("q")
		page := c.QueryDefault("page", "1")
		return c.JSON(http.StatusOK, flux.Map{
			"searching_for": query,
			"page":          page,
		})
	})

	// -------------------------------------------------------------------------
	// 5. RESPONSES (HTML, Redirect, Files, Context Store)
	// -------------------------------------------------------------------------

	// HTML Response
	app.GET("/hello", func(c *flux.Context) error {
		return c.HTML(http.StatusOK, "<h1>Hello from ⚡ Flux</h1>")
	})

	// Redirect
	app.GET("/old-page", func(c *flux.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/new-page")
	})

	// Per-Request Context Store (Passing data between middleware/handlers)
	app.GET("/secure",
		func(next flux.HandlerFunc) flux.HandlerFunc {
			return func(c *flux.Context) error {
				c.Set("user_role", "admin")
				return next(c)
			}
		},
		func(c *flux.Context) error {
			role := c.MustGet("user_role").(string)
			return c.String(http.StatusOK, "Secret area accessed as: "+role)
		},
	)

	// -------------------------------------------------------------------------
	// 6. MULTI-METHOD COMPATIBILITY (Any/Match)
	// -------------------------------------------------------------------------

	// Handle all methods on one path
	app.Any("/webhook", func(c *flux.Context) error {
		return c.String(http.StatusOK, "Received: "+c.Method())
	})

	// Handle specific subset of methods
	app.Match([]string{"GET", "DELETE"}, "/resource", func(c *flux.Context) error {
		return c.String(http.StatusOK, "Action allowed: "+c.Method())
	})

	// -------------------------------------------------------------------------
	// 7. GROUPS & SCOPED MIDDLEWARE
	// -------------------------------------------------------------------------

	// v1 API Group
	v1 := app.Group("/api/v1", "v1-api") // Second arg adds a tag to all routes
	{
		v1.Use(func(next flux.HandlerFunc) flux.HandlerFunc {
			fmt.Println("➡️ Entering API v1 Group")
			return next
		})

		v1.GET("/stats", func(c *flux.Context) error {
			return c.JSON(http.StatusOK, flux.Map{"cpu": 0.04, "uptime": "99.99%"})
		})

		// Nested Groups
		admin := v1.Group("/admin", "admin-only")
		admin.GET("/system", func(c *flux.Context) error {
			return c.NoContent()
		})
	}

	// -------------------------------------------------------------------------
	// 8. START THE SERVER
	// -------------------------------------------------------------------------
	fmt.Println("🚀 Starting Flux Grand Tour on :8080...")
	fmt.Println("📚 Open http://localhost:8080/docs for the Scalar AI Documentation")

	if err := app.Start(":8080"); err != nil {
		fmt.Printf("❌ Failed to start: %v\n", err)
	}
}
