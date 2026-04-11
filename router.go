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

	// Method-specific optimized static maps for O(1) lock-free matching
	staticMethods map[string]map[string]HandlerFunc
}

// NewRouter creates a new empty Router.
func NewRouter() *Router {
	return &Router{
		root: &RouteNode{
			static:   make(map[string]*RouteNode),
			handlers: make(map[string]HandlerFunc),
		},
		cache:         make(map[string]*CacheEntry),
		staticMethods: make(map[string]map[string]HandlerFunc),
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

	// Also add to the optimized static map
	r.mu.Lock()
	if r.staticMethods[method] == nil {
		r.staticMethods[method] = make(map[string]HandlerFunc)
	}
	r.staticMethods[method][path] = handler
	
	// Invalidate any cached entry for this route
	delete(r.cache, method+" "+path)
	r.mu.Unlock()
}

// Match finds the handler for method+path.
//
// Returns:
//   - handler:           the matched HandlerFunc (nil if not found)
//   - params:            path parameters extracted from the URL
//   - methodNotAllowed:  true when the path matched but not for this method (→ 405)
// Match finds the handler for method+path.
func (r *Router) Match(method, path string, params *[]Param) (HandlerFunc, bool) {
	// 1. FASTEST PATH: Lock-free Method-Specific Static Lookup
	// We check for purely static routes first.
	if m, ok := r.staticMethods[method]; ok {
		if h, ok := m[path]; ok {
			return h, false
		}
	}

	// 2. Slow path: Trie Traversal
	node := r.root
	isStatic := true

	// Manual segmentation to avoid strings.Split
	search := path
	if len(search) > 1 && search[0] == '/' {
		search = search[1:]
	}
	if len(search) > 0 && search[len(search)-1] == '/' {
		search = search[:len(search)-1]
	}

	if search == "" || search == "/" {
		handler, exists := node.handlers[method]
		if exists {
			return handler, false
		}
		if len(node.handlers) > 0 {
			return nil, true
		}
		return nil, false
	}

	start := 0
	for i := 0; i <= len(search); i++ {
		if i == len(search) || search[i] == '/' {
			if i == start {
				start = i + 1
				continue
			}
			segment := search[start:i]

			// a) Static match
			if next, ok := node.static[segment]; ok {
				node = next
			} else if node.param != nil {
				// b) Param match
				*params = append(*params, Param{Key: node.param.paramKey, Value: segment})
				node = node.param
				isStatic = false
			} else if node.wildcard != nil {
				// c) Wildcard match
				*params = append(*params, Param{Key: "*", Value: search[start:]})
				node = node.wildcard
				isStatic = false
				start = len(search)
				break
			} else {
				return nil, false
			}

			start = i + 1
		}
	}

	handler, exists := node.handlers[method]
	if !exists {
		if len(node.handlers) > 0 {
			return nil, true
		}
		return nil, false
	}

	// Cache purely static routes
	if isStatic {
		r.mu.Lock()
		r.cache[method+" "+path] = &CacheEntry{handler: handler, params: nil}
		r.mu.Unlock()
	}

	return handler, false
}

// Route represents a registered route (used by OpenAPI generation).
type Route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Name   string `json:"name"`
}
