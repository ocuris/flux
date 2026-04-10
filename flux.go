package flux

import (
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
	}

	for _, opt := range opts {
		opt(app)
	}

	app.pool = &sync.Pool{
		New: func() interface{} {
			return &Context{
				app:    app,
				params: make([]Param, 0, 8),
				store:  make(map[string]interface{}),
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
		panic(fmt.Sprintf("flux: no handler provided for %s %s", method, path))
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

	// 2. Wrap the core handler so errors are resolved at the innermost layer.
	errorHandled := func(c *Context) error {
		if err := handler(c); err != nil {
			f.handleError(c, err)
		}
		return nil
	}

	// 3. Compose framework/group middleware around the error-handling wrapper.
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
		// The user passed a function, but the signature doesn't match func(*Context) error.
		panic(fmt.Sprintf("flux: invalid handler signature. Expected func(*flux.Context) error, but got %s", v.Type().String()))
	}

	return nil
}

// Start registers the OpenAPI documentation endpoints, launches the HTTP
// server, and blocks until SIGINT or SIGTERM is received, then performs a
// graceful 30-second shutdown.
func (f *Flux) Start(addr string, opts ...StartOption) error {
	f.InitOpenAPI()

	f.server = &http.Server{
		Addr:    addr,
		Handler: f,

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	for _, opt := range opts {
		opt(f.server)
	}

	f.startupLogger.PrintStartup(addr)

	go func() {
		if err := f.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
	if f.server == nil {
		return nil
	}

	// Signal Start() to unblock if it's still waiting
	select {
	case <-f.stopChan:
	default:
		close(f.stopChan)
	}

	if err := f.server.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "flux: forced shutdown: %v\n", err)
		return err
	}

	fmt.Println("✓ Server exited gracefully")
	return nil
}

// ServeHTTP implements http.Handler. It obtains a pooled Context, runs any
// pre-routing middleware, matches the route, and dispatches to the handler
// (which already has global middleware composed in from addRoute).
func (f *Flux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := f.pool.Get().(*Context)
	c.Writer = w
	c.Request = r
	c.statusCode = 0
	c.written = false
	c.params = c.params[:0]

	defer func() {
		c.Writer = nil
		c.Request = nil
		c.statusCode = 0
		c.written = false
		for k := range c.store {
			delete(c.store, k)
		}
		f.pool.Put(c)
	}()

	// Match route. The returned handler already has global middleware baked in.
	handler, params, methodNotAllowed := f.router.Match(r.Method, r.URL.Path)

	if methodNotAllowed {
		_ = c.JSON(http.StatusMethodNotAllowed, Map{"error": "method not allowed"})
		return
	}
	if handler == nil {
		_ = c.JSON(http.StatusNotFound, Map{
			"error": fmt.Sprintf("endpoint '%s' not found", r.URL.Path),
		})
		return
	}

	c.params = params

	// Apply pre-routing middleware around the composed handler.
	final := handler
	for i := len(f.preMiddleware) - 1; i >= 0; i-- {
		final = f.preMiddleware[i](final)
	}

	if err := final(c); err != nil {
		f.handleError(c, err)
	}
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
