package storage

import "sync"

// Store tracks processed messages to avoid re-processing.
type Store interface {
	WasProcessed(chatID int64, messageID int) bool
	MarkProcessed(chatID int64, messageID int)
}

type InMemoryStore struct {
	mu   sync.RWMutex
	data map[int64]map[int]struct{}
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{data: make(map[int64]map[int]struct{})}
}

func (s *InMemoryStore) WasProcessed(chatID int64, messageID int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.data[chatID]
	if !ok {
		return false
	}
	_, exists := m[messageID]
	return exists
}

func (s *InMemoryStore) MarkProcessed(chatID int64, messageID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.data[chatID]
	if !ok {
		m = make(map[int]struct{})
		s.data[chatID] = m
	}
	m[messageID] = struct{}{}
}
