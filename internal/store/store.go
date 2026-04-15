package store

import (
	"sync"
	"time"
)

type ProofResult struct {
	ID            string
	ProofJSON     []byte
	PublicSignals []string
	Valid         bool
	Error         string
	ReceivedAt    time.Time
}

type Store struct {
	mu      sync.RWMutex
	results map[string]*ProofResult
}

func New() *Store {
	return &Store{results: make(map[string]*ProofResult)}
}

func (s *Store) Save(r *ProofResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[r.ID] = r
}

func (s *Store) Get(id string) (*ProofResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.results[id]
	return r, ok
}
