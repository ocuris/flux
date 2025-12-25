package flux

import "net/http"

type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request
	app     *Flux
	params  []Param
	depends   map[string]interface{}  // dependency injection store
}

type Param struct {
	Key   string
	Value string
}

