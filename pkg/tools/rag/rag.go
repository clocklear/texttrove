package rag

import (
	"context"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/tools"
)

type ragger interface {
	Query(ctx context.Context, queryText string, nResults int, where, whereDocument map[string]any) ([]schema.Document, error)
}

// Tool defines a tool implementation that can search a knowledge base for relevant content.
type Tool struct {
	CallbacksHandler callbacks.Handler
	r                ragger
	nMaxResults      int
	t                prompts.PromptTemplate
}

func New(r ragger, nMaxResults int, template prompts.PromptTemplate) *Tool {
	return &Tool{
		r:           r,
		nMaxResults: nMaxResults,
		t:           template,
	}
}

// Ensure the thing implements the interface
var _ tools.Tool = Tool{}

// Name returns a name for the tool.
func (t Tool) Name() string {
	return "RAG"
}

// Description returns a description for the tool.
func (t Tool) Description() string {
	return `Searches a knowledge base for relevant content.  Input should be a question or topic to search for.`
}

// Call searches a knowledge base for relevant content based on the input question or topic.
func (t Tool) Call(ctx context.Context, input string) (string, error) {
	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolStart(ctx, input)
	}

	// Do the search
	d, err := t.r.Query(ctx, input, t.nMaxResults, nil, nil)
	if err != nil {
		if t.CallbacksHandler != nil {
			t.CallbacksHandler.HandleToolError(ctx, err)
		}
		return "", err
	}

	// Format the results
	content := make([]string, 0, len(d))
	for _, doc := range d {
		content = append(content, doc.PageContent)
	}
	tpl, err := t.t.Format(map[string]interface{}{"contexts": content})
	if err != nil {
		if t.CallbacksHandler != nil {
			t.CallbacksHandler.HandleToolError(ctx, err)
		}
		return "", err
	}

	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolEnd(ctx, tpl)
	}
	return tpl, nil
}
