package rag

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/clocklear/texttrove/pkg/document/markdown"

	"github.com/philippgille/chromem-go"
	"github.com/tmc/langchaingo/schema"
)

type ChromemRag struct {
	db      *chromem.DB
	col     *chromem.Collection
	prompts ModelPrompts
}

func NewChromemRag(dbPath string, prompts ModelPrompts, embedding chromem.EmbeddingFunc) (*ChromemRag, error) {
	db, err := chromem.NewPersistentDB(dbPath, true)
	if err != nil {
		return nil, err
	}
	col, err := db.GetOrCreateCollection("texttrove", nil, embedding)
	if err != nil {
		return nil, err
	}
	return &ChromemRag{
		db:      db,
		col:     col,
		prompts: prompts,
	}, nil
}

func (r *ChromemRag) LoadDocuments(ctx context.Context, basePath, filePattern string) error {
	// Use the given basePath and filePattern to find matching files
	var matches []string
	err := filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			matched, err := filepath.Match(filePattern, filepath.Base(path))
			if err != nil {
				return err
			}
			if matched {
				matches = append(matches, path)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// For each file, parse and build collection
	var docs []chromem.Document
	for _, match := range matches {
		// Strip the basepath off the beginning of the match
		relPath := match[len(basePath):]
		doc, err := markdown.Load(ctx, basePath, relPath)
		if err != nil {
			// log.Printf("Failed to load document %s: %v\n", match, err)
			continue
		}

		// Convert the schema.document(s) into chromem.document(s)
		bLoaded := false
		for _, d := range doc {
			docId := sha256Hash(d.PageContent)
			// Is thing already in the DB?
			exists, err := r.docExistsInDB(ctx, docId)
			if err != nil {
				log.Printf("Failed to check if document %s exists in DB: %v\n", docId, err)
				continue
			}
			if exists {
				// log.Printf("Document %s already exists in DB, skipping...\n", docId)
				continue
			}
			if !bLoaded {
				log.Printf("loading: %s\n", match)
				bLoaded = true
			}

			md := stringifyMetadata(d.Metadata)
			docs = append(docs, chromem.Document{
				Content:  r.prompts.EmbeddingPrefix + d.PageContent,
				Metadata: md,
				ID:       docId, // TODO: Figure out how to do ID in such a way that we can smartly update documents
			})
		}
	}

	if len(docs) == 0 {
		// nothing to do
		return nil
	}

	// Add the raw collection to the DB
	log.Println("Adding docs to DB...")
	err = r.col.AddDocuments(ctx, docs, runtime.NumCPU())
	log.Println("Done adding docs to DB.")
	return err
}

func (r *ChromemRag) Query(ctx context.Context, queryText string, nResults int, where, whereDocument map[string]any) ([]schema.Document, error) {
	// Convert the metadata maps
	whereString := stringifyMetadata(where)
	whereDocumentString := stringifyMetadata(whereDocument)
	res, err := r.col.Query(ctx, r.prompts.QueryPrefix+queryText, nResults, whereString, whereDocumentString)
	if err != nil {
		return nil, err
	}

	// Convert the chromem.Document(s) into schema.Document(s)
	var docs []schema.Document
	for _, d := range res {
		// Convert metadata into a map[string]any
		metadata := make(map[string]any)
		for k, v := range d.Metadata {
			metadata[k] = v
		}
		docs = append(docs, schema.Document{
			PageContent: d.Content,
			Metadata:    metadata,
			Score:       d.Similarity,
		})
	}

	return docs, nil
}

func (r *ChromemRag) docExistsInDB(ctx context.Context, id string) (bool, error) {
	_, err := r.col.GetByID(ctx, id)
	return err == nil, nil
}

func stringifyMetadata(m map[string]any) map[string]string {
	sm := make(map[string]string)
	for k, v := range m {
		sm[k] = fmt.Sprintf("%v", v)
	}
	return sm
}

func sha256Hash(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
