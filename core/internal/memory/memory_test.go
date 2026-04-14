package memory

import "testing"

func TestConversationMemory(t *testing.T) {
	s := NewStore(nil)
	sess := s.GetOrCreateSession("s1", "agent1", 3)
	if sess.ID != "s1" {
		t.Fatal("wrong session ID")
	}

	s.AddTurn("s1", Turn{Role: "user", Content: "hello"})
	s.AddTurn("s1", Turn{Role: "assistant", Content: "hi"})
	s.AddTurn("s1", Turn{Role: "user", Content: "how are you"})
	s.AddTurn("s1", Turn{Role: "assistant", Content: "good"})

	// MaxTurns=3, should trim oldest
	turns := s.GetTurns("s1", 0)
	if len(turns) != 3 {
		t.Errorf("expected 3 turns (trimmed), got %d", len(turns))
	}
	if turns[0].Content != "hi" {
		t.Errorf("oldest should be 'hi', got %q", turns[0].Content)
	}
}

func TestGetTurns_LastN(t *testing.T) {
	s := NewStore(nil)
	s.GetOrCreateSession("s1", "a1", 100)
	s.AddTurn("s1", Turn{Role: "user", Content: "1"})
	s.AddTurn("s1", Turn{Role: "user", Content: "2"})
	s.AddTurn("s1", Turn{Role: "user", Content: "3"})

	turns := s.GetTurns("s1", 2)
	if len(turns) != 2 {
		t.Errorf("expected 2, got %d", len(turns))
	}
	if turns[0].Content != "2" {
		t.Errorf("expected '2', got %q", turns[0].Content)
	}
}

func TestLongTermMemory(t *testing.T) {
	s := NewStore(nil)
	s.AddEntry(&VectorEntry{ID: "e1", AgentID: "a1", Content: "fact 1"})
	s.AddEntry(&VectorEntry{ID: "e2", AgentID: "a1", Content: "fact 2"})
	s.AddEntry(&VectorEntry{ID: "e3", AgentID: "a1", Content: "fact 3"})

	entries := s.SearchEntries("a1", nil, 2)
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestListSessions(t *testing.T) {
	s := NewStore(nil)
	s.GetOrCreateSession("s1", "a1", 10)
	s.GetOrCreateSession("s2", "a1", 10)
	s.GetOrCreateSession("s3", "a2", 10)

	ids := s.ListSessions("a1")
	if len(ids) != 2 {
		t.Errorf("expected 2 sessions for a1, got %d", len(ids))
	}
}
