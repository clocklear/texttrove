package main

// A simple program demonstrating the text area component from the Bubbles
// component library.
// TODO: this is currently a spaghetti mess while I am learning and experimenting.
// This will be cleaned up and refactored into a more maintainable state.

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/clocklear/chromem-go"
	"github.com/clocklear/texttrove/app"
	"github.com/clocklear/texttrove/pkg/db/rag"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/kelseyhightower/envconfig"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

type config struct {
	Model struct {
		Conversation struct {
			Name    string `default:"llama3.2:latest"`
			URL     string `default:"http://localhost:11434"`
			Headers StringMap
			Type    string `default:"ollama"`
		}
		Embedding struct {
			Name         string `default:"mxbai-embed-large:latest"`
			URL          string `default:"http://localhost:11434"`
			PromptPrefix struct {
				Query     string `default:"Represent this sentence for searching relevant passages: "`
				Embedding string
			}
		}
	}
	SystemPromptPath  string `default:"./prompts/system.tpl"`
	ContextPromptPath string `default:"./prompts/context.tpl"`
	Document          struct {
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
	log.Printf("Starting TextTrove, using conversation model server: %v", cliCfg.Model.Conversation.URL)

	// Testing portkey gateway -- create a custom http agent and add some portkey headers
	var c *http.Client
	c = http.DefaultClient
	if len(cliCfg.Model.Conversation.Headers) > 0 {
		t := &StaticHeadersTransport{
			Transport: http.DefaultTransport,
			Headers:   cliCfg.Model.Conversation.Headers,
		}
		c = &http.Client{
			Transport: t,
		}
	}

	// Create a (conversation) LLM
	var conversationLlm llms.Model
	var err error
	switch cliCfg.Model.Conversation.Type {
	case "ollama":
		conversationLlm, err = ollama.New(
			ollama.WithModel(cliCfg.Model.Conversation.Name),
			ollama.WithServerURL(cliCfg.Model.Conversation.URL),
			ollama.WithHTTPClient(c))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create conversation LLM: %v\n", err)
			os.Exit(1)
		}
	case "openai":
		conversationLlm, err = openai.New(
			openai.WithModel(cliCfg.Model.Conversation.Name),
			openai.WithBaseURL(cliCfg.Model.Conversation.URL),
			openai.WithHTTPClient(c))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create conversation LLM: %v\n", err)
			os.Exit(1)
		}
	}
	if conversationLlm == nil {
		fmt.Fprintf(os.Stderr, "Failed to create conversation LLM: unknown type %s\n", cliCfg.Model.Conversation.Type)
		os.Exit(1)
	}

	// Build doc DB
	// TODO: this only supports ollama right now
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
	appCfg.ChatSystemPromptPath = cliCfg.SystemPromptPath
	appCfg.ChatContextPromptPath = cliCfg.ContextPromptPath
	appModel, err := app.New(appCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create app model: %v\n", err)
		os.Exit(1)
	}

	// Swap the RAG logger with one that can hook into the TUI
	r.SetLogger(appModel.Log)

	p := tea.NewProgram(appModel, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithKeyboardEnhancements())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Oof: %v\n", err)
	}
}

// StaticHeadersTransport is a custom RoundTripper that adds a specific set of headers to every request
type StaticHeadersTransport struct {
	Transport http.RoundTripper
	Headers   map[string]string
}

// RoundTrip executes a single HTTP transaction and adds the custom headers
func (t *StaticHeadersTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}
	return t.Transport.RoundTrip(req)
}

type StringMap map[string]string

func (m *StringMap) Decode(value string) error {
	*m = make(map[string]string)
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("invalid map item: %s", pair)
		}
		(*m)[kv[0]] = kv[1]
	}
	return nil
}
