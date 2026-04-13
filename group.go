package flux

import (
	"fmt"
	"net/http"
	"reflect"
)

// Group facilitates route prefixing, shared middleware, and metadata inheritance.
type Group struct {
	prefix     string
	middleware []MiddlewareFunc
	tags       []string
	app        *Flux
}

// newGroup is the internal constructor used by Flux.Group.
func newGroup(app *Flux, prefix string, args ...any) *Group {
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
func (g *Group) Group(prefix string, args ...any) *Group {
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
func (g *Group) GET(path string, args ...any) {
	g.add(http.MethodGet, path, args...)
}

// POST registers a POST handler within this group.
func (g *Group) POST(path string, args ...any) {
	g.add(http.MethodPost, path, args...)
}

// PUT registers a PUT handler within this group.
func (g *Group) PUT(path string, args ...any) {
	g.add(http.MethodPut, path, args...)
}

// DELETE registers a DELETE handler within this group.
func (g *Group) DELETE(path string, args ...any) {
	g.add(http.MethodDelete, path, args...)
}

// PATCH registers a PATCH handler within this group.
func (g *Group) PATCH(path string, args ...any) {
	g.add(http.MethodPatch, path, args...)
}

// HEAD registers a HEAD handler within this group.
func (g *Group) HEAD(path string, args ...any) {
	g.add(http.MethodHead, path, args...)
}

// OPTIONS registers an OPTIONS handler within this group.
func (g *Group) OPTIONS(path string, args ...any) {
	g.add(http.MethodOptions, path, args...)
}

// Any registers a route for ALL standard HTTP methods within this group.
func (g *Group) Any(path string, args ...any) {
	methods := []string{
		http.MethodGet, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodPatch, http.MethodHead,
		http.MethodOptions,
	}
	for _, m := range methods {
		g.add(m, path, args...)
	}
}

// Match registers a route for a specific set of HTTP methods within this group.
func (g *Group) Match(methods []string, path string, args ...any) {
	for _, m := range methods {
		g.add(m, path, args...)
	}
}

// add parses the variadic args, wraps the handler with group middleware, and
// delegates to the Flux app's addRoute (which composes global middleware).
func (g *Group) add(method, path string, args ...any) {
	var doc *DocBuilder
	var handler HandlerFunc
	var mws []MiddlewareFunc

	for _, arg := range args {
		if d, ok := arg.(*DocBuilder); ok {
			doc = d
			continue
		}
		if info, ok := arg.(Info); ok {
			doc = Doc(info.Summary, info.Description, info.Tags...)
			continue
		}

		// Attempt to extract handler or middleware
		if h := extractHandler(arg); h != nil {
			handler = h
			continue
		}

		// If it's not a handler, maybe it's a middleware?
		v := reflect.ValueOf(arg)
		if v.IsValid() && v.Kind() == reflect.Func {
			t := v.Type()
			if t.NumIn() == 1 && t.NumOut() == 1 &&
				t.In(0).ConvertibleTo(handlerType) &&
				t.Out(0).ConvertibleTo(handlerType) {
				mws = append(mws, v.Convert(reflect.TypeOf((*MiddlewareFunc)(nil)).Elem()).Interface().(MiddlewareFunc))
				continue
			}
			panic(fmt.Sprintf("flux: invalid function signature for %s %s. Expected HandlerFunc or MiddlewareFunc, but got %s", method, path, t.String()))
		}
	}

	if handler == nil {
		panic("flux: no handler provided for " + method + " " + g.prefix + path)
	}

	// 1. Apply route-specific middleware
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}

	// 2. Apply group-scoped middleware
	final := handler
	for i := len(g.middleware) - 1; i >= 0; i-- {
		final = g.middleware[i](final)
	}

	g.app.addRoute(method, g.prefix+path, final, doc, g.tags)
}
