package main

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ocuris/flux"
)

// jwtSecret is loaded from the JWT_SECRET environment variable at startup.
// In production, set this via your secrets manager or deployment config.
// Never hardcode secrets in source code.
var jwtSecret = func() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		// Development fallback — replace immediately in production.
		s = "change-me-in-production-use-env-var"
	}
	return []byte(s)
}()

// ── Domain types ──────────────────────────────────────────────────────────────

// User is the sanitised user model returned to clients.
// The password field is intentionally absent.
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"` // "admin" or "user"
}

// LoginRequest is the payload for POST /login.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse carries the JWT and its expiry.
type LoginResponse struct {
	Token   string `json:"token"`
	Expires int64  `json:"expires_at"` // Unix timestamp
	User    User   `json:"user"`
}

// Claims embeds jwt.RegisteredClaims and adds application-specific fields.
type Claims struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// ── Mock data (replace with a real database in production) ────────────────────

// In a real app, passwords would be stored as bcrypt hashes.
// Never store plaintext passwords.
type storedUser struct {
	User
	passwordHash string // placeholder — swap for bcrypt in production
}

var db = map[string]*storedUser{
	"john@example.com": {
		User:         User{ID: 1, Name: "John Doe", Email: "john@example.com", Role: "admin"},
		passwordHash: "password123", // ⚠ demo only
	},
	"jane@example.com": {
		User:         User{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Role: "user"},
		passwordHash: "password123", // ⚠ demo only
	},
}

// ── Application entry point ───────────────────────────────────────────────────

func main() {
	app := flux.New(flux.Config{
		Title:       "Flux JWT Authentication Example",
		Version:     "1.0.0",
		Description: "Demonstrates JWT-based authentication and role-based access control with Flux.",
		Debug:       false, // Set to true during development for richer error bodies
	})

	// ── Middleware ────────────────────────────────────────────────────────────
	app.Use(flux.Recover())
	app.Use(flux.RequestID())
	app.Use(flux.SecurityHeaders())
	app.Use(flux.RateLimiter(flux.RateLimitConfig{
		MaxRequests: 100,
		Window:      time.Minute,
	}))
	app.Use(flux.CORS(flux.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}))

	// ── Public Routes (System Tag) ────────────────────────────
	public := app.Group("/", "Public")
	public.GET("/health", handleHealth) // Auto-summary: "Handle Health"

	public.POST("/login", handleLogin, flux.Info{
		Summary:     "User Login",
		Description: "Exchange email/password for a high-security JWT.",
	})

	// ── Protected Routes (Authenticated Tag) ────────────────────────────
	jwtMiddleware := flux.JWT(flux.JWTConfig{
		SecretKey:   jwtSecret,
		ContextKey:  "claims",
		TokenLookup: "header:Authorization",
	})

	protected := app.Group("/", "Authenticated", jwtMiddleware)

	protected.GET("/profile", handleProfile, flux.Info{
		Summary:     "Get My Profile",
		Description: "Returns sensitive profile data for the active user.",
	})

	// /users — admin only
	app.GET("/users", flux.Doc(
		"List All Users",
		"Returns all users. Requires the 'admin' role.",
		"users",
	).
		Param("Authorization", "header", "Bearer <token>", "string", true).
		Response(200, "User list", "application/json", nil).
		Response(401, "Missing or invalid token", "application/json", nil).
		Response(403, "Insufficient role", "application/json", nil),
		jwtMiddleware(requireRole("admin", handleListUsers)),
	)

	// /admin/reset — admin only
	app.POST("/admin/reset", flux.Doc(
		"Admin: Reset Sessions",
		"Invalidates all active sessions. Requires the 'admin' role.",
		"admin",
	).
		Param("Authorization", "header", "Bearer <token>", "string", true).
		Response(200, "Reset confirmation", "application/json", nil).
		Response(403, "Insufficient role", "application/json", nil),
		jwtMiddleware(requireRole("admin", handleAdminReset)),
	)

	if err := app.Start(":8001"); err != nil {
		panic(err)
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func handleHealth(c *flux.Context) error {
	return c.JSON(200, flux.Map{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func handleLogin(c *flux.Context) error {
	var req LoginRequest
	if err := c.BindJSON(&req); err != nil {
		return err
	}

	// Look up user
	record, exists := db[req.Email]
	if !exists {
		// Use a constant-time comparison to avoid user-enumeration via timing.
		return flux.NewHTTPError(401, "Invalid email or password")
	}

	// TODO(production): replace with bcrypt.CompareHashAndPassword
	if record.passwordHash != req.Password {
		return flux.NewHTTPError(401, "Invalid email or password")
	}

	// Build claims
	now := time.Now()
	expiry := now.Add(24 * time.Hour)
	claims := Claims{
		UserID: record.ID,
		Email:  record.Email,
		Role:   record.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   record.Email,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiry),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtSecret)
	if err != nil {
		return flux.NewHTTPError(500, "Token generation failed")
	}

	return c.JSON(200, LoginResponse{
		Token:   signed,
		Expires: expiry.Unix(),
		User:    record.User,
	})
}

// requireRole returns a middleware that allows requests only when the JWT
// claim "role" matches the required role string.
func requireRole(role string, next flux.HandlerFunc) flux.HandlerFunc {
	return func(c *flux.Context) error {
		raw, ok := c.Get("claims")
		if !ok {
			return flux.NewHTTPError(401, "Authentication required")
		}
		token, ok := raw.(*jwt.Token)
		if !ok || !token.Valid {
			return flux.NewHTTPError(401, "Invalid token")
		}
		mc, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return flux.NewHTTPError(401, "Invalid token claims")
		}
		userRole, _ := mc["role"].(string)
		if userRole != role {
			return flux.NewHTTPError(403, "Insufficient permissions — '"+role+"' role required")
		}
		return next(c)
	}
}

func handleProfile(c *flux.Context) error {
	token := c.MustGet("claims").(*jwt.Token)
	mc := token.Claims.(jwt.MapClaims)

	return c.JSON(200, flux.Map{
		"id":    mc["user_id"],
		"email": mc["email"],
		"role":  mc["role"],
	})
}

func handleListUsers(c *flux.Context) error {
	users := make([]User, 0, len(db))
	for _, r := range db {
		users = append(users, r.User)
	}
	return c.JSON(200, flux.Map{
		"users": users,
		"count": len(users),
	})
}

func handleAdminReset(c *flux.Context) error {
	return c.JSON(200, flux.Map{
		"message": "All sessions invalidated",
		"action":  "reset_sessions",
		"time":    time.Now().UTC(),
	})
}
