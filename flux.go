package flux

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
)

// handlerType is the reflect type for HandlerFunc. Used to reinterpret 
// external functions via reflect.Convert during route registration.
var handlerType = reflect.TypeOf((*HandlerFunc)(nil)).Elem()

// HandlerFunc is the core request handler signature. 
type HandlerFunc func(*Context) error

// MiddlewareFunc wraps a HandlerFunc with additional behaviour.
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// StartOption is a functional option applied to the underlying *http.Server.
type StartOption func(*http.Server)

// Map is a shortcut for map[string]any, used primarily for JSON payloads.
type Map map[string]any

// Flux is the main framework instance.
type Flux struct {
	config        Config
	middleware    []MiddlewareFunc // composed into handlers at route registration
	preMiddleware []MiddlewareFunc // applied before routing on every request
	pool          *sync.Pool
	router        *Router
	openapi       *OpenAPISpec
	server        *http.Server
	startupLogger *StartupLogger
	encoder       Encoder

	registeredRoutes []RouteInfo
	routesMu         sync.RWMutex
	stopChan         chan struct{}
	bufPool          *sync.Pool
	mu               sync.RWMutex // protects server and other shared state
}

// Encoder defines the JSON serialization interface.
type Encoder interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

type defaultEncoder struct{}

func (e defaultEncoder) Marshal(v any) ([]byte, error)   { return json.Marshal(v) }
func (e defaultEncoder) Unmarshal(d []byte, v any) error { return json.Unmarshal(d, v) }

// Option is a functional configuration for the Flux app.
type Option func(*Flux)

// WithEncoder allows providing a custom JSON encoder.
func WithEncoder(e Encoder) Option {
	return func(f *Flux) {
		f.encoder = e
	}
}

// New initializes a Flux instance. Accepts optional configurators 
// for future-proofing (e.g. custom encoders).
func New(cfg Config, opts ...Option) *Flux {
	app := &Flux{
		config:           cfg,
		middleware:       make([]MiddlewareFunc, 0),
		preMiddleware:    make([]MiddlewareFunc, 0),
		router:           NewRouter(),
		startupLogger:    NewStartupLogger(cfg),
		registeredRoutes: make([]RouteInfo, 0),
		stopChan:         make(chan struct{}),
		encoder:          defaultEncoder{},
		bufPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}

	for _, opt := range opts {
		opt(app)
	}

	app.pool = &sync.Pool{
		New: func() interface{} {
			return &Context{
				app:    app,
				params: make([]Param, 0, 16),
			}
		},
	}

	return app
}

// Use appends global middleware that will be composed into every route
// registered after this call. Middleware MUST be registered before routes.
func (f *Flux) Use(middleware ...MiddlewareFunc) {
	f.middleware = append(f.middleware, middleware...)
}

// Pre registers pre-routing middleware. Unlike Use(), Pre() middleware runs
// before the route is matched on every request, making it suitable for
// tasks like request rewriting, canonical path normalisation, or early abort.
func (f *Flux) Pre(middleware ...MiddlewareFunc) {
	f.preMiddleware = append(f.preMiddleware, middleware...)
}

// GET registers a handler for HTTP GET requests.
func (f *Flux) GET(path string, args ...interface{}) {
	f.addDocumentedRoute(http.MethodGet, path, args...)
}

// POST registers a handler for HTTP POST requests.
func (f *Flux) POST(path string, args ...interface{}) {
	f.addDocumentedRoute(http.MethodPost, path, args...)
}

// PUT registers a handler for HTTP PUT requests.
func (f *Flux) PUT(path string, args ...interface{}) {
	f.addDocumentedRoute(http.MethodPut, path, args...)
}

// DELETE registers a handler for HTTP DELETE requests.
func (f *Flux) DELETE(path string, args ...interface{}) {
	f.addDocumentedRoute(http.MethodDelete, path, args...)
}

// PATCH registers a handler for HTTP PATCH requests.
func (f *Flux) PATCH(path string, args ...interface{}) {
	f.addDocumentedRoute(http.MethodPatch, path, args...)
}

// HEAD registers a handler for HTTP HEAD requests.
func (f *Flux) HEAD(path string, args ...interface{}) {
	f.addDocumentedRoute(http.MethodHead, path, args...)
}

// OPTIONS registers a handler for HTTP OPTIONS requests.
func (f *Flux) OPTIONS(path string, args ...interface{}) {
	f.addDocumentedRoute(http.MethodOptions, path, args...)
}

// Any registers a route for ALL standard HTTP methods.
func (f *Flux) Any(path string, args ...interface{}) {
	methods := []string{
		http.MethodGet, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodPatch, http.MethodHead,
		http.MethodOptions,
	}
	for _, m := range methods {
		f.addDocumentedRoute(m, path, args...)
	}
}

