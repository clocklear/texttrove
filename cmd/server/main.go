package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/clocklear/texttrove/pkg/server"
	"github.com/clocklear/texttrove/pkg/tools/rag"
)

func main() {
	var (
		port     = flag.Int("port", 8080, "Port to listen on")
		basePath = flag.String("base-path", "", "Base path for documents")
		pattern  = flag.String("pattern", "*.md", "File pattern for documents")
	)
	flag.Parse()

	if *basePath == "" {
		fmt.Println("base-path is required")
		os.Exit(1)
	}

	// Initialize RAG
	ragger, err := rag.NewRagger()
	if err != nil {
		log.Fatal(err)
	}

	// Load initial documents
	if err := ragger.LoadDocuments(context.Background(), *basePath, *pattern); err != nil {
		log.Fatal(err)
	}

	// Create and start server
	srv := server.New(ragger)
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatal(err)
	}
}
