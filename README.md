# Flux — Go Web Framework

**A lightweight, production-ready web framework for Go.**

Flux gives you trie-based routing, automatic OpenAPI 3.0 docs, a JWT-ready middleware stack, and graceful shutdown — all on top of the standard `net/http` library with zero mandatory dependencies beyond the JWT library.

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green)](./LICENSE)

---

## Install

```bash
go get github.com/ocuris/flux
```

---

## Hello World

```go
package main

import "github.com/ocuris/flux"

func main() {
    app := flux.New(flux.Config{
        Title:   "My API",
        Version: "1.0.0",
    })

    app.Use(flux.Recover())
    app.Use(flux.RequestID())
    app.Use(flux.Logger())

    app.GET("/", func(c *flux.Context) error {
        return c.JSON(200, flux.Map{"message": "Hello, Flux!"})
    })

    app.Start(":8000") // blocks; graceful shutdown on SIGINT/SIGTERM
}
```

Visit `http://localhost:8000/docs` for the auto-generated Swagger UI.

---

## Core Concepts

### Configuration

```go
app := flux.New(flux.Config{
    Title:       "Payments API",     // shown in Swagger UI
    Version:     "2.1.0",
    Description: "Internal payment processing service",
    Debug:       false,              // true → include error detail in 500 responses
})
```

### Routing

```go
app.GET("/users",          listUsers)
app.POST("/users",         createUser)
app.GET("/users/:id",      getUser)      // named param
app.PUT("/users/:id",      updateUser)
app.DELETE("/users/:id",   deleteUser)
app.GET("/files/*",        serveFile)    // wildcard catch-all
```

### Route Groups

Groups share a prefix and optional group-scoped middleware. Global middleware still runs first.

```go
api := app.Group("/api/v1")
api.Use(authMiddleware)           // applies only inside this group

api.GET("/users",        listUsers)
api.POST("/users",       createUser)

admin := api.Group("/admin")      // nested: /api/v1/admin/...
admin.Use(requireAdmin)
admin.GET("/stats", getStats)
```

### Reading Requests

```go
func handler(c *flux.Context) error {
    id    := c.Param("id")                      // path param
    page  := c.QueryDefault("page", "1")        // query param with default
    token := c.Header("Authorization")          // request header
    sess  := c.Cookie("session_id")             // cookie

    var body MyRequest
    if err := c.BindJSON(&body); err != nil {   // decode + validate JSON body
        return err
    }

    val, ok := c.Get("user_id")                 // per-request store (set by middleware)
    _ = c.MustGet("user_id")                    // panics if missing

    return c.JSON(200, flux.Map{"ok": true})
}
```

### Writing Responses

```go
c.JSON(200, data)          // generic JSON
c.OK(data)                 // 200
c.Created(data)            // 201
c.Accepted(data)           // 202
c.NoContent()              // 204 — no body

// Errors — return these; the framework writes the JSON body
return flux.NewHTTPError(404, "User not found")
return flux.NewHTTPError(400, "Validation failed", details) // details → "details" field

// Convenience error methods
return c.BadRequest("Invalid payload")
return c.Unauthorized("Authentication required")
return c.Forbidden("Insufficient permissions")
return c.NotFound("Resource not found")
return c.InternalServerError("Something went wrong")

c.String(200, "plain text")
c.HTML(200, "<h1>Hello</h1>")
c.Redirect(301, "https://example.com")
c.SetHeader("X-Custom", "value")
c.SetCookie(&http.Cookie{Name: "session", Value: "abc", HttpOnly: true, Secure: true})
```

---

## Middleware

Register middleware with `app.Use()` **before** the routes it should apply to. Middleware composes into each handler at **registration time**, not per-request.

### Built-in Middleware

