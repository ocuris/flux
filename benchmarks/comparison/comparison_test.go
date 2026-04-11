package bench

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"
	"github.com/ocuris/flux"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

// ----------------------------------------------------------------------------
// 1. DEEP ROUTE CHALLENGE (7 segments deep)
// ----------------------------------------------------------------------------

func BenchmarkDeep_Flux(b *testing.B) {
	app := flux.New(flux.Config{})
	app.GET("/api/v1/cloud/storage/files/metadata/download", func(c *flux.Context) error { return nil })
	req := httptest.NewRequest("GET", "/api/v1/cloud/storage/files/metadata/download", nil)
	w := httptest.NewRecorder()
	for b.Loop() {
		app.ServeHTTP(w, req)
	}
}

func BenchmarkDeep_Gin(b *testing.B) {
	router := gin.New()
	router.GET("/api/v1/cloud/storage/files/metadata/download", func(c *gin.Context) {})
	req := httptest.NewRequest("GET", "/api/v1/cloud/storage/files/metadata/download", nil)
	w := httptest.NewRecorder()
	for b.Loop() {
		router.ServeHTTP(w, req)
	}
}

func BenchmarkDeep_Echo(b *testing.B) {
	e := echo.New()
	e.GET("/api/v1/cloud/storage/files/metadata/download", func(c echo.Context) error { return nil })
	req := httptest.NewRequest("GET", "/api/v1/cloud/storage/files/metadata/download", nil)
	w := httptest.NewRecorder()
	for b.Loop() {
		e.ServeHTTP(w, req)
	}
}

func BenchmarkDeep_Fiber(b *testing.B) {
	f := fiber.New()
	f.Get("/api/v1/cloud/storage/files/metadata/download", func(c *fiber.Ctx) error { return nil })
	req := httptest.NewRequest("GET", "/api/v1/cloud/storage/files/metadata/download", nil)
	for b.Loop() {
		_, _ = f.Test(req)
	}
}

// ----------------------------------------------------------------------------
// 2. MIDDLEWARE CHAIN CHALLENGE (5 layers of middleware)
// ----------------------------------------------------------------------------

func dummyFluxMiddleware(next flux.HandlerFunc) flux.HandlerFunc {
	return func(c *flux.Context) error { return next(c) }
}

func BenchmarkMiddleware_Flux(b *testing.B) {
	app := flux.New(flux.Config{})
	app.Use(dummyFluxMiddleware, dummyFluxMiddleware, dummyFluxMiddleware, dummyFluxMiddleware, dummyFluxMiddleware)
	app.GET("/bench", func(c *flux.Context) error { return nil })
	req := httptest.NewRequest("GET", "/bench", nil)
	w := httptest.NewRecorder()
	for b.Loop() {
		app.ServeHTTP(w, req)
	}
}

func dummyGinMiddleware(c *gin.Context) { c.Next() }

func BenchmarkMiddleware_Gin(b *testing.B) {
	router := gin.New()
	router.Use(dummyGinMiddleware, dummyGinMiddleware, dummyGinMiddleware, dummyGinMiddleware, dummyGinMiddleware)
	router.GET("/bench", func(c *gin.Context) {})
	req := httptest.NewRequest("GET", "/bench", nil)
	w := httptest.NewRecorder()
	for b.Loop() {
		router.ServeHTTP(w, req)
	}
}

func dummyEchoMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error { return next(c) }
}

func BenchmarkMiddleware_Echo(b *testing.B) {
	e := echo.New()
	e.Use(dummyEchoMiddleware, dummyEchoMiddleware, dummyEchoMiddleware, dummyEchoMiddleware, dummyEchoMiddleware)
	e.GET("/bench", func(c echo.Context) error { return nil })
	req := httptest.NewRequest("GET", "/bench", nil)
	w := httptest.NewRecorder()
	for b.Loop() {
		e.ServeHTTP(w, req)
	}
}

func dummyFiberMiddleware(c *fiber.Ctx) error { return c.Next() }

func BenchmarkMiddleware_Fiber(b *testing.B) {
	f := fiber.New()
	f.Use(dummyFiberMiddleware, dummyFiberMiddleware, dummyFiberMiddleware, dummyFiberMiddleware, dummyFiberMiddleware)
	f.Get("/bench", func(c *fiber.Ctx) error { return nil })
	req := httptest.NewRequest("GET", "/bench", nil)
	for b.Loop() {
		_, _ = f.Test(req)
	}
}

