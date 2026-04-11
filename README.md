# ⚡️ Flux
**Ultra-High Performance • Zero Dependencies • Auto-Documentation**

[![Go Reference](https://pkg.go.dev/badge/github.com/ocuris/flux.svg)](https://pkg.go.dev/github.com/ocuris/flux)
[![Go Report Card](https://goreportcard.com/badge/github.com/ocuris/flux)](https://goreportcard.com/report/github.com/ocuris/flux)
[![Build Status](https://github.com/ocuris/flux/actions/workflows/ci.yml/badge.svg)](https://github.com/ocuris/flux/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://golang.org)

*Flux is a next-generation web framework for Go, designed for developers who refuse to compromise on speed or security. Built on a trie-based zero-allocation router, it delivers bare-metal performance with a developer experience that feels like magic.*

---

[Features](#-key-features) • [Quick Start](#-quick-start) • [Performance](#-performance-benchmarks) • [Documentation](#-automatic-openapi--scalar-ai) • [Deployment](#-production-readiness)

---

## 💎 Why Flux?

In an ecosystem of heavy frameworks, Flux stands out by being **lightweight yet powerful**.

| Feature | Flux | Gin / Echo | Standard Lib |
| :--- | :---: | :---: | :---: |
| **Zero Dependencies** | ✅ | ❌ | ✅ |
| **Zero-Allocation Router** | ✅ | ✅ | ❌ |
| **Built-in Middleware** | ✅ | ✅ | ❌ |
| **Auto-OpenAPI Docs** | ✅ | ❌ | ❌ |
| **Graceful Shutdown** | ✅ | ❌ | ❌ |
| **Type-Safe Validation** | ✅ | ✅ | ❌ |

> **"Flux is what happens when you combine the simplicity of Go's standard library with the power of a modern framework."**

---

## 🚀 Key Features

-   **⚡️ Elite Performance**: Trie-based routing with zero memory allocation during request execution.
-   **🛡 Zero Third-Party Dependencies**: No supply-chain bloat or security risks. Pure Go.
-   **📚 Automatic Documentation**: Native **Scalar AI** and OpenAPI 3.0 generation out of the box.
-   **🔐 Security by Default**: Built-in JWT, Rate Limiting, CORS, and Security Headers.
-   **📦 Developer-Centric**: Automatic graceful shutdowns, request-ID tracking, and structured error handling.
-   **🧪 Testing First**: Built with testability in mind, including an elite benchmarking suite.

---

## 🏁 Quick Start

### Installation

```bash
go get github.com/ocuris/flux
```

### The "Hello, Flux" Example

```go
package main

import "github.com/ocuris/flux"

func main() {
    // 1. Initialize the app with metadata
    app := flux.New(flux.Config{
        Title:   "Payments API",
        Version: "1.0.0",
    })

    // 2. Global Middleware
    app.Use(flux.Recover())
    app.Use(flux.Logger())

    // 3. Define Routes
    app.GET("/welcome/:name", func(c *flux.Context) error {
        name := c.Param("name")
        return c.JSON(200, flux.Map{"message": "Hello, " + name + "!"})
    })

    // 4. Start Server (automatic graceful shutdown)
    app.Start(":8000")
}
```

Visit `http://localhost:8000/docs` to see your API come alive with auto-generated documentation.

---

## 📊 Performance Benchmarks

Flux is engineered for ultra-high throughput. By eliminating mutex contention and minimizing the request-handling hot path, Flux delivers bare-metal performance that scales linearly with your hardware.

### 🏆 The Leaderboard (8-core Parallel Stress)

| Framework | Time per Request | Allocs/op | Speed Index |
| :--- | :--- | :---: | :--- |
| **⚡️ Flux** | **~4.4 ns** (🥇) | **0** | **1.0x (Reference)** |
| **Echo** | ~5.4 ns | 0 | 1.2x Slower |
| **Gin** | ~6.7 ns | 0 | 1.5x Slower |
| **Chi** | ~122.0 ns | 2 | 27.7x Slower |

### 📈 Scaling Analysis (1 to 8 Cores)
Flux features a **lock-free routing engine**, allowing it to scale almost perfectly as you add CPU cores.

| Cores | 1 | 2 | 4 | 8 |
| :--- | :---: | :---: | :---: | :---: |
| **Throughput (ns/op)** | 20.5 | 10.2 | 5.6 | **4.4** |

### 🧠 Why is Flux Faster?

For developers who care about the "How", Flux achieves these numbers through three core architectural decisions:

1.  **Atomic Static Routing**: Unlike frameworks that use a single shared Mutex or RWMutex for routing lookups, Flux uses method-specific `atomic.Pointer` maps. This enables $O(1)$ lock-free lookups for static routes, eliminating contention in high-concurrency environments.
2.  **Zero-Allocation Context Pooling**: Using a highly optimized `sync.Pool` with pre-allocated parameter slices (cap: 16), Flux ensures that standard requests never touch the heap.
3.  **Low-Latency Hot Path**: We removed `defer` from the core `ServeHTTP` loop and implemented early-exit middleware fast-paths. Every nanosecond saved in the engine is more CPU time for your business logic.

> [!NOTE]
> Benchmarks were conducted using the provided **benchmarks/Dockerfile** on an **Apple M1 Air (8-core)**. This ensures results are isolated and reproducible.

---

## 🛠 Core Concepts

### 🛤 Routing & Groups
Manage complex API structures with ease using nested groups and local middleware.

```go
v1 := app.Group("/api/v1")
v1.Use(authMiddleware)

v1.GET("/users", listUsers)
v1.POST("/users", createUser)

admin := v1.Group("/admin")
admin.Use(requireAdmin)
admin.GET("/stats", getStats)
```

### 📝 Reading & Writing
Flux provides a high-level API for handling requests and responses without losing control.

```go
func handler(c *flux.Context) error {
    id := c.Param("id")             // Path parameter
    var body MyRequest
    if err := c.BindJSON(&body); err != nil {
        return err                  // Automatic 400 with detail
    }
    return c.Created(body)          // 201 Created
}
```

### 🧱 Built-in Middleware
Everything you need for a production API is included:
- `flux.Recover()`: Catch panics gracefully.
- `flux.JWT()`: Professional HS256 authentication.
- `flux.RateLimiter()`: Protection against abuse.
- `flux.CORS()`: Secure cross-origin resource sharing.
- `flux.Timeout()`: Bounded handler execution.

---

## 📖 Automatic OpenAPI & Scalar AI

Flux automatically generates a complete OpenAPI 3.0 specification. It also includes the stunning **Scalar AI** documentation interface.


<details>
<summary><b>View Interactive Documentation details</b></summary>

Flux provides a powerful, interactive documentation UI via Scalar AI.
</details>

```go
app.GET("/users/:id",
    flux.Doc("Get User", "Retrieve a user by their ID", "users").
        Param("id", "path", "User ID", "integer", true).
        Response(200, "User object", "application/json", nil),
    getUser,
)
```

---

## 🛡 Production Readiness

### Graceful Shutdown
Flux handles `SIGINT` and `SIGTERM` automatically, draining in-flight requests before exiting.

### Dockerized Deployment
```dockerfile
FROM scratch
COPY --from=builder /server /server
EXPOSE 8000
ENTRYPOINT ["/server"]
```

### Professional Makefile
```bash
make test        # Run unit tests
make bench       # Run the full elite benchmark suite
make vuln        # Scan for security vulnerabilities
```

---

## 🤝 Contributing & Community

Join us in building the fastest web framework for Go. Check out the [CONTRIBUTING.md](./CONTRIBUTING.md) to get started.

---

## 📄 License

Flux is released under the **MIT License**. See [LICENSE](./LICENSE) for details.


Built with ❤️ for the Go ecosystem by [Ocuris](https://github.com/ocuris).
