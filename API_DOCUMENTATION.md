# Flux API Documentation Guide

How to document your Flux API using the built-in `DocBuilder` and auto-generated OpenAPI 3.0 spec.

> **See also:** [README.md](./README.md) for the full framework reference.

---

## What You Get For Free

Add `Doc()` to any route and Flux automatically:

- Generates an **OpenAPI 3.0** spec at `GET /openapi.json`
- Serves **Swagger UI** at `GET /docs`

No extra setup required.

---

## 5-Minute Quick Start

**Before (undocumented):**

```go
app.GET("/users/:id", getUser)
```

**After (documented):**

```go
app.GET("/users/:id",
    flux.Doc("Get User", "Retrieve a user by their numeric ID", "users").
        Param("id", "path", "User ID", "integer", true).
        Response(200, "User found",  "application/json", nil).
        Response(404, "Not found",   "application/json", nil),
    getUser,
)
```

- Interactive docs: `http://localhost:8000/docs`
- Raw spec:         `http://localhost:8000/openapi.json`

---

## `DocBuilder` API Reference

### Creating the builder

```go
flux.Doc(summary, description string, tags ...string) *DocBuilder
```

### Chained methods

| Method | Signature | Description |
|---|---|---|
| `.Summary()` | `(s string) *DocBuilder` | Override the summary after creation |
| `.Description()` | `(desc string) *DocBuilder` | Override the description |
| `.Tags()` | `(tags ...string) *DocBuilder` | Set the grouping tags |
| `.Param()` | `(name, in, desc, schemaType string, required bool) *DocBuilder` | Add a parameter |
| `.ParamWithExample()` | `(name, in, desc, schemaType string, required bool, example interface{}) *DocBuilder` | Parameter with an example value |
| `.RequestBody()` | `(desc string, example interface{}) *DocBuilder` | Document the request body |
| `.Response()` | `(code int, desc, contentType string, example interface{}) *DocBuilder` | Document a response |
| `.Security()` | `(scheme string) *DocBuilder` | Add a security requirement |
| `.OperationID()` | `(id string) *DocBuilder` | Set the OpenAPI `operationId` |
| `.Deprecated()` | `(b bool) *DocBuilder` | Mark the endpoint as deprecated |
| `.Meta()` | `(key string, value interface{}) *DocBuilder` | Attach arbitrary metadata |

---

## Parameter Locations

The second argument to `.Param()` is the location (`in`):

| Value | Example |
|---|---|
| `"path"` | `/users/:id` — required by definition |
| `"query"` | `GET /users?page=1&limit=20` |
| `"header"` | `Authorization: Bearer <token>` |
| `"cookie"` | `Cookie: session=abc123` |

---

## Schema Types

The fourth argument to `.Param()` is the OpenAPI primitive type:

| Type | Use for |
|---|---|
| `"string"` | Text values |
| `"integer"` | Whole numbers |
| `"number"` | Floating-point numbers |
| `"boolean"` | `true` / `false` |
| `"array"` | Lists of items |
| `"object"` | Nested JSON objects |

---

## Common Patterns

### GET — List with query params

```go
app.GET("/users",
    flux.Doc("List Users", "Returns a paginated user list", "users").
        Param("page",  "query", "Page number (default 1)",             "integer", false).
        Param("limit", "query", "Items per page, max 100 (default 20)", "integer", false).
        Response(200, "User array", "application/json", nil),
    listUsers,
)
```

### GET — Single resource by ID

```go
app.GET("/users/:id",
    flux.Doc("Get User", "Returns a user by numeric ID", "users").
        Param("id", "path", "User ID", "integer", true).
        Response(200, "User object",  "application/json", nil).
        Response(404, "Not found",    "application/json", nil),
    getUser,
)
```

### POST — Create

```go
app.POST("/users",
    flux.Doc("Create User", "Creates a new user after validation", "users").
        RequestBody("User creation payload", CreateUserRequest{
            Name: "Alice", Email: "alice@example.com", Age: 28,
        }).
        Response(201, "Created user",   "application/json", nil).
        Response(400, "Invalid payload","application/json", nil),
    createUser,
)
```

### PUT — Update

```go
app.PUT("/users/:id",
    flux.Doc("Update User", "Partially updates a user", "users").
        Param("id", "path", "User ID", "integer", true).
        RequestBody("Fields to update", UpdateUserRequest{Name: "Alice Updated"}).
        Response(200, "Updated user", "application/json", nil).
        Response(404, "Not found",    "application/json", nil),
    updateUser,
)
```

### DELETE

```go
app.DELETE("/users/:id",
    flux.Doc("Delete User", "Permanently removes a user", "users").
        Param("id", "path", "User ID", "integer", true).
        Response(204, "Deleted — no body", "application/json", nil).
        Response(404, "Not found",         "application/json", nil),
    deleteUser,
)
```

### Protected endpoint (JWT)

```go
app.GET("/admin/stats",
    flux.Doc("Admin Stats", "Requires 'admin' role", "admin").
        Param("Authorization", "header", "Bearer <token>", "string", true).
        Security("bearerAuth").
        Response(200, "Stats",       "application/json", nil).
        Response(401, "Unauthorized","application/json", nil).
        Response(403, "Forbidden",   "application/json", nil),
    jwtMiddleware(requireAdmin(getStats)),
)
```

### Multiple tags

```go
app.GET("/users/:id/orders",
    flux.Doc("Get User Orders", "List all orders for a user", "users", "orders").
        Param("id", "path", "User ID", "integer", true).
        Response(200, "Order list", "application/json", nil),
    getUserOrders,
)
```

### Deprecated endpoint

```go
app.GET("/v1/users",
    flux.Doc("List Users (v1)", "Deprecated — use /v2/users", "users").
        Deprecated(true).
        Response(200, "User list", "application/json", nil),
    listUsersV1,
)
```

---

## Grouping with `app.Group()`

Routes registered on a `Group` inherit the prefix and group-level middleware, and their documentation is included automatically in the OpenAPI spec.

```go
v1 := app.Group("/api/v1")

v1.GET("/users", flux.Doc("List Users", "V1 user list", "users-v1"), listUsers)
v1.GET("/users/:id", flux.Doc("Get User", "V1 get user", "users-v1").
    Param("id", "path", "User ID", "integer", true), getUser)
```

---

## Best Practices

### ✅ Do

- Document **every** public-facing endpoint
- Include both success **and** error responses (400, 401, 403, 404, 500)
- Use **consistent HTTP status codes** (201 for creation, 204 for delete, etc.)
- Group related endpoints with **the same tag**
- Provide **realistic examples** in `RequestBody` and `Response`
- Set `.OperationID()` for predictable SDK generation

### ❌ Don't

- Leave endpoints without a `Doc()` annotation (they appear in the spec with minimal info)
- Use ambiguous or duplicate summary strings
- Forget error response documentation
- Use `"object"` as the content type — use `"application/json"`

---

## Testing Your Docs

```bash
# 1. Start your server
go run main.go

# 2. Open the Swagger UI
open http://localhost:8000/docs

# 3. Download and validate the spec
curl http://localhost:8000/openapi.json > spec.json

# 4. Validate against the OpenAPI spec (requires npx)
npx @redocly/cli lint spec.json

# 5. Generate client SDKs
openapi-generator generate \
    -i spec.json \
    -g typescript-fetch \
    -o ./generated/client
```
