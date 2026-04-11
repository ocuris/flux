package main

import (
	"fmt"

	"github.com/ocuris/flux"
)

func main() {
	app := flux.New(flux.Config{Title: "Method Test API"})

	// Middleware to log the method being hit
	app.Use(func(next flux.HandlerFunc) flux.HandlerFunc {
		return func(c *flux.Context) error {
			fmt.Printf("📥 Received %s %s\n", c.Method(), c.Path())
			return next(c)
		}
	})

	// Standard methods
	app.GET("/test", func(c *flux.Context) error { return c.String(200, "GET OK") })
	app.POST("/test", func(c *flux.Context) error { return c.String(200, "POST OK") })

	// --- NEW: Data Handling Scenarios ---

	// 1. Path Parameters
	app.GET("/users/:id", func(c *flux.Context) error {
		id := c.Param("id")
		return c.JSON(200, flux.Map{"user_id": id, "action": "profile_view"})
	})

	// 2. Query Parameters
	app.GET("/search", func(c *flux.Context) error {
		query := c.Query("q")
		page := c.QueryDefault("page", "1")
		return c.JSON(200, flux.Map{
			"results_for": query,
			"page":        page,
			"filter":      "active",
		})
	})

	// 3. Request Body & Serialized Response
	type EchoRequest struct {
		Message string `json:"message" validate:"required"`
		Score   int    `json:"score"`
	}
	app.POST("/echo", func(c *flux.Context) error {
		var req EchoRequest
		if err := c.BindJSON(&req); err != nil {
			return err
		}
		// Return serialized struct
		return c.JSON(200, flux.Map{
			"received": req,
			"time":     "2026-04-11",
			"status":   "success",
		})
	})

	// --- End Data Handling ---

	app.PUT("/test", func(c *flux.Context) error { return c.String(200, "PUT OK") })
	app.DELETE("/test", func(c *flux.Context) error { return c.String(200, "DELETE OK") })
	app.PATCH("/test", func(c *flux.Context) error { return c.String(200, "PATCH OK") })
	app.OPTIONS("/test", func(c *flux.Context) error {
		c.SetHeader("Allow", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		return c.NoContent()
	})
	app.HEAD("/test", func(c *flux.Context) error {
		c.SetHeader("X-Flux-Head", "true")
		return c.NoContent()
	})

	// Any method
	app.Any("/any", func(c *flux.Context) error {
		return c.String(200, "Any OK: "+c.Method())
	})

	// Match subset
	app.Match([]string{"GET", "POST"}, "/match", func(c *flux.Context) error {
		return c.String(200, "Match OK: "+c.Method())
	})

	fmt.Println("🚀 Starting method-test on :8007")
	app.Start(":8007")
}
