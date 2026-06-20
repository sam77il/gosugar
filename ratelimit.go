package sugar

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type ipWindow struct {
	count    int
	resetAt  time.Time
	lastSeen time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	windows map[string]*ipWindow
	limit   int
	per     time.Duration
}

func newRateLimiter(limit int, per time.Duration) *rateLimiter {
	rl := &rateLimiter{
		windows: make(map[string]*ipWindow),
		limit:   limit,
		per:     per,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) allow(ip string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	w, ok := rl.windows[ip]
	if !ok || now.After(w.resetAt) {
		rl.windows[ip] = &ipWindow{
			count:    1,
			resetAt:  now.Add(rl.per),
			lastSeen: now,
		}
		return true
	}

	w.lastSeen = now
	if w.count >= rl.limit {
		return false
	}
	w.count++
	return true
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		rl.mu.Lock()
		for ip, w := range rl.windows {
			if now.Sub(w.lastSeen) > 3*time.Minute {
				delete(rl.windows, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (s *sugar) RateLimit(limit int, per time.Duration) {
	rl := newRateLimiter(limit, per)
	s.middlewares = append([]*Middleware{{
		URL:      "/*",
		segments: []string{"", "*"},
		starIdx:  1,
		Handler: func(ctx *Context) error {
			ip, _, _ := net.SplitHostPort(ctx.Request.req.RemoteAddr)
			if !rl.allow(ip) {
				http.Error(ctx.Request.writer, "Too Many Requests", http.StatusTooManyRequests)
				return nil
			}
			return ctx.Request.Next()
		},
	}}, s.middlewares...)
}
