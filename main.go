package main

// A simple program demonstrating the text area component from the Bubbles
// component library.
// TODO: this is currently a spaghetti mess while I am learning and experimenting.
// This will be cleaned up and refactored into a more maintainable state.

import (
	"context"
	"fmt"
	"os"

	"github.com/clocklear/texttrove/app"
	"github.com/clocklear/texttrove/pkg/db/rag"
	"github.com/philippgille/chromem-go"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kelseyhightower/envconfig"
	"github.com/tmc/langchaingo/llms/ollama"
)

type config struct {
	Model struct {
		Conversation string `default:"llama3.2-128k:latest"`
		Embedding    string `default:"mxbai-embed-large:latest"`
	}
	Document struct {
		Path        string `required:"true"`
		FilePattern string `default:"*.md"`
	}
}

func main() {

	// Load config from environment (using envconfig)
	var cliCfg config
	envconfig.MustProcess("", &cliCfg)

	// Create a (conversation) LLM
	conversationLlm, err := ollama.New(ollama.WithModel(cliCfg.Model.Conversation))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create conversation LLM: %v\n", err)
		os.Exit(1)
	}

	// Build doc DB
	r, err := rag.NewChromemRag("texttrove.db", rag.ModelPrompts{
		QueryPrefix: "Represent this sentence for searching relevant passages: ", // TODO: specific to our embedding model, parameterize
	}, chromem.NewEmbeddingFuncOllama(cliCfg.Model.Embedding, ""))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create rag: %v\n", err)
		os.Exit(1)
	}

	// Load the DB
	err = r.LoadDocuments(context.TODO(), cliCfg.Document.Path, cliCfg.Document.FilePattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load documents: %v\n", err)
		os.Exit(1)
	}

	// Create a new app model
	appCfg, err := app.DefaultConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create default config: %v\n", err)
		os.Exit(1)
	}
	appCfg.ConversationLLM = conversationLlm
	appCfg.RAG = r
	appModel := app.New(appCfg)

	p := tea.NewProgram(appModel, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithKeyboardEnhancements())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Oof: %v\n", err)
	}
}
