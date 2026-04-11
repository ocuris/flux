package flux

import (
	"strings"
	"sync"
	"sync/atomic"
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

// Router implements a trie-based HTTP router with an atomic static-route lookup.
type Router struct {
	root *RouteNode
	mu   sync.Mutex // protects trie and static map updates

	// Method-specific atomic maps for lock-free O(1) lookup
	staticGET    atomic.Pointer[map[string]HandlerFunc]
	staticPOST   atomic.Pointer[map[string]HandlerFunc]
	staticPUT    atomic.Pointer[map[string]HandlerFunc]
	staticDELETE atomic.Pointer[map[string]HandlerFunc]
	staticPATCH  atomic.Pointer[map[string]HandlerFunc]
	staticOTHER  atomic.Pointer[map[string]map[string]HandlerFunc]
}

// NewRouter creates a new empty Router.
func NewRouter() *Router {
	r := &Router{
		root: &RouteNode{
			static:   make(map[string]*RouteNode),
			handlers: make(map[string]HandlerFunc),
		},
	}
	empty := make(map[string]HandlerFunc)
	r.staticGET.Store(&empty)
	r.staticPOST.Store(&empty)
	r.staticPUT.Store(&empty)
	r.staticDELETE.Store(&empty)
	r.staticPATCH.Store(&empty)

	other := make(map[string]map[string]HandlerFunc)
	r.staticOTHER.Store(&other)
	return r
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

	// Update method-specific atomic map
	r.mu.Lock()
	defer r.mu.Unlock()

	switch method {
	case "GET":
		r.staticGET.Store(r.cloneAndAdd(r.staticGET.Load(), path, handler))
	case "POST":
		r.staticPOST.Store(r.cloneAndAdd(r.staticPOST.Load(), path, handler))
	case "PUT":
		r.staticPUT.Store(r.cloneAndAdd(r.staticPUT.Load(), path, handler))
	case "DELETE":
		r.staticDELETE.Store(r.cloneAndAdd(r.staticDELETE.Load(), path, handler))
	case "PATCH":
		r.staticPATCH.Store(r.cloneAndAdd(r.staticPATCH.Load(), path, handler))
	default:
		oldOther := r.staticOTHER.Load()
		newOther := make(map[string]map[string]HandlerFunc)
		if oldOther != nil {
			for m, inner := range *oldOther {
				newInner := make(map[string]HandlerFunc)
				for p, h := range inner {
					newInner[p] = h
				}
				newOther[m] = newInner
			}
		}
		if newOther[method] == nil {
			newOther[method] = make(map[string]HandlerFunc)
		}
		newOther[method][path] = handler
		r.staticOTHER.Store(&newOther)
	}
}

func (r *Router) cloneAndAdd(old *map[string]HandlerFunc, path string, handler HandlerFunc) *map[string]HandlerFunc {
	newMap := make(map[string]HandlerFunc)
	if old != nil {
		for k, v := range *old {
			newMap[k] = v
		}
	}
	newMap[path] = handler
	return &newMap
}

// Match finds the handler for method+path.
//
// Returns:
//   - handler:           the matched HandlerFunc (nil if not found)
//   - params:            path parameters extracted from the URL
//   - methodNotAllowed:  true when the path matched but not for this method (→ 405)
//
// Match finds the handler for method+path.
func (r *Router) Match(method, path string, params *[]Param) (HandlerFunc, bool) {
	// 1. FASTEST PATH: Method-specific single lookup
	switch method {
	case "GET":
		if m := r.staticGET.Load(); m != nil {
			if h, ok := (*m)[path]; ok {
				return h, false
			}
		}
	case "POST":
		if m := r.staticPOST.Load(); m != nil {
			if h, ok := (*m)[path]; ok {
				return h, false
			}
		}
	case "PUT":
		if m := r.staticPUT.Load(); m != nil {
			if h, ok := (*m)[path]; ok {
				return h, false
			}
		}
	case "DELETE":
		if m := r.staticDELETE.Load(); m != nil {
			if h, ok := (*m)[path]; ok {
				return h, false
			}
		}
	case "PATCH":
		if m := r.staticPATCH.Load(); m != nil {
			if h, ok := (*m)[path]; ok {
				return h, false
			}
		}
	default:
		if m := r.staticOTHER.Load(); m != nil {
			if inner, ok := (*m)[method]; ok {
				if h, ok := inner[path]; ok {
					return h, false
				}
			}
		}
	}

	// Path is already cleaned by ServeHTTP
	node := r.root
	search := path
	if len(search) > 0 && search[0] == '/' {
		search = search[1:]
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

			isStatic := false
			_ = isStatic // Silence if unused (only if we need it for caching, which we removed)
			if next, ok := node.static[segment]; ok {
				node = next
			} else if node.param != nil {
				*params = append(*params, Param{Key: node.param.paramKey, Value: segment})
				node = node.param
			} else if node.wildcard != nil {
				*params = append(*params, Param{Key: "*", Value: search[start:]})
				node = node.wildcard
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

	return handler, false
}

// Route represents a registered route (used by OpenAPI generation).
type Route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Name   string `json:"name"`
}
