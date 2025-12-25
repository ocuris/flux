package main

import (
	"fmt"

	"github.com/ocuris/flux"
)

func main() {
	app := flux.New(flux.Config{
		Title:       "My Flux App",
		Version:     "1.0.0",
		Description: "An example Flux application",
		Debug:       true,
	})

	app.GET("/hello", func(c *flux.Context) error {
		fmt.Println("Hello world")
		return nil
	})

	if err := app.Start(":8080"); err != nil {
		panic(err)
	}
}
