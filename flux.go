package flux

import "sync"


type HandlerFunc func(*Context) error
type MiddlewareFunc func(HandlerFunc) HandlerFunc
type Map map[string]interface{}


type Flux struct {
	Config  Config
	pool    *sync.Pool
}

type Route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Name   string `json:"name"`
}



type Router interface {
	Handle(method, path string, handler HandlerFunc)
}

func New(cfg Config) *Flux {
	return &Flux{
		Config:  cfg,
	}
}
func (f *Flux) Use(middleware ...MiddlewareFunc) {
	// f.middleware = append(f.middleware, middleware...)
}

func (f *Flux) Pre(middleware ...MiddlewareFunc) {
	// f.middleware = append(f.middleware, middleware...)
}

func (f *Flux) GET(path string, handler HandlerFunc, ms ...RouteOption)

func (f *Flux) POST(path string, handler HandlerFunc, ms ...RouteOption)

func (f *Flux) PUT(path string, handler HandlerFunc, ms ...RouteOption)

func (f *Flux) DELETE(path string, handler HandlerFunc, ms ...RouteOption)

func (f *Flux) PATCH(path string, handler HandlerFunc, ms ...RouteOption)

