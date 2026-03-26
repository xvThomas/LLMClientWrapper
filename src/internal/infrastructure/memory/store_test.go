package memory

import (
	"llmclientwrapper/src/internal/domain"
	"testing"
)

func TestStore_AddAndAll(t *testing.T) {
	s := NewStore()
	s.Add(domain.Message{Role: domain.RoleUser, Content: "hello"})
	s.Add(domain.Message{Role: domain.RoleAssistant, Content: "world"})

	msgs := s.All()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "hello" || msgs[1].Content != "world" {
		t.Error("unexpected message contents")
	}
}

func TestStore_AllReturnsCopy(t *testing.T) {
	s := NewStore()
	s.Add(domain.Message{Role: domain.RoleUser, Content: "original"})

	msgs := s.All()
	msgs[0].Content = "mutated"

	stored := s.All()
	if stored[0].Content != "original" {
		t.Error("All() should return a copy, not a reference")
	}
}

func TestStore_Clear(t *testing.T) {
	s := NewStore()
	s.Add(domain.Message{Role: domain.RoleUser, Content: "hello"})
	s.Clear()

	if len(s.All()) != 0 {
		t.Error("expected empty store after Clear()")
	}
}

func TestStore_EmptyAll(t *testing.T) {
	s := NewStore()
	if msgs := s.All(); len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}
