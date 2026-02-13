package ratelimit

import (
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestStore_LimiterReturnsSameInstance(t *testing.T) {
	s := NewStore(time.Hour, time.Hour)
	defer s.Stop()

	l1 := s.Limiter("login", "1.2.3.4", rate.Every(time.Second), 5)
	l2 := s.Limiter("login", "1.2.3.4", rate.Every(time.Second), 5)

	if l1 != l2 {
		t.Fatal("expected same limiter instance for same rule+ip")
	}
}

func TestStore_LimiterSeparatesRulesAndIPs(t *testing.T) {
	s := NewStore(time.Hour, time.Hour)
	defer s.Stop()

	r := rate.Every(time.Second)
	a := s.Limiter("login", "1.1.1.1", r, 5)
	b := s.Limiter("register", "1.1.1.1", r, 5)
	c := s.Limiter("login", "2.2.2.2", r, 5)

	if a == b {
		t.Fatal("different rules should produce different limiters")
	}
	if a == c {
		t.Fatal("different IPs should produce different limiters")
	}
}

func TestStore_SweepRemovesStaleEntries(t *testing.T) {
	s := NewStore(time.Hour, time.Hour)
	defer s.Stop()

	s.Limiter("login", "1.1.1.1", rate.Every(time.Second), 5)

	// Backdate the entry so it appears stale.
	v, _ := s.entries.Load("login:1.1.1.1")
	v.(*entry).lastSeen.Store(time.Now().Add(-2 * time.Hour).Unix())

	s.sweep()

	_, ok := s.entries.Load("login:1.1.1.1")
	if ok {
		t.Fatal("expected stale entry to be removed by sweep")
	}
}

func TestStore_SweepKeepsFreshEntries(t *testing.T) {
	s := NewStore(time.Hour, time.Hour)
	defer s.Stop()

	s.Limiter("login", "1.1.1.1", rate.Every(time.Second), 5)
	s.sweep()

	_, ok := s.entries.Load("login:1.1.1.1")
	if !ok {
		t.Fatal("expected fresh entry to survive sweep")
	}
}
