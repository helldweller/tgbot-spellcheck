package ratelimit

import (
	"sync"
	"time"
)

type Limiter interface {
	Allow(now time.Time) bool
}

type IntervalLimiter struct {
	mu          sync.Mutex
	minInterval time.Duration
	last        time.Time
}

func NewIntervalLimiter(minInterval time.Duration) *IntervalLimiter {
	return &IntervalLimiter{minInterval: minInterval}
}

// Allow returns true if processing is allowed at "now" and updates internal state.
func (l *IntervalLimiter) Allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.last.IsZero() || now.Sub(l.last) >= l.minInterval {
		l.last = now
		return true
	}
	return false
}
