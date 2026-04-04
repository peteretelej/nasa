// Token-bucket rate limiting with multi-key rotation and adaptive throttling.

package nasa

import (
	"context"
	"math/rand/v2"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Cooldown tiers for different error types.
const (
	cooldownRateLimit   = 30 * time.Minute // HTTP 429
	cooldownUnavailable = 60 * time.Second // HTTP 503
	cooldownNetwork     = 10 * time.Minute // dial timeout, DNS failure, connection refused
	cooldownServerError = 5 * time.Minute  // other 5xx
)

// keyRotator manages multiple API keys with cooldown tracking.
type keyRotator struct {
	mu      sync.Mutex
	keys    []string
	current int
	limited map[string]time.Time // key -> cooldown expiry time
}

// newKeyRotator creates a key rotator with a shuffled copy of the provided keys.
func newKeyRotator(keys []string) *keyRotator {
	copied := make([]string, len(keys))
	copy(copied, keys)
	rand.Shuffle(len(copied), func(i, j int) {
		copied[i], copied[j] = copied[j], copied[i]
	})
	return &keyRotator{
		keys:    copied,
		limited: make(map[string]time.Time),
	}
}

// next returns the next available (non-limited) key. It wraps around the key
// list starting from the current index, skipping keys whose cooldown has not
// expired. Expired cooldowns are cleaned up during iteration. Returns ("", false)
// if all keys are currently limited.
func (kr *keyRotator) next() (string, bool) {
	kr.mu.Lock()
	defer kr.mu.Unlock()

	n := len(kr.keys)
	if n == 0 {
		return "", false
	}

	now := time.Now()
	for i := range n {
		idx := (kr.current + i) % n
		key := kr.keys[idx]

		if expiry, limited := kr.limited[key]; limited {
			if now.Before(expiry) {
				continue // still in cooldown
			}
			// Cooldown expired, key has recovered.
			delete(kr.limited, key)
		}

		kr.current = (idx + 1) % n
		return key, true
	}

	return "", false
}

// markLimited puts a key into cooldown for the specified duration.
func (kr *keyRotator) markLimited(key string, cooldown time.Duration) {
	kr.mu.Lock()
	defer kr.mu.Unlock()
	kr.limited[key] = time.Now().Add(cooldown)
}

// shortestCooldown returns the shortest remaining cooldown across all limited
// keys, or 0 if no keys are limited.
func (kr *keyRotator) shortestCooldown() time.Duration {
	kr.mu.Lock()
	defer kr.mu.Unlock()

	if len(kr.limited) == 0 {
		return 0
	}

	now := time.Now()
	shortest := time.Duration(0)
	first := true
	for _, expiry := range kr.limited {
		remaining := expiry.Sub(now)
		if remaining <= 0 {
			continue
		}
		if first || remaining < shortest {
			shortest = remaining
			first = false
		}
	}
	return shortest
}

// rateLimiter combines token-bucket rate limiting with multi-key rotation.
type rateLimiter struct {
	limiter *rate.Limiter
	keys    *keyRotator
}

// newRateLimiter creates a rate limiter with the given requests-per-second and
// API keys. The burst size is rps+1 (minimum 1).
func newRateLimiter(rps float64, keys []string) *rateLimiter {
	burst := int(rps) + 1
	if burst < 1 {
		burst = 1
	}
	return &rateLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
		keys:    newKeyRotator(keys),
	}
}

// wait blocks until the rate limiter allows a request or the context is done.
func (rl *rateLimiter) wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}
