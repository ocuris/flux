package main

import (
	"fmt"
	"net/http"

	"github.com/ocuris/flux"
)

type CommandRequest struct {
	Level int    `json:"level"`
	Force bool   `json:"force"`
	Label string `json:"label"`
}

func main() {
	app := flux.New(flux.Config{Title: "Deep Nesting Test"})

	// Nested Groups Scenario
	// Route: POST /api/v1/orgs/:orgID/projects/:projectID/files/*
	v1 := app.Group("/api/v1")
	orgs := v1.Group("/orgs/:orgID")
	projects := orgs.Group("/projects/:projectID")

	projects.POST("/files/*", func(c *flux.Context) error {
		// 1. Capture Params from different levels of the tree
		orgID := c.Param("orgID")
		projectID := c.Param("projectID")

		// 2. Capture Wildcard (captured as "*")
		// Note: we also allow capturing via the name if preferred
		filepath := c.Param("*")

		// 3. Deserialize Payload
		var req CommandRequest
		if err := c.BindJSON(&req); err != nil {
			return err
		}

		// 4. Serialize Deep Response
		return c.JSON(http.StatusOK, flux.Map{
			"status": "success",
			"hierarchy": flux.Map{
				"org":     orgID,
				"project": projectID,
				"path":    filepath,
			},
			"payload_received": req,
		})
	})

	fmt.Println("🚀 Starting deep-nesting test on :8009")
	app.Start(":8009")
}
