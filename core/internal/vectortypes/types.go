// Package vectortypes defines shared types for vector similarity search.
// It exists to break the import cycle between the store and knowledge packages.
package vectortypes

// VectorStore defines the interface for vector similarity search.
// PGVectorStore in the store package implements this interface.
// When nil, semantic search returns an error.
type VectorStore interface {
	// Upsert stores or replaces an embedding for a knowledge entry ID.
	Upsert(id string, vector []float32, meta map[string]any) error
	// Search returns the top-K most similar entries to queryVector by cosine similarity.
	Search(queryVector []float32, topK int) ([]SearchResult, error)
	// Delete removes the embedding for a knowledge entry ID.
	Delete(id string) error
}

// SearchResult is returned by VectorStore.Search.
type SearchResult struct {
	ID       string         `json:"id"`
	Score    float32        `json:"score"`    // 0.0–1.0 cosine similarity
	Metadata map[string]any `json:"metadata,omitempty"`
}
