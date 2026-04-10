# Flux Helpbook & Deployment Guide

This guide covers frequently asked questions, production preparation, and deployment specifics for the Flux framework.

## 1. Documentation Accessibility in Production

**Question:** Is the `/docs` page accessible after deployment, or do I need to copy HTML files onto my server?

**Answer:** It is **100% accessible automatically without copying any files.**
Flux natively uses the Go `//go:embed` library. When you run `go build` to compile your application into an executable binary, the underlying `new.html` layout (which powers the beautiful Scalar UI) is permanently baked into the binary code itself. 

* You deploy just **one single binary file** to your server.
* The `/docs` endpoint reads the embedded file from memory instantly. 

### Accessing the Docs Live
Once your server starts, simply visit your domain or IP address in the browser exactly exactly like you did locally.
* **If running on a VPS (like DigitalOcean):** `http://198.51.100.22/docs`
* **If hosted on a domain:** `https://api.yourdomain.com/docs`
*(Note: ensure your cloud provider firewall allows inbound traffic to your chosen port).*

---

## 2. Making Your API Publicly Accessible (or Not)

Whether your API can be accessed from the public internet depends entirely on the string you pass to `app.Start()`. This is called **binding**.

### To make it PUBLICLY accessible (Open to the World)
Bind to `:port` or `0.0.0.0:port`. By dropping the IP entirely, Go listens on *all* available network interfaces simultaneously.
```go
// Listens on every network interface. Anyone on the internet 
// with your server's IP address can reach the API.
app.Start(":8080")

// Also identical to:
app.Start("0.0.0.0:8080")
```

### To make it PRIVATE (Only local to the machine)
Bind specifically to the `127.0.0.1` (localhost) interface. This is **strongly recommended** if you are running Nginx, Apache, or a reverse proxy on the same server to handle HTTPS!
```go
// Only the server itself can reach the API. 
// External traffic from the internet is completely blocked out.
app.Start("127.0.0.1:8080")
```
*Why do this?* If you run a reverse proxy like Nginx that listens on port `443` (HTTPS) and proxies secure traffic back to your Go app on `:8080`, binding to `127.0.0.1:8080` guarantees that hackers cannot bypass Nginx to hit your unprotected port directly.

### Hiding Docs in Production
If you do *not* want public users seeing your Swagger/Scalar documentation in production, you can simply wrap the route registration in an environment variable check (though Flux registers them automatically, so a feature for conditionally disabling `InitOpenAPI` could be planned based on environment). For now, it is safe to expose them as they just serve the JSON schema and HTML.

## 2. Compiling for Production

When you are ready to deploy your backend to a Linux server (like AWS, DigitalOcean, or a Docker container) from your Mac or Windows machine, trace standard Go cross-compilation:

```bash
# Build a highly optimized Linux binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o api myapp/main.go
```
* `CGO_ENABLED=0`: Keeps the binary standalone so it doesn't rely on OS C libraries.
* `-ldflags="-s -w"`: Strips debug symbols to make the binary size much smaller.

## 3. Recommended Dockerfile

Because Flux requires no external assets and outputs a single embedded binary, the `Dockerfile` is extremely clean:

```dockerfile
# Step 1: Build the binary
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server main.go

# Step 2: Minimal Production Image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .

# Expose your chosen port
EXPOSE 8080
CMD ["./server"]
```

## 4. Best Practices & Scale

1. **Use `c.BindJSON()`:** Avoid manually unmarshalling HTTP bodies. Flux's binder is vastly safer, handles malformed JSON efficiently, and auto-returns precise 400 Bad Requests if the client sends garbage.
2. **Panic Recovery is Mandatory:** Never remove `app.Use(flux.Recover())`. A panic in a random Go routine handler without this middleware might crash the entire server instance instead of gracefully dropping that specific connection.
3. **Pointers for Context:** As seen in recent developments, ensure your handler signature is strictly `func(c *flux.Context) error`. Passing context tightly as a pointer minimizes memory allocation allowing Flux to serve thousands of concurrent connections efficiently.
