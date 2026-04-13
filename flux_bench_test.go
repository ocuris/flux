package flux

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkRouterStatic(b *testing.B) {
	router := NewRouter()
	handler := func(c *Context) error { return nil }
	router.Add("GET", "/api/v1/users", handler)

	var params []Param

	for b.Loop() {
		params = params[:0]
		router.Match("GET", "/api/v1/users", &params)
	}
}

func BenchmarkRouterParam(b *testing.B) {
	router := NewRouter()
	handler := func(c *Context) error { return nil }
	router.Add("GET", "/api/v1/users/:id", handler)

	var params []Param

	for b.Loop() {
		params = params[:0]
		router.Match("GET", "/api/v1/users/123", &params)
	}
}

func BenchmarkFullRequest(b *testing.B) {
	app := New(Config{})
	app.GET("/api/v1/users/:id", func(c *Context) error {
		return c.JSON(200, Map{"id": c.Param("id")})
	})

	req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
	w := httptest.NewRecorder()

	for b.Loop() {
		app.ServeHTTP(w, req)
	}
}
