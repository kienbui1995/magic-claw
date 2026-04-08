package knowledge

import "github.com/kienbui1995/magic/core/internal/vectortypes"

// VectorStore defines the interface for vector similarity search.
// It is separate from store.Store to keep vector concerns out of the main storage interface.
// PGVectorStore in the store package implements this interface.
// When nil, semantic search returns an error.
type VectorStore = vectortypes.VectorStore

// SearchResult is returned by VectorStore.Search.
type SearchResult = vectortypes.SearchResult
