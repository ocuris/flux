package flux

import (
	"net/http"
	"sync"
)

type HandlerFunc func(*Context) error
type MiddlewareFunc func(HandlerFunc) HandlerFunc
type StartOption func(*http.Server)
type Map map[string]any

type Flux struct {
	Config        Config
	middleware    []MiddlewareFunc
	logger        *Logger
	pool          *sync.Pool
	openapi       *OpenAPISpec
	server        *http.Server
	startupLogger *StartupLogger
}

func New(cfg Config) *Flux {
	startupLogger := NewStartupLogger(cfg)
	return &Flux{
		Config:        cfg,
		startupLogger: startupLogger,
	}
}
func (f *Flux) Use(middleware ...MiddlewareFunc) {
	// f.middleware = append(f.middleware, middleware...)
}

func (f *Flux) Pre(middleware ...MiddlewareFunc) {
	// f.middleware = append(f.middleware, middleware...)
}

func (f *Flux) GET(path string, handler HandlerFunc, ms ...RouteOption) {
}

func (f *Flux) POST(path string, handler HandlerFunc, ms ...RouteOption) {}

func (f *Flux) PUT(path string, handler HandlerFunc, ms ...RouteOption) {}

func (f *Flux) DELETE(path string, handler HandlerFunc, ms ...RouteOption) {}

func (f *Flux) PATCH(path string, handler HandlerFunc, ms ...RouteOption) {}

func (f *Flux) Start(addr string, opts ...StartOption) error {
	server := &http.Server{
		Addr: addr,
	}
	for _, opt := range opts {
		opt(server)
	}
	f.startupLogger.PrintStartup(addr)
	return server.ListenAndServe()
}
