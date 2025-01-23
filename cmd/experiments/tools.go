package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/clocklear/texttrove/pkg/tools/date"

	"github.com/kelseyhightower/envconfig"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/tools"
)

type config struct {
	Model struct {
		Conversation string `default:"llama3.2:latest"`
	}
}

// This example demonstrates how to use a tool to help an LLM answer a question.
// The first question is one that the LLM shouldn't be able to answer, and the second question is one that the LLM should be able to answer with the help of a tool.
func main() {
	// Load config from environment (using envconfig)
	var cliCfg config
	envconfig.MustProcess("", &cliCfg)

	// Create a (conversation) LLM
	llm, err := ollama.New(ollama.WithModel(cliCfg.Model.Conversation))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create conversation LLM: %v\n", err)
		os.Exit(1)
	}

	streamingConsoleOut := func(ctx context.Context, chunk []byte) error {
		fmt.Print(string(chunk))
		return nil
	}

	// Ask the question that the LLM shouldn't be able to answer
	q := "Find yesterday's date."
	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are a helpful assistant."),
		llms.TextParts(llms.ChatMessageTypeHuman, q),
	}
	fmt.Printf("Asking (no tools): %s\n", q)
	completion, err := llm.GenerateContent(context.TODO(), content, llms.WithStreamingFunc(streamingConsoleOut))
	if err != nil {
		log.Fatal(err)
	}
	_ = completion

	// Should yield something like:
	// I'm not aware of the current date, as I'm a large language model, I don't have real-time access to the current date and time. However, I can suggest ways for you to find out the current date.
	// You can check your device's clock or calendar app, or search online for "current date" to get the latest information.

	fmt.Println("\n\n----\n")

	// Let's do the same thing, but with a tool introduced that can help with today's date
	agentTools := []tools.Tool{
		date.New(),
	}

	// Initialize the agent
	// TODO: Determine why this doesn't work with the oneshotagent and llama3.2
	agent := agents.NewConversationalAgent(llm,
		agentTools,
		agents.WithMaxIterations(3),
	)
	executor := agents.NewExecutor(agent)

	// run a chain with the executor and defined input
	fmt.Printf("Asking (with tools): %s\n", q)
	answer, err := chains.Run(context.Background(), executor, q)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(answer)
	// Should yield todays date
}
