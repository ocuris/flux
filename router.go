package flux

import (
	"strings"
	"sync"
)

// RouteNode is one node in the routing trie.
type RouteNode struct {
	static   map[string]*RouteNode  // static path segments
	param    *RouteNode             // :param segment (single segment capture)
	wildcard *RouteNode             // * catch-all segment (captures rest of path)
	paramKey string                 // parameter name, e.g. "id" for :id
	handlers map[string]HandlerFunc // HTTP method → already-composed handler
}

// CacheEntry is a cached result of a successful route match.
type CacheEntry struct {
	handler HandlerFunc
	params  []Param
}

// Router implements a trie-based HTTP router with a static-route cache.
type Router struct {
	root  *RouteNode
	cache map[string]*CacheEntry
	mu    sync.RWMutex
}

// NewRouter creates a new empty Router.
func NewRouter() *Router {
	return &Router{
		root: &RouteNode{
			static:   make(map[string]*RouteNode),
			handlers: make(map[string]HandlerFunc),
		},
		cache: make(map[string]*CacheEntry),
	}
}

// Add registers method+path in the trie with the given (already middleware-composed) handler.
//
// Segment types:
//   - "segment"  → static match (fastest)
//   - ":name"    → named parameter (captures one path segment)
//   - "*"        → wildcard catch-all (captures the rest of the path as param "*")
func (r *Router) Add(method, path string, handler HandlerFunc) {
	node := r.root
	segments := strings.Split(strings.Trim(path, "/"), "/")

	for _, segment := range segments {
		if segment == "" {
			continue
		}
		if segment == "*" {
			// Catch-all: create a wildcard child and stop descending
			if node.wildcard == nil {
				node.wildcard = &RouteNode{
					static:   make(map[string]*RouteNode),
					handlers: make(map[string]HandlerFunc),
				}
			}
			node = node.wildcard
			break
		} else if strings.HasPrefix(segment, ":") {
			paramKey := segment[1:]
			if node.param == nil {
				node.param = &RouteNode{
					static:   make(map[string]*RouteNode),
					paramKey: paramKey,
					handlers: make(map[string]HandlerFunc),
				}
			}
			node = node.param
		} else {
			if node.static == nil {
				node.static = make(map[string]*RouteNode)
			}
			if _, exists := node.static[segment]; !exists {
				node.static[segment] = &RouteNode{
					static:   make(map[string]*RouteNode),
					handlers: make(map[string]HandlerFunc),
				}
			}
			node = node.static[segment]
		}
	}

	node.handlers[method] = handler

	// Invalidate any cached entry for this route
	r.mu.Lock()
	delete(r.cache, method+" "+path)
	r.mu.Unlock()
}

// Match finds the handler for method+path.
//
// Returns:
//   - handler:           the matched HandlerFunc (nil if not found)
//   - params:            path parameters extracted from the URL
//   - methodNotAllowed:  true when the path matched but not for this method (→ 405)
func (r *Router) Match(method, path string) (HandlerFunc, []Param, bool) {
	cacheKey := method + " " + path

	// Fast path: static-route cache
	r.mu.RLock()
	if cached, ok := r.cache[cacheKey]; ok {
		r.mu.RUnlock()
		paramsCopy := make([]Param, len(cached.params))
		copy(paramsCopy, cached.params)
		return cached.handler, paramsCopy, false
	}
	r.mu.RUnlock()

	// Slow path: trie traversal
	node := r.root
	params := make([]Param, 0, 4)
	isStatic := true // will be set false if any param/wildcard is captured
	trimmed := strings.Trim(path, "/")

	if trimmed == "" {
		// Root path
		handler, exists := node.handlers[method]
		if exists {
			r.mu.Lock()
			r.cache[cacheKey] = &CacheEntry{handler: handler, params: []Param{}}
			r.mu.Unlock()
			return handler, nil, false
		}
		if len(node.handlers) > 0 {
			return nil, nil, true // path exists, wrong method
		}
		return nil, nil, false
	}

	segments := strings.Split(trimmed, "/")

	for i, segment := range segments {
		if segment == "" {
			continue
		}

		// 1. Static match (highest priority)
		if next, ok := node.static[segment]; ok {
			node = next
			continue
		}

		// 2. Named parameter match
		if node.param != nil {
			params = append(params, Param{Key: node.param.paramKey, Value: segment})
			node = node.param
			isStatic = false
			continue
		}

		// 3. Wildcard catch-all
		if node.wildcard != nil {
			params = append(params, Param{
				Key:   "*",
				Value: strings.Join(segments[i:], "/"),
			})
			node = node.wildcard
			isStatic = false
			break
		}

		// No match
		return nil, nil, false
	}

	handler, exists := node.handlers[method]
	if !exists {
		// Check wildcard child as a final fallback (e.g. /path/* with method mismatch)
		if node.wildcard != nil {
			if h, ok := node.wildcard.handlers[method]; ok {
				return h, params, false
			}
			if len(node.wildcard.handlers) > 0 {
				return nil, nil, true
			}
		}
		if len(node.handlers) > 0 {
			return nil, nil, true // path found, method not allowed → 405
		}
		return nil, nil, false
	}

	// Cache purely-static routes for O(1) future matches
	if isStatic {
		r.mu.Lock()
		r.cache[cacheKey] = &CacheEntry{handler: handler, params: []Param{}}
		r.mu.Unlock()
	}

	return handler, params, false
}

// Route represents a registered route (used by OpenAPI generation).
type Route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Name   string `json:"name"`
}
