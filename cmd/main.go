package main

import (
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"log"
	"os"
	// Blank-import the function package so the init() runs
	_ "github.com/call-me-maple/sleepyhead"
)

func main() {
	// Use PORT environment variable, or default to 8080.
	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	if err := funcframework.Start(port); err != nil {
		log.Fatalf("funcframework.Start: %v\n", err)
	}
}
