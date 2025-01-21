package main

// A simple program demonstrating the text area component from the Bubbles
// component library.
// TODO: this is currently a spaghetti mess while I am learning and experimenting.
// This will be cleaned up and refactored into a more maintainable state.

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/clocklear/chromem-go"
	"github.com/clocklear/texttrove/app"
	"github.com/clocklear/texttrove/pkg/db/rag"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kelseyhightower/envconfig"
	"github.com/tmc/langchaingo/llms/ollama"
)

type config struct {
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
	Behavior struct {
		ShowPrompt bool `default:"false" split_words:"true"`
	}
	Logger struct {
		HistorySize uint `default:"100"`
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
	r, err := rag.NewChromemRag(cliCfg.Database.Path, rag.ModelPrompts{
		QueryPrefix:     cliCfg.Model.Embedding.PromptPrefix.Query,
		EmbeddingPrefix: cliCfg.Model.Embedding.PromptPrefix.Embedding,
	}, chromem.NewEmbeddingFuncOllama(cliCfg.Model.Embedding.Name, ""))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create rag: %v\n", err)
		os.Exit(1)
	}

	// Load the DB
	log.Println("Loading DB, this may take a bit on the first run...")
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
	appCfg.ShowPromptInChat = cliCfg.Behavior.ShowPrompt
	appCfg.LoggerHistorySize = cliCfg.Logger.HistorySize
	appModel := app.New(appCfg)

	// Swap the RAG logger with one that can hook into the TUI
	r.SetLogger(appModel.Log)

	p := tea.NewProgram(appModel, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithKeyboardEnhancements())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Oof: %v\n", err)
	}
}
