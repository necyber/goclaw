package lane

import (
	"context"
	"sync"
	"time"
)

// TokenBucket implements the token bucket algorithm for rate limiting.
type TokenBucket struct {
	rate       float64   // tokens per second
	capacity   float64   // bucket capacity
	tokens     float64   // current tokens
	lastUpdate time.Time // last time tokens were updated
	
	mu sync.Mutex
}

// NewTokenBucket creates a new TokenBucket rate limiter.
// rate is the number of tokens added per second.
// capacity is the maximum number of tokens the bucket can hold.
func NewTokenBucket(rate, capacity float64) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity, // Start with a full bucket
		lastUpdate: time.Now(),
	}
}

// Allow checks if a token is available without blocking.
// Returns true if a token was consumed, false otherwise.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	tb.addTokens(time.Now())
	
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// Wait blocks until a token is available or the context is cancelled.
// Returns nil if a token was acquired, otherwise returns the context error.
func (tb *TokenBucket) Wait(ctx context.Context) error {
	// Fast path: try to get a token immediately
	if tb.Allow() {
		return nil
	}
	
	// Slow path: wait for a token
	ticker := time.NewTicker(time.Millisecond * 10)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if tb.Allow() {
				return nil
			}
		}
	}
}

// WaitTimeout tries to acquire a token with a timeout.
// Returns true if a token was acquired, false if timeout occurred.
func (tb *TokenBucket) WaitTimeout(timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	return tb.Wait(ctx) == nil
}

// addTokens adds tokens to the bucket based on elapsed time.
func (tb *TokenBucket) addTokens(now time.Time) {
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.tokens += elapsed * tb.rate
	
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	
	tb.lastUpdate = now
}

// Tokens returns the current number of tokens in the bucket.
func (tb *TokenBucket) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	tb.addTokens(time.Now())
	return tb.tokens
}

// Rate returns the rate of token generation (tokens per second).
func (tb *TokenBucket) Rate() float64 {
	return tb.rate
}

// Capacity returns the bucket capacity.
func (tb *TokenBucket) Capacity() float64 {
	return tb.capacity
}

// SetRate updates the rate of token generation.
func (tb *TokenBucket) SetRate(rate float64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	tb.rate = rate
}

// SetCapacity updates the bucket capacity.
func (tb *TokenBucket) SetCapacity(capacity float64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	tb.capacity = capacity
	if tb.tokens > capacity {
		tb.tokens = capacity
	}
}

// LeakyBucket implements the leaky bucket algorithm for rate limiting.
// Unlike token bucket which allows bursts, leaky bucket provides a constant outflow rate.
type LeakyBucket struct {
	rate       float64       // leak rate (tokens per second)
	capacity   int           // bucket capacity
	queue      chan struct{} // queued requests
	leakTicker *time.Ticker
	stopCh     chan struct{}
	stopOnce   sync.Once
}

// NewLeakyBucket creates a new LeakyBucket rate limiter.
func NewLeakyBucket(rate float64, capacity int) *LeakyBucket {
	lb := &LeakyBucket{
		rate:     rate,
		capacity: capacity,
		queue:    make(chan struct{}, capacity),
		stopCh:   make(chan struct{}),
	}
	
	// Start the leak goroutine
	interval := time.Second / time.Duration(rate)
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	lb.leakTicker = time.NewTicker(interval)
	
	go lb.leak()
	
	return lb
}

// leak continuously drains the bucket at the specified rate.
func (lb *LeakyBucket) leak() {
	for {
		select {
		case <-lb.stopCh:
			lb.leakTicker.Stop()
			return
		case <-lb.leakTicker.C:
			select {
			case <-lb.queue:
				// Leaked one token
			default:
				// Bucket is empty
			}
		}
	}
}

// Allow tries to add a request to the bucket.
// Returns true if the request was accepted, false if the bucket is full.
func (lb *LeakyBucket) Allow() bool {
	select {
	case lb.queue <- struct{}{}:
		return true
	default:
		return false
	}
}

// Wait blocks until the request can be added to the bucket.
func (lb *LeakyBucket) Wait(ctx context.Context) error {
	select {
	case lb.queue <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-lb.stopCh:
		return ErrLeakyBucketStopped
	}
}

// Stop stops the leaky bucket.
func (lb *LeakyBucket) Stop() {
	lb.stopOnce.Do(func() {
		close(lb.stopCh)
	})
}

// QueueSize returns the current number of queued requests.
func (lb *LeakyBucket) QueueSize() int {
	return len(lb.queue)
}

var ErrLeakyBucketStopped = &LeakyBucketStoppedError{}

type LeakyBucketStoppedError struct{}

func (e *LeakyBucketStoppedError) Error() string {
	return "leaky bucket is stopped"
}