// ----------------------------------------------------------------------------
// 3. PARALLEL THROUGHPUT CHALLENGE (Concurrency test)
// ----------------------------------------------------------------------------

func BenchmarkParallel_Flux(b *testing.B) {
	app := flux.New(flux.Config{})
	app.GET("/parallel", func(c *flux.Context) error { return nil })
	b.RunParallel(func(pb *testing.PB) {
		req := httptest.NewRequest("GET", "/parallel", nil)
		w := httptest.NewRecorder()
		for pb.Next() {
			app.ServeHTTP(w, req)
		}
	})
}

func BenchmarkParallel_Gin(b *testing.B) {
	router := gin.New()
	router.GET("/parallel", func(c *gin.Context) {})
	b.RunParallel(func(pb *testing.PB) {
		req := httptest.NewRequest("GET", "/parallel", nil)
		w := httptest.NewRecorder()
		for pb.Next() {
			router.ServeHTTP(w, req)
		}
	})
}

func BenchmarkParallel_Echo(b *testing.B) {
	e := echo.New()
	e.GET("/parallel", func(c echo.Context) error { return nil })
	b.RunParallel(func(pb *testing.PB) {
		req := httptest.NewRequest("GET", "/parallel", nil)
		w := httptest.NewRecorder()
		for pb.Next() {
			e.ServeHTTP(w, req)
		}
	})
}

func BenchmarkParallel_Fiber(b *testing.B) {
	f := fiber.New()
	f.Get("/parallel", func(c *fiber.Ctx) error { return nil })
	b.RunParallel(func(pb *testing.PB) {
		req := httptest.NewRequest("GET", "/parallel", nil)
		for pb.Next() {
			_, _ = f.Test(req)
		}
	})
}

// ----------------------------------------------------------------------------
// 4. JSON BIG RESPONSE CHALLENGE
// ----------------------------------------------------------------------------

func BenchmarkJSONBig_Flux(b *testing.B) {
	app := flux.New(flux.Config{})
	data := flux.Map{
		"id": 1, "name": "Test User", "role": "admin",
		"meta": flux.Map{"login_count": 42, "last_ip": "127.0.0.1"},
		"tags": []string{"fast", "secure", "lean"},
	}
	app.GET("/json", func(c *flux.Context) error {
		return c.JSON(200, data)
	})
	req := httptest.NewRequest("GET", "/json", nil)
	w := httptest.NewRecorder()
	for b.Loop() {
		app.ServeHTTP(w, req)
	}
}

func BenchmarkJSONBig_Gin(b *testing.B) {
	router := gin.New()
	data := gin.H{
		"id": 1, "name": "Test User", "role": "admin",
		"meta": gin.H{"login_count": 42, "last_ip": "127.0.0.1"},
		"tags": []string{"fast", "secure", "lean"},
	}
	router.GET("/json", func(c *gin.Context) {
		c.JSON(200, data)
	})
	req := httptest.NewRequest("GET", "/json", nil)
	w := httptest.NewRecorder()
	for b.Loop() {
		router.ServeHTTP(w, req)
	}
}

func BenchmarkJSONBig_Echo(b *testing.B) {
	e := echo.New()
	data := map[string]interface{}{
		"id": 1, "name": "Test User", "role": "admin",
		"meta": map[string]interface{}{"login_count": 42, "last_ip": "127.0.0.1"},
		"tags": []string{"fast", "secure", "lean"},
	}
	e.GET("/json", func(c echo.Context) error {
		return c.JSON(200, data)
	})
	req := httptest.NewRequest("GET", "/json", nil)
	w := httptest.NewRecorder()
	for b.Loop() {
		e.ServeHTTP(w, req)
	}
}

func BenchmarkJSONBig_Fiber(b *testing.B) {
	f := fiber.New()
	data := fiber.Map{
		"id": 1, "name": "Test User", "role": "admin",
		"meta": fiber.Map{"login_count": 42, "last_ip": "127.0.0.1"},
		"tags": []string{"fast", "secure", "lean"},
	}
	f.Get("/json", func(c *fiber.Ctx) error {
		return c.JSON(data)
	})
	req := httptest.NewRequest("GET", "/json", nil)
	for b.Loop() {
		_, _ = f.Test(req)
	}
}
