# ⚡️ Flux
**Ultra-High Performance • Zero Dependencies • Auto-Documentation**

[![Go Reference](https://pkg.go.dev/badge/github.com/ocuris/flux.svg)](https://pkg.go.dev/github.com/ocuris/flux)
[![Go Report Card](https://goreportcard.com/badge/github.com/ocuris/flux)](https://goreportcard.com/report/github.com/ocuris/flux)
[![Build Status](https://github.com/ocuris/flux/actions/workflows/ci.yml/badge.svg)](https://github.com/ocuris/flux/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://golang.org)

*Flux is a next-generation web framework for Go, designed for developers who refuse to compromise on speed or security. Built on a trie-based zero-allocation router, it delivers bare-metal performance with a developer experience that feels like magic.*

---

[Features](#-key-features) • [Quick Start](#-quick-start) • [Performance](./PERFORMANCE.md) • [API Guide](./API.md) • [Deployment](./DEPLOYMENT.md)

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

-   **⚡️ Elite Performance**: Trie-based routing with **Dynamic Parameter Pre-scaling** for zero heap-allocations.
-   **🛡 Zero Dependencies**: No supply-chain bloat or security risks. Pure Go from the ground up.
-   **📚 Automatic Documentation**: Native **Scalar AI** and OpenAPI 3.0 generation (at `/docs`).
-   **🔐 Security by Default**: Built-in JWT, Rate Limiting, CORS, and Security Headers middleware.
-   **🔄 Hot Reloading**: Native CLI tool for an instant development experience with zero setup.
-   **🧪 Multi-Core Scalability**: Engineered for massive parallel throughput on modern ARM64 silicon.

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

## 💻 Development Mode

Flux includes a native CLI (**flx**) to make development effortless. It supports "hot reloading" which automatically rebuilds and restarts your app when you save changes.

### Installation (Homebrew)
```bash
brew tap ocuris/flux
brew install flx
```

### Installation (Go)
```bash
go install github.com/ocuris/flux/cmd/flux@latest
```

### Usage
Run your app with the `--reload` (or `-r`) flag to watch for changes:
```bash
flx --reload main.go
```

You can also enable it directly in your `Config`:
```go
app := flux.New(flux.Config{
    Reload: true,
})
```

---

## 📊 Performance Benchmarks

Flux is engineered for ultra-high throughput. By eliminating mutex contention and minimizing the request-handling hot path, Flux delivers bare-metal performance that scales linearly with your hardware.

### 🏆 Comprehensive Framework Comparison
All tests conducted on **Apple M1 (8-core)** with `GOGC=100`.

| Metric (lower is better) | ⚡️ Flux | Gin | Echo | Fiber |
| :--- | :--- | :--- | :--- | :--- |
| **8-Core Parallel Stress** | **3.9 ns** 🥇 | 5.2 ns | 4.8 ns | 2435 ns |
| **Deep Route (7 segments)** | **22.0 ns** 🥇 | 26.7 ns | 34.1 ns | 4070 ns |
| **Middleware (5 layers)** | **25.6 ns** 🥇 | 41.8 ns | 109.0 ns | 3881 ns |
| **Large Scale (100 routes)** | **27.9 ns** 🥇 | 37.9 ns | 36.9 ns | N/A |
| **JSON Response** | 1133 ns | 1089 ns | **1039 ns** | 7620 ns |
| **Path Params (:id)** | 53.7 ns | 33.2 ns | **30.0 ns** | 4000 ns |

### 📈 Scaling Analysis (1 to 8 Cores)
Flux features a **lock-free routing engine**, allowing it to scale almost perfectly as you add CPU cores.

| Cores | 1 | 2 | 4 | 8 |
| :--- | :---: | :---: | :---: | :---: |
| **Throughput (ns/op)** | 19.4 ns | 10.1 ns | 5.8 ns | **3.9 ns** |

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
