package providers

import (
	"sync"
	"time"

	"audio-scraper/internal/models"
	"audio-scraper/internal/ports"
)

const cleanupInterval = 30 * time.Minute
const storeTTL = 10 * time.Minute

type choiceItem struct {
	choices   []models.Choice
	timestamp time.Time
}

type storeClient struct {
	log         ports.Logger
	requestData map[string]choiceItem
	mu          sync.RWMutex
	done        chan struct{}
}

func NewStoreProvider(l ports.Logger) ports.StoreProvider {
	store := &storeClient{
		log:         l,
		requestData: make(map[string]choiceItem),
		done:        make(chan struct{}),
	}

	go store.cleanupRoutine()
	return store
}

func (s *storeClient) Set(key string, choices []models.Choice) {
	s.log.Info("storing data in store", "key", key)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestData[key] = choiceItem{
		choices:   choices,
		timestamp: time.Now(),
	}
}

func (s *storeClient) Get(key string) ([]models.Choice, bool) {
	s.log.Info("retrieving data from store", "key", key)
	s.mu.RLock()
	defer s.mu.RUnlock()
	choices, exists := s.requestData[key]
	return choices.choices, exists
}

func (s *storeClient) Delete(key string) {
	s.log.Info("deleting data from store", "key", key)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.requestData, key)
}

func (s *storeClient) cleanupRoutine() {
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

func (s *storeClient) purgeExpiredKeys() {
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

func (s *storeClient) Shutdown() {
	close(s.done)
}
