package main

import (
	"fmt"
	"time"

	"github.com/ocuris/flux"
)

// User represents a user in the system
type User struct {
	ID        int    `json:"id" doc:"Unique user identifier" example:"123"`
	Name      string `json:"name" doc:"User's full name" example:"John Doe"`
	Email     string `json:"email" doc:"User's email address" example:"john@example.com"`
	Role      string `json:"role" doc:"User role" example:"admin"`
	Active    bool   `json:"active" doc:"Is user active"`
	CreatedAt string `json:"created_at" doc:"ISO 8601 creation timestamp"`
}

// CreateUserRequest for creating new users
type CreateUserRequest struct {
	Name     string `json:"name" doc:"User's full name" required:"true" example:"John Doe"`
	Email    string `json:"email" doc:"User's email" required:"true" example:"john@example.com"`
	Password string `json:"password" doc:"Account password" required:"true" minLength:"8"`
	Role     string `json:"role" doc:"User role (admin, user, guest)" required:"true" example:"user"`
}

// ErrorResponse for error handling
type ErrorResponse struct {
	Code    string                 `json:"code" doc:"Error code"`
	Message string                 `json:"message" doc:"Human-readable error message"`
	Details map[string]interface{} `json:"details,omitempty" doc:"Additional error context"`
}

var users = map[int]*User{
	1: {ID: 1, Name: "John Doe", Email: "john@example.com", Role: "admin", Active: true, CreatedAt: time.Now().Format(time.RFC3339)},
	2: {ID: 2, Name: "Jane Smith", Email: "jane@example.com", Role: "user", Active: true, CreatedAt: time.Now().Format(time.RFC3339)},
}
var nextID = 3

func main() {
	app := flux.New(flux.Config{
		Title:       "Documented User API",
		Version:     "2.0.0",
		Description: "Complete example showing how to document Flux APIs with metadata",
	})

	// Middleware
	app.Use(flux.Recover())
	app.Use(flux.RequestID())
	app.Use(flux.Logger())

	// =====================================================
	// DOCUMENTED ROUTES
	// =====================================================

	// GET /health - Health check (undocumented for brevity)
	app.GET("/health", func(c *flux.Context) error {
		return c.JSON(200, flux.Map{"status": "healthy"})
	})

	// GET /users - List all users
	app.GET("/users",
		flux.Doc(
			"List All Users",
			"Retrieve a paginated list of all users in the system",
			"users",
		).
			ParamWithExample("page", "query", "Page number", "integer", false, 1).
			ParamWithExample("limit", "query", "Results per page", "integer", false, 10).
			Response(200, "List of users", "", []User{}).
			Response(400, "Invalid query parameters", "", nil),
		listUsers,
	)

	// GET /users/:id - Get specific user
	app.GET("/users/:id",
		flux.Doc(
			"Get User by ID",
			"Retrieve a specific user by their unique ID",
			"users",
		).
			Param("id", "path", "User ID", "integer", true).
			Response(200, "User found", "", User{}).
			Response(404, "User not found", "", ErrorResponse{Code: "USER_NOT_FOUND"}).
			OperationID("getUserById"),
		getUser,
	)

	// POST /users - Create new user
	app.POST("/users",
		flux.Doc(
			"Create User",
			"Create a new user account with email and password",
			"users",
		).
			RequestBody("User creation data", CreateUserRequest{
				Name:     "John Doe",
				Email:    "john@example.com",
				Password: "securepass123",
				Role:     "user",
			}).
			Response(201, "User created successfully", "", User{}).
			Response(400, "Invalid request data", "", ErrorResponse{Code: "INVALID_INPUT"}).
			Response(409, "Email already exists", "", ErrorResponse{Code: "EMAIL_EXISTS"}).
			OperationID("createUser"),
		createUser,
	)

	// PUT /users/:id - Update user
	app.PUT("/users/:id",
		flux.Doc(
			"Update User",
			"Update an existing user's information",
			"users",
		).
			Param("id", "path", "User ID to update", "integer", true).
			RequestBody("Updated user data", CreateUserRequest{
				Name:     "Jane Doe",
				Email:    "jane@example.com",
				Password: "newpass123",
				Role:     "admin",
			}).
			Response(200, "User updated successfully", "", User{}).
			Response(400, "Invalid request data", "", ErrorResponse{}).
			Response(404, "User not found", "", ErrorResponse{}).
			Response(409, "Email already in use", "", ErrorResponse{}).
			OperationID("updateUser"),
		updateUser,
	)

	// DELETE /users/:id - Delete user
	app.DELETE("/users/:id",
		flux.Doc(
			"Delete User",
			"Permanently delete a user account",
			"users",
		).
			Param("id", "path", "User ID to delete", "integer", true).
			Response(204, "User deleted successfully", "", nil).
			Response(404, "User not found", "", ErrorResponse{}).
			OperationID("deleteUser").
			Deprecated(false),
		deleteUser,
	)

	// GET /users/:id/activity - Get user activity
	app.GET("/users/:id/activity",
		flux.Doc(
			"Get User Activity",
			"Retrieve recent activity log for a specific user",
			"users", "activity",
		).
			Param("id", "path", "User ID", "integer", true).
			ParamWithExample("limit", "query", "Number of recent activities", "integer", false, 20).
			Response(200, "Activity log", "", []flux.Map{}).
			Response(404, "User not found", "", ErrorResponse{}),
		getUserActivity,
	)

	// GET /stats/users - User statistics
	app.GET("/stats/users",
		flux.Doc(
			"User Statistics",
			"Get aggregate statistics about users in the system",
			"stats",
		).
			Response(200, "User statistics", "", flux.Map{}).
			OperationID("getUserStats"),
		getUserStats,
	)

	fmt.Println("📚 Documented API running on :8000")
	fmt.Println("📖 API Docs: http://localhost:8000/docs")
	fmt.Println("📄 OpenAPI Spec: http://localhost:8000/openapi.json")
	fmt.Println("")
	fmt.Println("Try these endpoints:")
	fmt.Println("  GET http://localhost:8000/users")
	fmt.Println("  GET http://localhost:8000/users/1")
	fmt.Println("  POST http://localhost:8000/users (with JSON body)")
	fmt.Println("  GET http://localhost:8000/openapi.json (view full spec)")

	if err := app.Start(":8000"); err != nil {
		panic(err)
	}
}

