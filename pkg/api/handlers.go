package api

import (
	"context"
	"encoding/json"
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

func NewChatHandler(rag models.Ragger, llm llms.Model) *ChatHandler {
	return &ChatHandler{
		rag: rag,
		llm: llm,
	}
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a new chat
	chat, err := models.NewChat()
	if err != nil {
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	// Add messages to chat
	for _, msg := range req.Messages {
		switch msg.Role {
		case "user":
			chat.AppendUserMessage(msg.Content)
		case "assistant":
			chat.AppendAssistantMessage(msg.Content)
		case "system":
			chat.AppendSystemMessage(msg.Content)
		}
	}

	// Get the last user message for RAG
	lastUserMessage := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastUserMessage = req.Messages[i].Content
			break
		}
	}

	// Query RAG if we have a user message
	if lastUserMessage != "" {
		ctxs, err := h.rag.Query(r.Context(), lastUserMessage, 5, nil, nil)
		if err != nil {
			http.Error(w, "Failed to query RAG", http.StatusInternalServerError)
			return
		}
		if err := chat.AddContexts(ctxs); err != nil {
			http.Error(w, "Failed to add contexts", http.StatusInternalServerError)
			return
		}
	}

	// Create a channel for streaming responses
	stream := make(chan string)
	done := make(chan struct{})

	// Start streaming response
	go func() {
		defer close(done)
		chat.BeginStreaming()

		// Generate response
		_, err := h.llm.GenerateContent(r.Context(), chat.Log(),
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				stream <- string(chunk)
				return nil
			}),
		)
		if err != nil {
			// Send error as SSE
			stream <- "data: {\"error\":\"" + err.Error() + "\"}\n\n"
			return
		}

		chat.EndStreaming()
	}()

	// Stream SSE responses
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case chunk := <-stream:
			// Write SSE data
			_, err := w.Write([]byte("data: " + chunk + "\n\n"))
			if err != nil {
				return
			}
			flusher.Flush()
		case <-done:
			return
		case <-r.Context().Done():
			return
		case <-time.After(30 * time.Second):
			// Timeout after 30 seconds
			return
		}
	}
}