```go
app.Use(flux.Recover())           // catch panics → 500; log stack trace to stderr
app.Use(flux.RequestID())         // ensure X-Request-ID on every request/response
app.Use(flux.Logger())            // colourised: method | path | status | duration
app.Use(flux.SecurityHeaders())   // X-Frame-Options, X-Content-Type-Options, etc.

app.Use(flux.CORS(flux.CORSConfig{
    AllowOrigins:     []string{"https://yourapp.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    AllowCredentials: true,
}))

app.Use(flux.RateLimiter(flux.RateLimitConfig{
    MaxRequests: 100,
    Window:      time.Minute,
}))

app.Use(flux.Timeout(10 * time.Second)) // propagates via request context.Context

app.Use(flux.JWT(flux.JWTConfig{
    SecretKey:   []byte(os.Getenv("JWT_SECRET")),
    ContextKey:  "user",                   // default — token stored at c.Get("user")
    TokenLookup: "header:Authorization",   // default — supports "Bearer" prefix
}))
```

### Pre-routing Middleware

`Pre()` middleware runs before route matching — useful for path normalisation:

```go
app.Pre(canonicalisePathMiddleware)
```

### Custom Middleware

```go
func requestTimer(next flux.HandlerFunc) flux.HandlerFunc {
    return func(c *flux.Context) error {
        start := time.Now()
        err := next(c)
        c.SetHeader("X-Response-Time", time.Since(start).String())
        return err
    }
}
app.Use(requestTimer)
```

---

## JWT Authentication

```go
// 1. Protect routes (validates HS256 token, stores *jwt.Token in context)
jwtMW := flux.JWT(flux.JWTConfig{
    SecretKey: []byte(os.Getenv("JWT_SECRET")),
    Skipper: func(c *flux.Context) bool {
        // Skip auth for public endpoints
        return c.Path() == "/health" || c.Path() == "/login"
    },
})
app.Use(jwtMW)

// 2. Read claims inside a handler
func getProfile(c *flux.Context) error {
    claims := flux.JWTClaims(c)     // returns jwt.MapClaims; panics if no token
    email := claims["email"].(string)
    return c.JSON(200, flux.Map{"email": email})
}
```

Token sources supported via `TokenLookup`:

| Format | Example |
|---|---|
| `"header:Authorization"` (default) | `Authorization: Bearer <token>` |
| `"query:token"` | `GET /resource?token=<token>` |
| `"cookie:jwt"` | `Cookie: jwt=<token>` |

---

## Error Handling

```go
// Structured — framework writes `{"error":"...","details":...}` at the given status
return flux.NewHTTPError(422, "Validation failed", map[string]string{
    "email": "already registered",
})

// Untyped — framework writes 500; detail surfaced only when Config.Debug=true
return fmt.Errorf("db unavailable")
```

---

## Validation

### In `BindJSON` (struct tags)

```go
type CreateUserRequest struct {
    Name  string `json:"name"  validate:"required,min=3,max=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age"   validate:"gte=0,lte=120"`
}
```

Supported rules: `required` `email` `min=N` `max=N` `gte=N` `lte=N`

### Standalone helpers

```go
flux.IsValidEmail("u@example.com")  // bool
flux.IsValidURL("https://x.com")    // bool
flux.IsValidUUID("550e8400-...")    // bool
flux.Sanitize(input)                // html.EscapeString
flux.TrimInput(input)               // strings.TrimSpace
flux.ValidateRequired(s)            // non-empty after trim
flux.ValidateStringLength(s, 3, 50) // length in [min,max]
```

---

## OpenAPI 3.0 Documentation

Flux generates `/openapi.json` and serves Swagger UI at `/docs` automatically.  
Add `flux.Doc()` to any route to populate the spec:

```go
app.GET("/users/:id",
    flux.Doc(
        "Get User",
        "Retrieve a user by their numeric ID.",
        "users",
    ).
        Param("id", "path", "Numeric user ID", "integer", true).
        Response(200, "User object", "application/json", nil).
        Response(404, "Not found",   "application/json", nil),
    getUser,
)

app.POST("/users",
    flux.Doc("Create User", "Create a new user.", "users").
        RequestBody("User payload", CreateUserRequest{Name: "Alice"}).
        Response(201, "Created", "application/json", nil).
        Response(400, "Invalid", "application/json", nil),
    createUser,
)
```

