package storage

import (
	"sync"
)

// MemoryStore is a thread-safe in-memory key-value store.
type MemoryStore struct {
	data map[string]string
	mu   sync.RWMutex
}

// NewMemoryStore creates a new MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]string),
	}
}

// Put adds or updates a key-value pair.
func (s *MemoryStore) Put(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// Get retrieves a value for a key.
func (s *MemoryStore) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// Delete removes a key from the store.
func (s *MemoryStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}