func listUsers(c *flux.Context) error {
	userList := make([]*User, 0, len(users))
	for _, u := range users {
		userList = append(userList, u)
	}
	return c.JSON(200, userList)
}

func getUser(c *flux.Context) error {
	id := c.Param("id")
	var userID int
	fmt.Sscanf(id, "%d", &userID)

	user, exists := users[userID]
	if !exists {
		return c.JSON(404, ErrorResponse{
			Code:    "USER_NOT_FOUND",
			Message: fmt.Sprintf("User with ID %d not found", userID),
		})
	}

	return c.JSON(200, user)
}

func createUser(c *flux.Context) error {
	var req CreateUserRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{
			Code:    "INVALID_INPUT",
			Message: "Invalid request body",
		})
	}

	// Check if email already exists
	for _, u := range users {
		if u.Email == req.Email {
			return c.JSON(409, ErrorResponse{
				Code:    "EMAIL_EXISTS",
				Message: "Email already registered",
			})
		}
	}

	user := &User{
		ID:        nextID,
		Name:      req.Name,
		Email:     req.Email,
		Role:      req.Role,
		Active:    true,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	users[nextID] = user
	nextID++

	return c.JSON(201, user)
}

func updateUser(c *flux.Context) error {
	id := c.Param("id")
	var userID int
	fmt.Sscanf(id, "%d", &userID)

	user, exists := users[userID]
	if !exists {
		return c.JSON(404, ErrorResponse{
			Code:    "USER_NOT_FOUND",
			Message: fmt.Sprintf("User with ID %d not found", userID),
		})
	}

	var req CreateUserRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, ErrorResponse{
			Code:    "INVALID_INPUT",
			Message: "Invalid request body",
		})
	}

	// Update user
	user.Name = req.Name
	user.Email = req.Email
	user.Role = req.Role

	return c.JSON(200, user)
}

func deleteUser(c *flux.Context) error {
	id := c.Param("id")
	var userID int
	fmt.Sscanf(id, "%d", &userID)

	_, exists := users[userID]
	if !exists {
		return c.JSON(404, ErrorResponse{
			Code:    "USER_NOT_FOUND",
			Message: fmt.Sprintf("User with ID %d not found", userID),
		})
	}

	delete(users, userID)
	return c.NoContent()
}

func getUserActivity(c *flux.Context) error {
	id := c.Param("id")
	var userID int
	fmt.Sscanf(id, "%d", &userID)

	_, exists := users[userID]
	if !exists {
		return c.JSON(404, ErrorResponse{
			Code:    "USER_NOT_FOUND",
			Message: "User not found",
		})
	}

	activities := []flux.Map{
		{"timestamp": time.Now().Format(time.RFC3339), "action": "login", "ip": "192.168.1.1"},
		{"timestamp": time.Now().Add(-1 * time.Hour).Format(time.RFC3339), "action": "viewed_profile", "ip": "192.168.1.1"},
		{"timestamp": time.Now().Add(-2 * time.Hour).Format(time.RFC3339), "action": "updated_settings", "ip": "192.168.1.2"},
	}

	return c.JSON(200, activities)
}

func getUserStats(c *flux.Context) error {
	return c.JSON(200, flux.Map{
		"total_users": len(users),
		"active_users": func() int {
			count := 0
			for _, u := range users {
				if u.Active {
					count++
				}
			}
			return count
		}(),
		"by_role": map[string]int{
			"admin": 1,
			"user":  1,
		},
	})
}
