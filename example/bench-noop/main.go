package main

import (
	"github.com/ocuris/flux"
)

func main() {
	app := flux.New(flux.Config{})
	// No middleware for absolute peak speed
	app.GET("/ping", func(c *flux.Context) error {
		_, err := c.Writer.Write([]byte("pong"))
		return err
	})
	app.Start(":8006")
}
