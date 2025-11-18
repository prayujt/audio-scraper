// Package ports defines the interfaces (ports) used by the domain layer.
// These interfaces describe the expected behavior of external systems such as logging
package ports

import (
	"net/http"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type Handlers interface {
	Log() Logger
	HealthHandler(w http.ResponseWriter, r *http.Request)
	Search(w http.ResponseWriter, r *http.Request)
	Download(w http.ResponseWriter, r *http.Request)
}
