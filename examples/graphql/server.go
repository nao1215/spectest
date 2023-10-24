package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-spectest/spectest/examples/graphql/graph"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, graph.NewHandler()))
}
