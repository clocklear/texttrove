package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clocklear/chromem-go"
	"github.com/clocklear/texttrove/pkg/api"
	"github.com/clocklear/texttrove/pkg/db/rag"
	"github.com/kelseyhightower/envconfig"
	"github.com/tmc/langchaingo/llms/ollama"
)

type config struct {
	Server struct {
		Addr string `default:":8080"`
	}
	Model struct {
		Conversation string `default:"llama3.2:latest"`
		Embedding    struct {
			Name         string `default:"mxbai-embed-large:latest"`
			PromptPrefix struct {
				Query     string `default:"Represent this sentence for searching relevant passages: "`
				Embedding string
			}
		}
	}
	Document struct {
		Path        string `required:"true"`
		FilePattern string `default:"*.md"`
	}
	Database struct {
		Path string `default:"texttrove.db"`
	}
}

func main() {
	// Load config from environment
	var cfg config
	if err := envconfig.Process("", &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process config: %v\n", err)
		os.Exit(1)
	}

	// Create RAG instance
	rag, err := rag.NewChromemRag(cfg.Database.Path, rag.ModelPrompts{
		QueryPrefix:     cfg.Model.Embedding.PromptPrefix.Query,
		EmbeddingPrefix: cfg.Model.Embedding.PromptPrefix.Embedding,
	}, chromem.NewEmbeddingFuncOllama(cfg.Model.Embedding.Name, ""))
	if err != nil {
		log.Fatalf("Failed to create RAG: %v", err)
	}

	// Create LLM instance
	llm, err := ollama.New(
		ollama.WithModel(cfg.Model.Conversation),
	)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Load documents
	if err := rag.LoadDocuments(context.Background(), cfg.Document.Path, cfg.Document.FilePattern); err != nil {
		log.Fatalf("Failed to load documents: %v", err)
	}

	// Create and start server
	server := api.NewServer(cfg.Server.Addr, rag, llm)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server listening on %s", cfg.Server.Addr)
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down server...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
