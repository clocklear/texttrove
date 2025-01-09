package main

// A simple program demonstrating the text area component from the Bubbles
// component library.
// TODO: this is currently a spaghetti mess while I am learning and experimenting.
// This will be cleaned up and refactored into a more maintainable state.

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clocklear/texttrove/app"
	"github.com/tmc/langchaingo/llms/ollama"
)

const (
	// textAreaHeight    = 5
	conversationModel = "llama3.2-128k:latest" // TODO: Parameterize
)

func main() {
	// Create an LLM
	llm, err := ollama.New(ollama.WithModel(conversationModel))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create LLM: %v\n", err)
		os.Exit(1)
	}

	// Create a new app model
	cfg := app.DefaultConfig()
	cfg.LLM = llm
	appModel := app.New(cfg)

	p := tea.NewProgram(appModel, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Oof: %v\n", err)
	}
}
