package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ipLimiter melacak rate limiter per-client berdasarkan IP.
type ipLimiter struct {
	mu     sync.Mutex
	visits map[string]*rate.Limiter
	r      rate.Limit
	b      int
}

func newIPLimiter(r float64, b int) *ipLimiter {
	return &ipLimiter{
		visits: make(map[string]*rate.Limiter),
		r:      rate.Limit(r),
		b:      b,
	}
}

func (l *ipLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, ok := l.visits[ip]
	if !ok {
		limiter = rate.NewLimiter(l.r, l.b)
		l.visits[ip] = limiter
	}
	return limiter
}

// RateLimitAuth membatasi jumlah request per IP untuk endpoint auth
// (brute force protection). Default: 10 request/detik, burst 5.
func RateLimitAuth() gin.HandlerFunc {
	limiter := newIPLimiter(10, 5)
	return func(c *gin.Context) {
		if !limiter.get(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "terlalu banyak percobaan. Silakan coba lagi nanti.",
			})
			return
		}
		c.Next()
	}
}

// RateLimitGeneral membatasi seluruh request API per IP
// (pencegahan DoS). Default: 30 request/detik, burst 20.
func RateLimitGeneral() gin.HandlerFunc {
	limiter := newIPLimiter(30, 20)
	return func(c *gin.Context) {
		if !limiter.get(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "terlalu banyak permintaan. Silakan coba lagi nanti.",
			})
			return
		}
		c.Next()
	}
}
