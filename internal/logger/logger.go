// Package logger implements the logging functionality using slog.
package logger

import (
	"log/slog"
	"os"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

type logger struct {
	l *slog.Logger
}

func NewLogger() Logger {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	return &logger{l: log}
}

func (s *logger) Debug(msg string, args ...any) {
	s.l.Debug(msg, args...)
}

func (s *logger) Info(msg string, args ...any) {
	s.l.Info(msg, args...)
}

func (s *logger) Warn(msg string, args ...any) {
	s.l.Warn(msg, args...)
}

func (s *logger) Error(msg string, args ...any) {
	s.l.Error(msg, args...)
}

func (s *logger) With(args ...any) Logger {
	return &logger{l: s.l.With(args...)}
}
