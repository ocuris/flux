package flux

import (
	"net/http"
)

// Group is a collection of routes sharing a common path prefix and optional
// group-scoped middleware. Groups are created via app.Group(prefix).
//
// Middleware execution order for a grouped route:
//
//	[global middleware] → [group middleware] → handler
type Group struct {
	prefix     string
	middleware []MiddlewareFunc
	app        *Flux
}

// newGroup is the internal constructor used by Flux.Group.
func newGroup(app *Flux, prefix string, middleware ...MiddlewareFunc) *Group {
	return &Group{
		prefix:     prefix,
		middleware: middleware,
		app:        app,
	}
}

// Use appends group-scoped middleware. These run after global middleware and
// before the route handler.
func (g *Group) Use(middleware ...MiddlewareFunc) {
	g.middleware = append(g.middleware, middleware...)
}

// Group creates a nested group with an additional prefix relative to this group.
// The parent group's middleware is inherited.
func (g *Group) Group(prefix string, middleware ...MiddlewareFunc) *Group {
	// Clone parent middleware slice to avoid mutation across sibling groups
	inherited := make([]MiddlewareFunc, len(g.middleware))
	copy(inherited, g.middleware)
	return &Group{
		prefix:     g.prefix + prefix,
		middleware: append(inherited, middleware...),
		app:        g.app,
	}
}

// GET registers a GET handler within this group.
func (g *Group) GET(path string, args ...interface{}) {
	g.add(http.MethodGet, path, args...)
}

// POST registers a POST handler within this group.
func (g *Group) POST(path string, args ...interface{}) {
	g.add(http.MethodPost, path, args...)
}

// PUT registers a PUT handler within this group.
func (g *Group) PUT(path string, args ...interface{}) {
	g.add(http.MethodPut, path, args...)
}

// DELETE registers a DELETE handler within this group.
func (g *Group) DELETE(path string, args ...interface{}) {
	g.add(http.MethodDelete, path, args...)
}

// PATCH registers a PATCH handler within this group.
func (g *Group) PATCH(path string, args ...interface{}) {
	g.add(http.MethodPatch, path, args...)
}

// HEAD registers a HEAD handler within this group.
func (g *Group) HEAD(path string, args ...interface{}) {
	g.add(http.MethodHead, path, args...)
}

// OPTIONS registers an OPTIONS handler within this group.
func (g *Group) OPTIONS(path string, args ...interface{}) {
	g.add(http.MethodOptions, path, args...)
}

// add parses the variadic args, wraps the handler with group middleware, and
// delegates to the Flux app's addRoute (which composes global middleware).
func (g *Group) add(method, path string, args ...interface{}) {
	var doc *DocBuilder
	var handler HandlerFunc

	for _, arg := range args {
		if d, ok := arg.(*DocBuilder); ok {
			doc = d
			continue
		}
		if h := extractHandler(arg); h != nil {
			handler = h
		}
	}

	if handler == nil {
		panic("flux: no handler provided for " + method + " " + g.prefix + path)
	}

	// Apply group-scoped middleware (runs after global middleware)
	final := handler
	for i := len(g.middleware) - 1; i >= 0; i-- {
		final = g.middleware[i](final)
	}

	g.app.addRoute(method, g.prefix+path, final, doc)
}
