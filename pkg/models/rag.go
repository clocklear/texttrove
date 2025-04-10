package models

import (
	"context"

	"github.com/tmc/langchaingo/schema"
)

// Ragger describes what we expect to be true of a thing that can RAG documents
type Ragger interface {
	LoadDocuments(ctx context.Context, basePath, filePattern string) error
	Query(ctx context.Context, queryText string, nResults int, where, whereDocument map[string]any) ([]schema.Document, error)
	Shutdown(ctx context.Context) error
}