**`DocBuilder` methods:**

| Method | Signature |
|---|---|
| `Doc()` | `Doc(summary, description string, tags ...string) *DocBuilder` |
| `.Param()` | `(name, in, desc, type string, required bool)` |
| `.ParamWithExample()` | `(name, in, desc, type string, required bool, example interface{})` |
| `.RequestBody()` | `(desc string, example interface{})` |
| `.Response()` | `(code int, desc, contentType string, example interface{})` |
| `.Security()` | `(scheme string)` |
| `.OperationID()` | `(id string)` |
| `.Deprecated()` | `(bool)` |
| `.Tags()` | `(tags ...string)` |
| `.Meta()` | `(key string, value interface{})` |

---

## Production Deployment

### Server Timeouts

Flux sets safe defaults. Override via `StartOption`:

```go
app.Start(":8000", func(s *http.Server) {
    s.ReadHeaderTimeout = 5 * time.Second
    s.ReadTimeout       = 30 * time.Second
    s.WriteTimeout      = 30 * time.Second
    s.IdleTimeout       = 120 * time.Second
})
```

### Graceful Shutdown

`app.Start()` blocks until `SIGINT` or `SIGTERM`, then drains in-flight requests with a 30-second timeout. No extra code required.

### Docker

```dockerfile
# Stage 1 — build
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server .

# Stage 2 — minimal runtime image
FROM scratch
COPY --from=builder /server /server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8000
ENTRYPOINT ["/server"]
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-api
  template:
    metadata:
      labels:
        app: my-api
    spec:
      containers:
        - name: my-api
          image: my-api:latest
          ports:
            - containerPort: 8000
          env:
            - name: ENV
              value: production
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: my-api-secrets
                  key: jwt_secret
          resources:
            requests:
              cpu: "100m"
              memory: "64Mi"
            limits:
              cpu: "500m"
              memory: "256Mi"
          livenessProbe:
            httpGet:
              path: /health
              port: 8000
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health
              port: 8000
            initialDelaySeconds: 3
            periodSeconds: 5
```

### Environment Variables

| Variable | Purpose | Default |
|---|---|---|
| `ENV` | Shown in startup banner | `development` |
| `JWT_SECRET` | JWT signing key (load in app) | — |

---

## Security Checklist

- [ ] Load `JWT_SECRET` and all secrets from env vars / secret manager — never hardcode
- [ ] Use `flux.Recover()` as the **first** middleware
- [ ] Set `Debug: false` in production (hides internal error detail)
- [ ] Apply `flux.SecurityHeaders()` on every response
- [ ] Restrict `CORS.AllowOrigins` to your actual domain(s)
- [ ] Apply `flux.RateLimiter()` to authentication endpoints at minimum
- [ ] Use `flux.Timeout()` to bound all handler execution
- [ ] Hash passwords with `bcrypt` — never store plaintext
- [ ] Set `Secure: true` and `HttpOnly: true` on session cookies
- [ ] Validate and sanitise all user input with `BindJSON` + struct tags

---

## Examples

| Directory | Port | Demonstrates |
|---|---|---|
| `example/basic` | `:8000` | CRUD, thread-safe store, DocBuilder, search |
| `example/auth` | `:8001` | JWT login, role-based access, env secret |
| `example/validation` | `:8002` | Field-level errors, partial updates, pagination |
| `example/high-throughput` | `:8003` | TTL cache, atomic metrics, cache invalidation |
| `example/advanced` | `:8004` | Structured `AppError`, conflict detection, panic demo |

```bash
# Run any example
cd example/basic && go run main.go
open http://localhost:8000/docs
```

See [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) for the complete `DocBuilder` reference.

---

## License

[MIT](./LICENSE)
