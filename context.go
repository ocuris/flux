package flux

import "net/http"

type Context interface {
	Logger() Logger
	Request() *http.Request
	Flux() *Flux
}

// type context struct {
// 	logger  Logger
// 	request *http.Request
// 	echo    *Flux
// }
