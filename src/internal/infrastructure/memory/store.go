package memory

import (
	"llmclientwrapper/src/internal/domain"
	"sync"
)

// Store is a thread-safe in-memory implementation of domain.MessageStore.
type Store struct {
	mu       sync.Mutex
	messages []domain.Message
}

// NewStore creates an empty in-memory Store.
func NewStore() *Store {
	return &Store{}
}

// Add appends a message to the store.
func (s *Store) Add(msg domain.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
}

// All returns a copy of all stored messages.
func (s *Store) All() []domain.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]domain.Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// Clear removes all messages from the store.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = nil
}
