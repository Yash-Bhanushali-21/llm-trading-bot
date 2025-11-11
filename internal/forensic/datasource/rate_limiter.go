package datasource

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	tokens         int
	maxTokens      int
	refillRate     time.Duration
	lastRefillTime time.Time
	mu             sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// maxTokens: maximum number of tokens in the bucket
// refillRate: how often to add a token (e.g., 100ms = 10 requests/second)
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:         maxTokens,
		maxTokens:      maxTokens,
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}
}

// Wait waits until a token is available
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if rl.tryAcquire() {
				return nil
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// tryAcquire attempts to acquire a token
func (rl *RateLimiter) tryAcquire() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(rl.lastRefillTime)
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefillTime = now
	}

	// Try to consume a token
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// MultiRateLimiter manages rate limiters for different sources
type MultiRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
}

// NewMultiRateLimiter creates a new multi-source rate limiter
func NewMultiRateLimiter() *MultiRateLimiter {
	return &MultiRateLimiter{
		limiters: make(map[string]*RateLimiter),
	}
}

// AddLimiter adds a rate limiter for a specific source
func (mrl *MultiRateLimiter) AddLimiter(source string, maxTokens int, refillRate time.Duration) {
	mrl.mu.Lock()
	defer mrl.mu.Unlock()

	mrl.limiters[source] = NewRateLimiter(maxTokens, refillRate)
}

// Wait waits for the specified source's rate limiter
func (mrl *MultiRateLimiter) Wait(ctx context.Context, source string) error {
	mrl.mu.RLock()
	limiter, ok := mrl.limiters[source]
	mrl.mu.RUnlock()

	if !ok {
		// No rate limiter for this source, return immediately
		return nil
	}

	return limiter.Wait(ctx)
}

// GetLimiter returns the rate limiter for a source
func (mrl *MultiRateLimiter) GetLimiter(source string) *RateLimiter {
	mrl.mu.RLock()
	defer mrl.mu.RUnlock()

	return mrl.limiters[source]
}

// WithRateLimit wraps a function with rate limiting
func WithRateLimit(ctx context.Context, limiter *RateLimiter, fn func() error) error {
	if limiter != nil {
		if err := limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return fn()
}
