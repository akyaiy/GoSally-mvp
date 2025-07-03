package logs

import (
	"context"
	"log/slog"
	"sync"
)

type MockHandler struct {
	mu   sync.Mutex
	Logs []slog.Record
}

func NewMockHandler() *MockHandler {
	return &MockHandler{}
}

func (h *MockHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *MockHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Logs = append(h.Logs, r.Clone())
	return nil
}

func (h *MockHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *MockHandler) WithGroup(_ string) slog.Handler {
	return h
}
