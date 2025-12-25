package flux


type Router interface {
	Handle(method, path string, handler HandlerFunc)
}


type Route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Name   string `json:"name"`
}
