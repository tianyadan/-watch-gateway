package bus

import (
	"sync"
	"time"
)

// RateLimiter 进程内滑动窗口限流：同一 key 在 window 内最多 limit 次。
type RateLimiter struct {
	limit  int
	window time.Duration

	mu   sync.Mutex
	hits map[string][]time.Time
}

// NewRateLimiter 创建限流器。
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:  limit,
		window: window,
		hits:   make(map[string][]time.Time),
	}
}

// Allow 若未超限返回 true，并记录本次请求。
func (l *RateLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-l.window)
	old := l.hits[key]
	recent := make([]time.Time, 0, len(old)+1)
	for _, t := range old {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= l.limit {
		l.hits[key] = recent
		return false
	}

	l.hits[key] = append(recent, now)
	return true
}
