package ratelimit

import (
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

type entry struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64
}

type Store struct {
	entries  sync.Map
	ttl      time.Duration
	stopOnce sync.Once
	stopCh   chan struct{}
}

func NewStore(ttl, cleanupInterval time.Duration) *Store {
	s := &Store{
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}
	go s.cleanup(cleanupInterval)
	return s
}

func (s *Store) Limiter(rule, ip string, r rate.Limit, burst int) *rate.Limiter {
	key := rule + ":" + ip
	now := time.Now().Unix()

	if v, ok := s.entries.Load(key); ok {
		e := v.(*entry)
		e.lastSeen.Store(now)
		return e.limiter
	}

	e := &entry{limiter: rate.NewLimiter(r, burst)}
	e.lastSeen.Store(now)
	if actual, loaded := s.entries.LoadOrStore(key, e); loaded {
		ae := actual.(*entry)
		ae.lastSeen.Store(now)
		return ae.limiter
	}
	return e.limiter
}

func (s *Store) sweep() {
	cutoff := time.Now().Add(-s.ttl).Unix()
	s.entries.Range(func(key, value any) bool {
		if value.(*entry).lastSeen.Load() < cutoff {
			s.entries.Delete(key)
		}
		return true
	})
}

func (s *Store) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.sweep()
		}
	}
}

func (s *Store) Stop() {
	s.stopOnce.Do(func() { close(s.stopCh) })
}
