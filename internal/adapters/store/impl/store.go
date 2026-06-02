// Package storeimpl implements the store.Provider port as an in-memory cache
// with TTL-based cleanup.
package storeimpl

import (
	"sync"
	"time"

	"audio-scraper/internal/adapters/store"
	"audio-scraper/internal/logger"
)

const cleanupInterval = 30 * time.Minute
const storeTTL = 10 * time.Minute

type choiceItem struct {
	choices   store.Choices
	timestamp time.Time
}

// Store is the in-memory request cache.
type Store struct {
	log         logger.Logger
	requestData map[string]choiceItem
	mu          sync.RWMutex
	done        chan struct{}
}

var _ store.Provider = (*Store)(nil)

// New returns an in-memory request cache and starts its cleanup routine.
func New(l logger.Logger) *Store {
	s := &Store{
		log:         l,
		requestData: make(map[string]choiceItem),
		done:        make(chan struct{}),
	}

	go s.cleanupRoutine()
	return s
}

func (s *Store) Set(key string, choices store.Choices) {
	s.log.Info("storing data in store", "key", key)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestData[key] = choiceItem{
		choices:   choices,
		timestamp: time.Now(),
	}
}

func (s *Store) Get(key string) (store.Choices, bool) {
	s.log.Info("retrieving data from store", "key", key)
	s.mu.RLock()
	defer s.mu.RUnlock()
	choices, exists := s.requestData[key]
	return choices.choices, exists
}

func (s *Store) Delete(key string) {
	s.log.Info("deleting data from store", "key", key)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.requestData, key)
}

func (s *Store) cleanupRoutine() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.purgeExpiredKeys()
		case <-s.done:
			return
		}
	}
}

func (s *Store) purgeExpiredKeys() {
	s.log.Debug("running scheduled cleanup of expired keys")
	cutoff := time.Now().Add(-storeTTL)

	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for key, item := range s.requestData {
		if item.timestamp.Before(cutoff) {
			count++
			s.log.Info("removing expired key", "key", key, "age", time.Since(item.timestamp))
			delete(s.requestData, key)
		}
	}
	s.log.Info("cleanup complete", "removed_keys", count)
}

func (s *Store) Shutdown() {
	close(s.done)
}
