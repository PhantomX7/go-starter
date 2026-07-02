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

// rateLimitHandler wraps an ipRateLimiter in the shared reject-or-continue
// Gin handler used by every limiter constructor below.
func rateLimitHandler(limiter *ipRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP()) {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, response.BuildResponseFailed("too many requests"))
			return
		}
		c.Next()
	}
}

// AuthRateLimiter returns a per-client-IP token-bucket limiter sized for
// credential-bearing auth endpoints (login/register). Refills at 1 req/sec
// with a burst of 5 — tight enough to blunt credential stuffing, loose enough
// for legitimate retries. Each call returns an INDEPENDENT limiter instance,
// so give every route its own: sharing one across endpoints lets traffic on
// one endpoint exhaust the budget of another.
func (m *Middleware) AuthRateLimiter() gin.HandlerFunc {
	return rateLimitHandler(newIPRateLimiter(rate.Every(time.Second), 5, 10*time.Minute))
}

// RefreshRateLimiter returns a per-client-IP limiter for /auth/refresh.
// Notably looser than AuthRateLimiter (1 req/sec, burst 20): refresh carries
// an unguessable bearer token rather than guessable credentials, and
// legitimate clients refresh routinely — several tabs or devices behind one
// NAT can plausibly burst well past the login budget without being abusive.
func (m *Middleware) RefreshRateLimiter() gin.HandlerFunc {
	return rateLimitHandler(newIPRateLimiter(rate.Every(time.Second), 20, 10*time.Minute))
}

// AdminRateLimiter returns a per-client-IP limiter for the /admin surface.
// Far looser than the auth limiter (10 req/sec, burst 20) — it exists to slow
// scripted abuse of mutating endpoints, not to throttle normal dashboard use.
func (m *Middleware) AdminRateLimiter() gin.HandlerFunc {
	return rateLimitHandler(newIPRateLimiter(rate.Every(100*time.Millisecond), 20, 10*time.Minute))
}
