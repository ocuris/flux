package main

import (
	"fmt"
	"net/http"
)

func main() {
	app := http.NewServeMux()
	fmt.Println("HTTP server initialized:", app)
	http.ListenAndServe(":8080", app)
}
