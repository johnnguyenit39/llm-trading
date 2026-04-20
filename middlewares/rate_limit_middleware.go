package middlewares

import (
	"j_ai_trade/common"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig describes a token-bucket rate limiter.
//
// - Capacity  : max burst of requests allowed at once.
// - Refill    : how often a single token is added back.
// - KeyFunc   : returns the bucket key (defaults to client IP).
type RateLimitConfig struct {
	Capacity int
	Refill   time.Duration
	KeyFunc  func(c *gin.Context) string
}

type bucket struct {
	tokens     int
	lastRefill time.Time
}

type tokenBucketLimiter struct {
	cfg     RateLimitConfig
	mu      sync.Mutex
	buckets map[string]*bucket
}

func newLimiter(cfg RateLimitConfig) *tokenBucketLimiter {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *gin.Context) string { return c.ClientIP() }
	}
	l := &tokenBucketLimiter{
		cfg:     cfg,
		buckets: make(map[string]*bucket),
	}
	go l.cleanup()
	return l
}

func (l *tokenBucketLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		l.buckets[key] = &bucket{tokens: l.cfg.Capacity - 1, lastRefill: now}
		return true
	}

	elapsed := now.Sub(b.lastRefill)
	if l.cfg.Refill > 0 && elapsed > 0 {
		add := int(elapsed / l.cfg.Refill)
		if add > 0 {
			b.tokens += add
			if b.tokens > l.cfg.Capacity {
				b.tokens = l.cfg.Capacity
			}
			b.lastRefill = b.lastRefill.Add(time.Duration(add) * l.cfg.Refill)
		}
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

// cleanup removes buckets that have been idle for a long time to avoid leaks.
func (l *tokenBucketLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-30 * time.Minute)
		l.mu.Lock()
		for k, b := range l.buckets {
			if b.lastRefill.Before(cutoff) {
				delete(l.buckets, k)
			}
		}
		l.mu.Unlock()
	}
}

// RateLimitMiddleware returns a middleware that blocks requests exceeding the
// configured rate, replying with HTTP 429.
func RateLimitMiddleware(cfg RateLimitConfig) gin.HandlerFunc {
	l := newLimiter(cfg)
	return func(c *gin.Context) {
		if !l.allow(l.cfg.KeyFunc(c)) {
			c.JSON(http.StatusTooManyRequests, common.BaseApiResponse[any]{
				HttpRequestStatus: http.StatusTooManyRequests,
				Success:           false,
				Message:           "rate limit exceeded, please slow down",
				Data:              nil,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AuthRateLimitMiddleware enforces a conservative rate on authentication
// endpoints (login / register / OTP) to deter credential-stuffing and spam.
// 20 requests burst, refill 1 token every 3s → ~20/min sustained per IP.
func AuthRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(RateLimitConfig{
		Capacity: 20,
		Refill:   3 * time.Second,
	})
}
