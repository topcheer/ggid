package detection

import (
	"context"
	"time"
)

// MemStateStore is an in-memory StateStore for testing and dev.
// Uses maps with simple TTL-based cleanup.
type MemStateStore struct {
	data map[string][]stateEntry
	counts map[string]int64
}

type stateEntry struct {
	ts     int64
	member string
}

// NewMemStateStore creates an in-memory state store.
func NewMemStateStore() *MemStateStore {
	return &MemStateStore{
		data:   make(map[string][]stateEntry),
		counts: make(map[string]int64),
	}
}

func (s *MemStateStore) AddEvent(_ context.Context, key string, ts int64, member string, _ time.Duration) error {
	s.data[key] = append(s.data[key], stateEntry{ts: ts, member: member})
	return nil
}

func (s *MemStateStore) EventsSince(_ context.Context, key string, since int64) ([]string, error) {
	var result []string
	for _, e := range s.data[key] {
		if e.ts >= since {
			result = append(result, e.member)
		}
	}
	return result, nil
}

func (s *MemStateStore) Incr(_ context.Context, key string, _ time.Duration) (int64, error) {
	s.counts[key]++
	return s.counts[key], nil
}
