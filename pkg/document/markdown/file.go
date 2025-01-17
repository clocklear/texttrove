package markdown

import (
	"bytes"
	"context"
	"os"
	"path"

	"github.com/adrg/frontmatter"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

// Load converts a markdown file into a slice of schema.Document.
func Load(ctx context.Context, basePath, relPath string) ([]schema.Document, error) {
	// Read the contents of path into a string.
	contents, err := os.ReadFile(path.Join(basePath, relPath))
	if err != nil {
		return nil, err
	}

	// Parse any potential frontmatter
	matter := make(map[string]any)
	rest, err := frontmatter.Parse(bytes.NewReader(contents), &matter)
	if err != nil {
		return nil, err
	}

	// Add relative path elements as metadata for context
	matter["Source"] = relPath

	// Parse (split) the markdown file into a slice of schema.Document.
	splitter := textsplitter.NewMarkdownTextSplitter(textsplitter.WithChunkSize(300), textsplitter.WithChunkOverlap(32), textsplitter.WithHeadingHierarchy(true))
	return textsplitter.CreateDocuments(splitter, []string{string(rest)}, []map[string]any{matter})
}
