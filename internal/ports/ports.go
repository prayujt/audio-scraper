// Package ports defines the interfaces (ports) used by the domain layer.
// These interfaces describe the expected behavior of external systems such as logging
package ports

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
