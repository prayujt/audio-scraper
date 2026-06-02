// Package storemock provides a test double for store.Provider.
package storemock

import (
	"audio-scraper/internal/adapters/store"
)

// Mock implements store.Provider; set the *Func fields to control behavior.
type Mock struct {
	SetFunc      func(key string, choices store.Choices)
	GetFunc      func(key string) (store.Choices, bool)
	DeleteFunc   func(key string)
	ShutdownFunc func()
}

var _ store.Provider = (*Mock)(nil)

func (m *Mock) Set(key string, choices store.Choices) {
	if m.SetFunc != nil {
		m.SetFunc(key, choices)
	}
}

func (m *Mock) Get(key string) (store.Choices, bool) {
	if m.GetFunc != nil {
		return m.GetFunc(key)
	}
	return nil, false
}

func (m *Mock) Delete(key string) {
	if m.DeleteFunc != nil {
		m.DeleteFunc(key)
	}
}

func (m *Mock) Shutdown() {
	if m.ShutdownFunc != nil {
		m.ShutdownFunc()
	}
}
