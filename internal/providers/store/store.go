package store

import (
	"sync"
	"time"

	"audio-scraper/internal/constants"
	"audio-scraper/internal/logger"
)

const cleanupInterval = 30 * time.Minute
const storeTTL = 10 * time.Minute

type Choice struct {
	Type  constants.SpotifyEntityType `json:"type"`
	ID    string                      `json:"id"`
	Label string                      `json:"label"`
}

type Choices []Choice

func (choices Choices) FindByLabel(label string) *Choice {
	for _, choice := range choices {
		if choice.Label == label {
			return &choice
		}
	}
	return nil
}

type choiceItem struct {
	choices   Choices
	timestamp time.Time
}

type Store struct {
	log         logger.Logger
	requestData map[string]choiceItem
	mu          sync.RWMutex
	done        chan struct{}
}

func NewStoreProvider(l logger.Logger) *Store {
	store := &Store{
		log:         l,
		requestData: make(map[string]choiceItem),
		done:        make(chan struct{}),
	}

	go store.cleanupRoutine()
	return store
}

func (s *Store) Set(key string, choices Choices) {
	s.log.Info("storing data in store", "key", key)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestData[key] = choiceItem{
		choices:   choices,
		timestamp: time.Now(),
	}
}

func (s *Store) Get(key string) (Choices, bool) {
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