// Match registers a route for a specific set of HTTP methods.
func (f *Flux) Match(methods []string, path string, args ...interface{}) {
	for _, m := range methods {
		f.addDocumentedRoute(strings.ToUpper(m), path, args...)
	}
}

// Group creates a route group with a shared prefix. Arguments can include
// MiddlewareFunc or string (for automatic documentation tags).
func (f *Flux) Group(prefix string, args ...interface{}) *Group {
	return newGroup(f, prefix, args...)
}

// addDocumentedRoute parses variadic args for (HandlerFunc, *DocBuilder) in
// any order, then delegates to addRoute.
func (f *Flux) addDocumentedRoute(method, path string, args ...interface{}) {
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
		// We use reflection here because anonymous functions might not 
		// satisfy the Type Assertion until converted.
		v := reflect.ValueOf(arg)
		if v.IsValid() && v.Kind() == reflect.Func {
			// Check if it's a MiddlewareFunc: func(HandlerFunc) HandlerFunc
			t := v.Type()
			if t.NumIn() == 1 && t.NumOut() == 1 && 
			   t.In(0).ConvertibleTo(handlerType) && 
			   t.Out(0).ConvertibleTo(handlerType) {
				mws = append(mws, v.Convert(reflect.TypeOf((*MiddlewareFunc)(nil)).Elem()).Interface().(MiddlewareFunc))
				continue
			}
			
			// If it's a function but doesn't match anything, THEN we panic
			panic(fmt.Sprintf("flux: invalid function signature for %s %s. Expected HandlerFunc or MiddlewareFunc, but got %s", method, path, t.String()))
		}
	}

	if handler == nil {
		panic(fmt.Sprintf("flux: no handler provided for %s %s", method, path))
	}

	// Apply route-specific middleware
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}

	f.addRoute(method, path, handler, doc, nil)
}

// addRoute maps a handler and its metadata to the internal router.
func (f *Flux) addRoute(method, path string, handler HandlerFunc, doc *DocBuilder, groupTags []string) {
	// 1. Auto-Documentation Logic
	if doc == nil {
		doc = Doc("", "")
	}

	// 1.a Auto-Summary from Handler Name
	if doc.summary == "" {
		doc.summary = getFunctionName(handler)
	}

	// 1.b Auto-Tagging from Group
	if len(groupTags) > 0 {
		tagMap := make(map[string]struct{})
		for _, t := range groupTags {
			tagMap[t] = struct{}{}
		}
		for _, t := range doc.tags {
			tagMap[t] = struct{}{}
		}
		newTags := make([]string, 0, len(tagMap))
		for t := range tagMap {
			newTags = append(newTags, t)
		}
		doc.tags = newTags
	}

	errorHandled := func(c *Context) error {
		if err := handler(c); err != nil {
			f.handleError(c, err)
		}
		return nil
	}

	final := HandlerFunc(errorHandled)
	for i := len(f.middleware) - 1; i >= 0; i-- {
		final = f.middleware[i](final)
	}

	f.router.Add(method, path, final)

	f.routesMu.Lock()
	f.registeredRoutes = append(f.registeredRoutes, RouteInfo{
		Method: method,
		Path:   path,
		Doc:    doc,
	})
	f.routesMu.Unlock()

	f.startupLogger.AddRoute(method, path, doc)
}

// extractHandler attempts to obtain a HandlerFunc from an interface{} value.
//
// It handles two cases:
//  1. The value IS a flux.HandlerFunc (direct assertion succeeds).
//  2. The value is a func(*Context) error defined in another package —
//     Go stores the concrete package-local function type in the interface,
//     so a direct assertion fails even though the signatures are identical.
//     reflect.Value.Convert solves this by reinterpreting the function pointer.
func extractHandler(arg interface{}) HandlerFunc {
	if h, ok := arg.(HandlerFunc); ok {
		return h
	}

	v := reflect.ValueOf(arg)
	if v.IsValid() && v.Kind() == reflect.Func {
		if v.Type().ConvertibleTo(handlerType) {
			return v.Convert(handlerType).Interface().(HandlerFunc)
		}
	}

	return nil
}

// Start registers the OpenAPI documentation endpoints, launches the HTTP
// server, and blocks until SIGINT or SIGTERM is received, then performs a
// graceful 30-second shutdown.
func (f *Flux) Start(addr string, opts ...StartOption) error {
	f.InitOpenAPI()

	f.mu.Lock()
	f.server = &http.Server{
		Addr:    addr,
		Handler: f,

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}
	server := f.server
	f.mu.Unlock()

	for _, opt := range opts {
		opt(server)
	}

	f.startupLogger.PrintStartup(addr)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "flux: server error: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		fmt.Println("\n⏹  Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return f.Stop(ctx)
	case <-f.stopChan:
		return nil
	}
}

