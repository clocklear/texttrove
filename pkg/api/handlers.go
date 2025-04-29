package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/clocklear/texttrove/pkg/models"
	"github.com/tmc/langchaingo/llms"
)

type ChatRequest struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type ChatHandler struct {
	rag  models.Ragger
	llm  llms.Model
	chat *models.Chat
}

// MessageEvent represents a structured message event in the SSE stream
type MessageEvent struct {
	Type    string `json:"type"`              // "message" or "chunk"
	Role    string `json:"role,omitempty"`    // "system", "assistant", or "user"
	Content string `json:"content,omitempty"` // The message content
	Chunk   string `json:"chunk,omitempty"`   // For streaming chunks
}

func NewChatHandler(rag models.Ragger, llm llms.Model) *ChatHandler {
	return &ChatHandler{
		rag: rag,
		llm: llm,
	}
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse request body
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a new chat
	chat, err := models.NewChat()
	if err != nil {
		log.Printf("Failed to create chat: %v", err)
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	// Track the last user message index
	lastUserMessageIndex := -1

	// Add messages to chat
	for i, msg := range req.Messages {
		switch msg.Role {
		case "user":
			chat.AppendUserMessage(msg.Content)
			lastUserMessageIndex = i
		case "assistant":
			chat.AppendAssistantMessage(msg.Content)
		case "system":
			chat.AppendSystemMessage(msg.Content)
		}
	}

	// Get the last user message for RAG
	lastUserMessage := ""
	if lastUserMessageIndex >= 0 {
		lastUserMessage = req.Messages[lastUserMessageIndex].Content
	}

	// Create a channel for streaming responses
	stream := make(chan MessageEvent)
	done := make(chan struct{})

	// Start streaming response
	go func() {
		defer close(done)
		chat.BeginStreaming()

		// Query RAG if we have a user message
		if lastUserMessage != "" {
			log.Printf("Querying RAG with message: %s", lastUserMessage)
			ctxs, err := h.rag.Query(r.Context(), lastUserMessage, 5, nil, nil)
			if err != nil {
				log.Printf("Failed to query RAG: %v", err)
				stream <- MessageEvent{Type: "message", Role: "system", Content: fmt.Sprintf("Failed to query RAG: %v", err)}
				return
			}
			log.Printf("Retrieved %d contexts from RAG", len(ctxs))
			if err := chat.AddContexts(ctxs); err != nil {
				log.Printf("Failed to add contexts: %v", err)
				stream <- MessageEvent{Type: "message", Role: "system", Content: fmt.Sprintf("Failed to add contexts: %v", err)}
				return
			}

			// Find the last user message index in the chat log
			messages := chat.Log()
			lastUserMessageIndex := -1
			for i := len(messages) - 1; i >= 0; i-- {
				if messages[i].Role == "user" {
					lastUserMessageIndex = i
					break
				}
			}

			// Stream any system messages added after the last user message
			startIndex := lastUserMessageIndex + 1
			if startIndex < len(messages) {
				for i := startIndex; i < len(messages); i++ {
					msg := messages[i]
					role := "system"
					if msg.Role == "assistant" {
						role = "assistant"
					}
					// Get the text content from the message parts
					var content string
					for _, part := range msg.Parts {
						if textPart, ok := part.(llms.TextContent); ok {
							content += textPart.Text
						}
					}
					stream <- MessageEvent{
						Type:    "message",
						Role:    role,
						Content: content,
					}
				}
			}
		}

		// Generate response
		_, err = h.llm.GenerateContent(r.Context(), chat.Log(),
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				chunkStr := string(chunk)
				log.Printf("Generated chunk: %q", chunkStr)
				select {
				case stream <- MessageEvent{Type: "chunk", Chunk: chunkStr}:
					log.Printf("Sent chunk to stream: %q", chunkStr)
					return nil
				case <-ctx.Done():
					log.Printf("Context cancelled while sending chunk: %q", chunkStr)
					return ctx.Err()
				}
			}),
		)
		if err != nil {
			log.Printf("Failed to generate content: %v", err)
			// Send error as SSE but don't return immediately
			errorMsg, _ := json.Marshal(map[string]string{"error": err.Error()})
			errorStr := string(errorMsg)
			log.Printf("Sending error message: %q", errorStr)
			select {
			case stream <- MessageEvent{Type: "message", Role: "system", Content: errorStr}:
				log.Printf("Error message sent to stream: %q", errorStr)
			case <-r.Context().Done():
				log.Printf("Context cancelled while sending error message")
				return
			}
		}

		chat.EndStreaming()
	}()

	// Stream SSE responses
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("Streaming not supported by response writer")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Create a ticker for periodic flushing
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-stream:
			// Marshal the event to JSON
			eventJSON, err := json.Marshal(event)
			if err != nil {
				log.Printf("Failed to marshal event: %v", err)
				continue
			}

			// Write SSE data
			log.Printf("Writing SSE data: %q", string(eventJSON))
			_, err = fmt.Fprintf(w, "data: %s\n\n", string(eventJSON))
			if err != nil {
				log.Printf("Failed to write SSE data: %v", err)
				return
			}
			flusher.Flush()
			log.Printf("Flushed SSE data: %q", string(eventJSON))
		case <-ticker.C:
			// Periodic flush to keep connection alive
			log.Printf("Periodic flush")
			flusher.Flush()
		case <-done:
			log.Printf("Streaming completed")
			return
		case <-r.Context().Done():
			log.Printf("Request context cancelled")
			return
		case <-time.After(5 * time.Minute):
			log.Printf("Streaming timeout after 5 minutes")
			return
		}
	}
}
