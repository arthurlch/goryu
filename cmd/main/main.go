package main

// ================ //
// ==== sample main to create dummy version of the package for testing and sample purpose === //
// ================ //

import (
	"fmt"
	"log"
	"net/http"

	"github.com/arthurlch/goryu"
)

func main() {
	app := goryu.Default()

	app.GET("/", func(c *goryu.Context) {
		c.Text(http.StatusOK, "Welcome to the Goryu Web Framework!")
	})

	app.GET("/hello/:name", func(c *goryu.Context) {
		name := c.Params["name"]
		c.Text(http.StatusOK, fmt.Sprintf("Hello, %s!", name))
	})

	port := "8080"
	log.Printf("Server starting on http://localhost:%s\n", port)

	if err := app.Run(":" + port); err != nil {
		log.Fatalf("Could not start server: %v\n", err)
	}
}