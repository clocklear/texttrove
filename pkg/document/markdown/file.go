package markdown

import (
	"bytes"
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"

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

	// Add relative path elements as metadata for context; split on path separators
	matter["doc_path"] = splitPathComponents(relPath)

	// Parse (split) the markdown file into a slice of schema.Document.
	splitter := textsplitter.NewMarkdownTextSplitter(textsplitter.WithChunkSize(300), textsplitter.WithChunkOverlap(32), textsplitter.WithHeadingHierarchy(true))
	return textsplitter.CreateDocuments(splitter, []string{string(rest)}, []map[string]any{matter})
}

func splitPathComponents(path string) []string {
	var components []string
	for {
		dir, file := filepath.Split(path)
		if file != "" {
			components = append([]string{file}, components...)
		}
		if dir == "" || dir == "/" {
			if dir == "/" {
				components = append([]string{"/"}, components...)
			}
			break
		}
		path = strings.TrimSuffix(dir, "/")
	}
	return components
}
