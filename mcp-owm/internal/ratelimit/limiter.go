package ratelimit

import "golang.org/x/time/rate"

// NewLimiter creates a per-minute rate limiter with burst equal to perMinute.
func NewLimiter(perMinute int) *rate.Limiter {
	r := rate.Limit(float64(perMinute) / 60.0)
	return rate.NewLimiter(r, perMinute)
}
