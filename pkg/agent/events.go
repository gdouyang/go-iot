package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type EventSink interface {
	Emit(event string, payload any)
}

type NopSink struct{}

func (NopSink) Emit(string, any) {}

type SSESink struct {
	w       http.ResponseWriter
	f       http.Flusher
	mu      sync.Mutex
	started bool
}

func NewSSESink(w http.ResponseWriter) *SSESink {
	f, ok := w.(http.Flusher)
	if !ok {
		return nil
	}
	return &SSESink{w: w, f: f}
}

func (s *SSESink) Emit(event string, payload any) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		h := s.w.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		s.w.WriteHeader(http.StatusOK)
		s.started = true
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte(`{}`)
	}
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, raw)
	s.f.Flush()
}

func (s *SSESink) Started() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

func WantsStream(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/event-stream")
}