// Stop signals the server to begin its graceful shutdown sequence using the
// provided context for timeout/deadline control.
func (f *Flux) Stop(ctx context.Context) error {
	f.mu.Lock()
	server := f.server
	f.mu.Unlock()

	if server == nil {
		return nil
	}

	// Signal Start() to unblock if it's still waiting
	select {
	case <-f.stopChan:
	default:
		// Safe way to close once
		f.mu.Lock()
		select {
		case <-f.stopChan:
		default:
			close(f.stopChan)
		}
		f.mu.Unlock()
	}

	if err := server.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "flux: forced shutdown: %v\n", err)
		return err
	}

	fmt.Println("✓ Server exited gracefully")
	return nil
}

// ServeHTTP implements http.Handler. It is the core entry point for every request.
// It features built-in panic recovery and path sanitisation to ensure stability.
func (f *Flux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := f.pool.Get().(*Context)
	c.Writer = w
	c.Request = r
	c.statusCode = 0
	c.written = false
	c.params = c.params[:0]

	// Note: defer was removed to eliminate overhead in the hot path. 
	// Panic recovery is now the responsibility of the recovery middleware.
	// We MUST manually return the context to the pool at every exit point.

	// 3. Path Normalisation (Security: Prevents path traversal attacks)
	path := r.URL.Path
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	// 4. Match route.
	handler, methodNotAllowed := f.router.Match(r.Method, path, &c.params)

	if methodNotAllowed {
		_ = c.JSON(http.StatusMethodNotAllowed, Map{"error": "method not allowed"})
		c.reset()
		f.pool.Put(c)
		return
	}
	if handler == nil {
		_ = c.JSON(http.StatusNotFound, Map{
			"error": fmt.Sprintf("endpoint '%s' not found", path),
		})
		c.reset()
		f.pool.Put(c)
		return
	}

	if len(f.preMiddleware) == 0 {
		if err := handler(c); err != nil {
			f.handleError(c, err)
		}
		c.reset()
		f.pool.Put(c)
		return
	}

	final := handler
	for i := len(f.preMiddleware) - 1; i >= 0; i-- {
		final = f.preMiddleware[i](final)
	}

	if err := final(c); err != nil {
		f.handleError(c, err)
	}
	c.reset()
	f.pool.Put(c)
}

// handleError writes an appropriate error response. If the response has
// already been committed (c.written == true), it does nothing.
func (f *Flux) handleError(c *Context, err error) {
	if c.written {
		return
	}
	if httpErr, ok := err.(*HTTPError); ok {
		resp := Map{"error": httpErr.Message}
		if httpErr.Details != nil {
			resp["details"] = httpErr.Details
		}
		_ = c.JSON(httpErr.Code, resp)
		return
	}
	// Generic error — hide internal details from clients in production.
	resp := Map{"error": "internal server error"}
	if f.config.Debug {
		resp["details"] = err.Error()
	}
	_ = c.JSON(http.StatusInternalServerError, resp)
}

// addRoute composes the global middleware chain once at registration time
// and stores the resulting handler in the trie.
//
// The raw handler is first wrapped in an error-resolving closure so that
// when a handler returns an HTTPError, c.statusCode is set (via c.JSON)
// BEFORE outer middleware (e.g. Logger) unwinds. This guarantees Logger
// always logs the correct HTTP status code.
func getFunctionName(i interface{}) string {
	fn := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	parts := strings.Split(fn, ".")
	raw := parts[len(parts)-1]
	// Remove "-fm" suffix for bound methods
	raw = strings.TrimSuffix(raw, "-fm")
	return camelToSpaces(raw)
}

func camelToSpaces(s string) string {
	if s == "" {
		return ""
	}
	var res strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			res.WriteRune(' ')
		}
		res.WriteRune(r)
	}
	return res.String()
}

// HTTPError represents a structured HTTP error that can be returned from
// any handler or middleware.
type HTTPError struct {
	Code    int
	Message string
	Details interface{}
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Code, e.Message)
}

// NewHTTPError creates a new HTTPError. An optional details argument is
// included in the "details" field of the JSON response.
func NewHTTPError(code int, message string, details ...interface{}) *HTTPError {
	var det interface{}
	if len(details) > 0 {
		det = details[0]
	}
	return &HTTPError{Code: code, Message: message, Details: det}
}
