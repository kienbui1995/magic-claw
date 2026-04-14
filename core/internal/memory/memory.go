// Package memory provides agent conversation memory with short-term
// (recent turns) and long-term (vector search) recall, scoped per session.
package memory

import (
	"sync"
	"time"
)

// Turn represents a single conversation turn.
type Turn struct {
	Role      string    `json:"role"` // user, assistant, system
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Session holds conversation state for one agent/user session.
type Session struct {
	ID        string  `json:"id"`
	AgentID   string  `json:"agent_id"`
	Turns     []Turn  `json:"turns"`
	MaxTurns  int     `json:"max_turns"` // sliding window size
	CreatedAt time.Time `json:"created_at"`
}

// VectorEntry is a long-term memory item stored with an embedding.
type VectorEntry struct {
	ID        string            `json:"id"`
	AgentID   string            `json:"agent_id"`
	Content   string            `json:"content"`
	Embedding []float32         `json:"embedding,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Score     float64           `json:"score,omitempty"` // similarity score from search
}

// VectorStore is the interface for long-term memory persistence.
type VectorStore interface {
	Upsert(id string, embedding []float32, metadata map[string]any) error
	Search(query []float32, topK int) ([]VectorResult, error)
}

// VectorResult from a similarity search.
type VectorResult struct {
	ID       string         `json:"id"`
	Score    float64        `json:"score"`
	Metadata map[string]any `json:"metadata"`
}

// Store manages agent memory (short-term + long-term).
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session // session ID -> session
	entries  map[string][]*VectorEntry // agent ID -> entries
	vectors  VectorStore // optional, for semantic search
}

// NewStore creates a new memory store. vectorStore can be nil.
func NewStore(vs VectorStore) *Store {
	return &Store{
		sessions: make(map[string]*Session),
		entries:  make(map[string][]*VectorEntry),
		vectors:  vs,
	}
}

// GetOrCreateSession returns an existing session or creates a new one.
func (s *Store) GetOrCreateSession(sessionID, agentID string, maxTurns int) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[sessionID]; ok {
		return sess
	}
	if maxTurns <= 0 {
		maxTurns = 50
	}
	sess := &Session{
		ID:        sessionID,
		AgentID:   agentID,
		MaxTurns:  maxTurns,
		CreatedAt: time.Now().UTC(),
	}
	s.sessions[sessionID] = sess
	return sess
}

// AddTurn appends a turn to a session, trimming to max window.
func (s *Store) AddTurn(sessionID string, turn Turn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		return
	}
	if turn.Timestamp.IsZero() {
		turn.Timestamp = time.Now().UTC()
	}
	sess.Turns = append(sess.Turns, turn)
	if len(sess.Turns) > sess.MaxTurns {
		sess.Turns = sess.Turns[len(sess.Turns)-sess.MaxTurns:]
	}
}

// GetTurns returns recent turns for a session.
func (s *Store) GetTurns(sessionID string, lastN int) []Turn {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}
	turns := sess.Turns
	if lastN > 0 && lastN < len(turns) {
		turns = turns[len(turns)-lastN:]
	}
	result := make([]Turn, len(turns))
	copy(result, turns)
	return result
}

// AddEntry stores a long-term memory entry for an agent.
func (s *Store) AddEntry(entry *VectorEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[entry.AgentID] = append(s.entries[entry.AgentID], entry)
}

// SearchEntries returns entries for an agent. If vector store is available
// and query embedding is provided, uses semantic search; otherwise returns recent entries.
func (s *Store) SearchEntries(agentID string, queryEmbedding []float32, topK int) []*VectorEntry {
	if topK <= 0 {
		topK = 5
	}

	// Try vector search if available
	if s.vectors != nil && len(queryEmbedding) > 0 {
		results, err := s.vectors.Search(queryEmbedding, topK)
		if err == nil && len(results) > 0 {
			var entries []*VectorEntry
			for _, r := range results {
				entries = append(entries, &VectorEntry{
					ID:      r.ID,
					AgentID: agentID,
					Content: metaString(r.Metadata, "content"),
					Score:   r.Score,
				})
			}
			return entries
		}
	}

	// Fallback: return most recent entries
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := s.entries[agentID]
	if len(all) <= topK {
		result := make([]*VectorEntry, len(all))
		copy(result, all)
		return result
	}
	result := make([]*VectorEntry, topK)
	copy(result, all[len(all)-topK:])
	return result
}

// ListSessions returns all session IDs for an agent.
func (s *Store) ListSessions(agentID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var ids []string
	for _, sess := range s.sessions {
		if sess.AgentID == agentID {
			ids = append(ids, sess.ID)
		}
	}
	return ids
}

func metaString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
