package middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/PhantomX7/athleton/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"golang.org/x/time/rate"
)

// maxTrackedIPs caps how many distinct client IPs a single limiter holds at
// once. The plain map this replaced had no ceiling — under IP churn it grew
// with every unique address seen within a TTL window. The LRU gives a hard
// bound: once full, the least-recently-seen IP is evicted. Actively-hammering
// IPs stay "recently used" and keep their bucket, so throttling is unaffected;
// only idle IPs are dropped early, and their bucket is recreated on return.
const maxTrackedIPs = 8192

type ipRateLimiter struct {
	// mu makes the get-or-create below atomic: the expirable cache is itself
	// thread-safe, but a bare Get-then-Add would let two concurrent first hits
	// for the same new IP each mint a limiter, discarding one bucket's state.
	mu       sync.Mutex
	visitors *expirable.LRU[string, *rate.Limiter]
	rate     rate.Limit
	burst    int
}

func newIPRateLimiter(r rate.Limit, burst int, ttl time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		visitors: expirable.NewLRU[string, *rate.Limiter](maxTrackedIPs, nil, ttl),
		rate:     r,
		burst:    burst,
	}
}

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	limiter, ok := l.visitors.Get(ip)
	if !ok {
		limiter = rate.NewLimiter(l.rate, l.burst)
		l.visitors.Add(ip, limiter)
	}
	l.mu.Unlock()
	return limiter.Allow()
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
