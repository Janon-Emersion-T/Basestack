package auth

import (
	"sync"
	"time"
)

type Limiter interface {
	Allow(key string, limit int) (bool, time.Duration)
}
type bucket struct {
	count int
	until time.Time
}
type MemoryLimiter struct {
	mu      sync.Mutex
	buckets map[string]bucket
	now     func() time.Time
	maximum int
}

func NewLimiter() *MemoryLimiter {
	return &MemoryLimiter{buckets: map[string]bucket{}, now: time.Now, maximum: 10000}
}
func (l *MemoryLimiter) Allow(key string, limit int) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, exists := l.buckets[key]
	if !exists || !now.Before(b.until) {
		if len(l.buckets) >= l.maximum {
			for k, v := range l.buckets {
				if !now.Before(v.until) {
					delete(l.buckets, k)
				}
			}
			if len(l.buckets) >= l.maximum {
				return false, time.Minute
			}
		}
		b = bucket{until: now.Add(time.Minute)}
	}
	if b.count >= limit {
		return false, b.until.Sub(now)
	}
	b.count++
	l.buckets[key] = b
	return true, 0
}
