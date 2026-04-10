package flux

import (
	"net/http"
)

// Group facilitates route prefixing, shared middleware, and metadata inheritance.
type Group struct {
	prefix     string
	middleware []MiddlewareFunc
	tags       []string
	app        *Flux
}

// newGroup is the internal constructor used by Flux.Group.
func newGroup(app *Flux, prefix string, args ...interface{}) *Group {
	g := &Group{
		prefix:     prefix,
		middleware: make([]MiddlewareFunc, 0),
		tags:       make([]string, 0),
		app:        app,
	}

	for _, arg := range args {
		if m, ok := arg.(MiddlewareFunc); ok {
			g.middleware = append(g.middleware, m)
		} else if t, ok := arg.(string); ok {
			g.tags = append(g.tags, t)
		}
	}
	return g
}

// Use appends group-scoped middleware. These run after global middleware and
// before the route handler.
func (g *Group) Use(middleware ...MiddlewareFunc) {
	g.middleware = append(g.middleware, middleware...)
}

// Group creates a nested group with an additional prefix relative to this group.
// The parent group's tags and middleware are inherited.
func (g *Group) Group(prefix string, args ...interface{}) *Group {
	inheritedMid := make([]MiddlewareFunc, len(g.middleware))
	copy(inheritedMid, g.middleware)

	inheritedTags := make([]string, len(g.tags))
	copy(inheritedTags, g.tags)

	newG := &Group{
		prefix:     g.prefix + prefix,
		middleware: inheritedMid,
		tags:       inheritedTags,
		app:        g.app,
	}

	for _, arg := range args {
		if m, ok := arg.(MiddlewareFunc); ok {
			newG.middleware = append(newG.middleware, m)
		} else if t, ok := arg.(string); ok {
			newG.tags = append(newG.tags, t)
		}
	}

	return newG
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
		if info, ok := arg.(Info); ok {
			doc = Doc(info.Summary, info.Description, info.Tags...)
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

	g.app.addRoute(method, g.prefix+path, final, doc, g.tags)
}
