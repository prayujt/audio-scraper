// Package logger implements the logging functionality using slog.
package logger

import (
	"log/slog"
	"os"

	"audio-scraper/internal/ports"
)

type Logger struct {
	l *slog.Logger
}

func NewLogger() ports.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	return &Logger{l: logger}
}

func (s *Logger) Debug(msg string, args ...any) {
	s.l.Debug(msg, args...)
}

func (s *Logger) Info(msg string, args ...any) {
	s.l.Info(msg, args...)
}

func (s *Logger) Warn(msg string, args ...any) {
	s.l.Warn(msg, args...)
}

func (s *Logger) Error(msg string, args ...any) {
	s.l.Error(msg, args...)
}

func (s *Logger) With(args ...any) ports.Logger {
	return &Logger{l: s.l.With(args...)}
}
