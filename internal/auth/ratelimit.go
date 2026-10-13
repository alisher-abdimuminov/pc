package auth

import (
	"sync"
	"time"
)

// LoginLimiter is an in-memory sliding-window limiter keyed by "ip|username".
type LoginLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	attempts map[string][]time.Time
	now      func() time.Time
}

func NewLoginLimiter(limit int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{limit: limit, window: window, attempts: map[string][]time.Time{}, now: time.Now}
}

func (l *LoginLimiter) prune(key string, now time.Time) []time.Time {
	kept := l.attempts[key][:0]
	for _, t := range l.attempts[key] {
		if now.Sub(t) < l.window {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.attempts, key)
	} else {
		l.attempts[key] = kept
	}
	return kept
}

// Allowed reports whether another attempt may be made for key.
func (l *LoginLimiter) Allowed(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.prune(key, l.now())) < l.limit
}

// RecordFailure counts a failed attempt.
func (l *LoginLimiter) RecordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.prune(key, now)
	l.attempts[key] = append(l.attempts[key], now)
}

// Reset clears the counter after a successful login.
func (l *LoginLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}
