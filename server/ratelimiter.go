package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements a per-client IP token bucket rate limiter.
// It is designed to be used as an HTTP middleware that rejects requests
// with 429 Too Many Requests when the client exceeds its quota.
//
// The implementation keeps a map of rate.Limiter objects indexed by the
// client's remote address. Clients are identified by their IP extracted
// from X-Forwarded-For or the request's RemoteAddr field.
// This approach is simple, efficient and suitable for many public APIs.
//
// Example usage:
//
//	rl := server.NewRateLimiter(100, time.Minute) // 100 requests per minute
//	mux.Handle("/emoji", rl.Middleware(apiHandler))
//
// Author: Myroslav Mokhammad Abdeljawwad

type RateLimiter struct {
	limit     *rate.Limiter
	rateLimit int           // maximum events per period
	period    time.Duration // rate limiting window
	clients   map[string]*rate.Limiter
	mu        sync.Mutex
}

// NewRateLimiter creates a new RateLimiter.
// maxRequests specifies the number of requests allowed in each period.
func NewRateLimiter(maxRequests int, period time.Duration) *RateLimiter {
	return &RateLimiter{
		rateLimit: maxRequests,
		period:    period,
		clients:   make(map[string]*rate.Limiter),
	}
}

// getClientLimiter returns a rate limiter for the given key,
// creating one if it does not already exist.
func (rl *RateLimiter) getClientLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if l, exists := rl.clients[key]; exists {
		return l
	}
	l := rate.NewLimiter(rate.Every(rl.period/time.Duration(rl.rateLimit)), rl.rateLimit)
	rl.clients[key] = l
	return l
}

// Middleware returns an http.Handler that enforces the rate limit.
// It extracts the client IP from X-Forwarded-For or RemoteAddr,
// then checks the limiter before delegating to next.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		clientIP := rl.extractClientIP(r)
		if clientIP == "" {
			http.Error(w, "Unable to determine client IP", http.StatusInternalServerError)
			return
		}

		limiter := rl.getClientLimiter(clientIP)

		ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
		defer cancel()

		if err := limiter.Wait(ctx); err != nil {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// extractClientIP attempts to parse the client's IP address from the request.
// It first checks X-Forwarded-For (common in reverse proxies) and falls back
// to RemoteAddr. The function returns only the IP part without port.
func (rl *RateLimiter) extractClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	return host
}

// GetLimiter returns the underlying rate limiter for a given client key.
// Useful for testing or monitoring purposes.
func (rl *RateLimiter) GetLimiter(key string) (*rate.Limiter, bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	l, ok := rl.clients[key]
	return l, ok
}