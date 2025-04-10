package api

import (
	"context"
	"net/http"

	"github.com/clocklear/texttrove/pkg/models"
	"github.com/tmc/langchaingo/llms"
)

type Server struct {
	server *http.Server
	rag    models.Ragger
	llm    llms.Model
}

func NewServer(addr string, rag models.Ragger, llm llms.Model) *Server {
	mux := http.NewServeMux()
	chatHandler := NewChatHandler(rag, llm)
	mux.Handle("/chat", chatHandler)

	return &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		rag: rag,
		llm: llm,
	}
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
