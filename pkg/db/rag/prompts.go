package rag

// ModelPrompts represents the model-specific prompts that are required to interact with a given model.
type ModelPrompts struct {
	// EmbeddingPrefix is the prefix used when passing documents to the embedding model.  Not all models require this.
	EmbeddingPrefix string

	// QueryPrefix is the prefix used when passing queries to the embedding model.  Not all models require this.
	QueryPrefix string
}
