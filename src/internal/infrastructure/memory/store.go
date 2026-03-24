package memory

import (
	"llmclientwrapper/src/internal"
	"sync"
)

// Store is a thread-safe in-memory implementation of internal.MessageStore.
type Store struct {
	mu       sync.Mutex
	messages []internal.Message
}

// NewStore creates an empty in-memory Store.
func NewStore() *Store {
	return &Store{}
}

// Add appends a message to the store.
func (s *Store) Add(msg internal.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
}

// All returns a copy of all stored messages.
func (s *Store) All() []internal.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]internal.Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// Clear removes all messages from the store.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = nil
}
