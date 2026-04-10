package flux

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// Logger logs every request: method, path, status code, and elapsed time.
// It reads c.statusCode after the full handler chain (including error handling)
// has completed, so it always reflects the actual HTTP status sent to the client.
func Logger() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			start := time.Now()
			method := c.Method()
			path := c.Path()

			err := next(c)

			// c.statusCode is guaranteed to be set by the time we reach here
			// because ServeHTTP calls handleError (which calls c.JSON) before
			// the middleware chain unwinds.
			status := c.statusCode
			if status == 0 {
				status = http.StatusOK
			}
			logRequest(method, path, status, time.Since(start))
			return err
		}
	}
}

// Recover catches any panic in downstream handlers, logs the panic value and
// full stack trace to stderr, and returns a 500 Internal Server Error.
func Recover() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					// Always emit the panic value + stack trace so it can be
					// investigated, even in production.
					fmt.Fprintf(os.Stderr, "flux: PANIC recovered: %v\n%s\n",
						r, debug.Stack())
					_ = c.JSON(http.StatusInternalServerError, Map{
						"error": "internal server error",
					})
				}
			}()
			return next(c)
		}
	}
}

// SecurityHeaders adds common defensive HTTP headers to every response.
func SecurityHeaders() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.SetHeader("X-Content-Type-Options", "nosniff")
			c.SetHeader("X-Frame-Options", "DENY")
			c.SetHeader("X-XSS-Protection", "1; mode=block")
			c.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")
			c.SetHeader("Cache-Control", "no-cache, no-store, must-revalidate")
			c.SetHeader("Pragma", "no-cache")
			c.SetHeader("Expires", "0")
			return next(c)
		}
	}
}

// RequestID ensures every request has a unique X-Request-ID header.
// It propagates any ID supplied by the client; otherwise a cryptographically
// random 32-hex-character ID is generated.
func RequestID() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			id := c.Request.Header.Get("X-Request-ID")
			if id == "" {
				b := make([]byte, 16)
				if _, err := rand.Read(b); err != nil {
					// Extremely unlikely: fall back to timestamp
					id = fmt.Sprintf("%d", time.Now().UnixNano())
				} else {
					id = hex.EncodeToString(b)
				}
			}
			c.SetHeader("X-Request-ID", id)
			return next(c)
		}
	}
}

// CORSConfig is the configuration for the CORS middleware.
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int
}

// CORS adds Cross-Origin Resource Sharing headers and handles OPTIONS preflight.
func CORS(config CORSConfig) MiddlewareFunc {
	if config.AllowOrigins == nil {
		config.AllowOrigins = []string{"*"}
	}
	if config.AllowMethods == nil {
		config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	}
	if config.AllowHeaders == nil {
		config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			origin := c.Header("Origin")

			allowed := false
			for _, o := range config.AllowOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				if origin == "" {
					c.SetHeader("Access-Control-Allow-Origin", "*")
				} else {
					c.SetHeader("Access-Control-Allow-Origin", origin)
				}
			}

			c.SetHeader("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			c.SetHeader("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))

			if config.AllowCredentials {
				c.SetHeader("Access-Control-Allow-Credentials", "true")
			}
			if config.MaxAge > 0 {
				c.SetHeader("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			}

			// Short-circuit preflight
			if c.Method() == http.MethodOptions {
				return c.NoContent()
			}

			return next(c)
		}
	}
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	MaxRequests int
	Window      time.Duration
}

type rateLimitEntry struct {
	count     int
	resetTime time.Time
}

// RateLimiter enforces a per-IP sliding-window request limit using an
// in-memory store protected by a mutex (safe for concurrent use).
//
// NOTE: This is designed for single-instance deployments. For distributed
// setups, replace the local store with a shared backend such as Redis.
func RateLimiter(config RateLimitConfig) MiddlewareFunc {
	var mu sync.Mutex
	store := make(map[string]*rateLimitEntry)

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			key := c.Request.RemoteAddr
			now := time.Now()

			mu.Lock()
			entry, exists := store[key]
			if !exists {
				store[key] = &rateLimitEntry{count: 1, resetTime: now.Add(config.Window)}
				mu.Unlock()
				return next(c)
			}
			if now.After(entry.resetTime) {
				entry.count = 1
				entry.resetTime = now.Add(config.Window)
				mu.Unlock()
				return next(c)
			}
			if entry.count >= config.MaxRequests {
				mu.Unlock()
				return NewHTTPError(http.StatusTooManyRequests, "Too Many Requests")
			}
			entry.count++
			mu.Unlock()

			return next(c)
		}
	}
}

// Timeout sets a deadline on the request's context.Context. Any downstream
// handler or I/O operation that respects ctx.Done() (e.g. database calls,
// outbound HTTP requests) will be cancelled when the timeout elapses.
//
// Unlike the previous goroutine-based implementation, this approach carries
// no risk of a data race or goroutine leak. The server-level WriteTimeout
// provides a hard cap for handlers that do not use the context.
func Timeout(duration time.Duration) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
			defer cancel()

			c.Request = c.Request.WithContext(ctx)
			err := next(c)

			// Surface a 408 if the deadline fired and the handler didn't
			// already return its own error.
			if err == nil && ctx.Err() == context.DeadlineExceeded {
				return NewHTTPError(http.StatusRequestTimeout, "Request Timeout")
			}
			return err
		}
	}
}
