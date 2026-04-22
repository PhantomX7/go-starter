package middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*ipLimiter
	rate     rate.Limit
	burst    int
	ttl      time.Duration
}

func newIPRateLimiter(r rate.Limit, burst int, ttl time.Duration) *ipRateLimiter {
	l := &ipRateLimiter{
		visitors: make(map[string]*ipLimiter),
		rate:     r,
		burst:    burst,
		ttl:      ttl,
	}
	go l.cleanupLoop()
	return l
}

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	v, ok := l.visitors[ip]
	if !ok {
		v = &ipLimiter{limiter: rate.NewLimiter(l.rate, l.burst)}
		l.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	l.mu.Unlock()
	return v.limiter.Allow()
}

func (l *ipRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.ttl)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-l.ttl)
		l.mu.Lock()
		for ip, v := range l.visitors {
			if v.lastSeen.Before(cutoff) {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}

// AuthRateLimiter returns a per-client-IP token-bucket limiter sized for auth
// endpoints (login/register/refresh). Refills at 1 req/sec with a burst of 5 —
// tight enough to blunt credential stuffing, loose enough for legitimate retries.
func (m *Middleware) AuthRateLimiter() gin.HandlerFunc {
	limiter := newIPRateLimiter(rate.Every(time.Second), 5, 10*time.Minute)
	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP()) {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, response.BuildResponseFailed("too many requests"))
			return
		}
		c.Next()
	}
}
