package nasa

import (
	"context"
	"net/http"
	"runtime"
	"testing"
	"time"
)

func TestKeyRotatorWrapsAround(t *testing.T) {
	kr := newKeyRotator([]string{"a", "b", "c"})
	// Reset to deterministic order for testing: override the shuffled keys.
	kr.keys = []string{"a", "b", "c"}
	kr.current = 0

	seen := make(map[string]int)
	for range 6 {
		key, ok := kr.next()
		if !ok {
			t.Fatal("expected a key, got none")
		}
		seen[key]++
	}

	// Each key should appear exactly twice after 6 calls with 3 keys.
	for _, k := range []string{"a", "b", "c"} {
		if seen[k] != 2 {
			t.Errorf("key %q appeared %d times, want 2", k, seen[k])
		}
	}
}

func TestKeyRotatorSkipsLimited(t *testing.T) {
	kr := newKeyRotator([]string{"a", "b", "c"})
	kr.keys = []string{"a", "b", "c"}
	kr.current = 0

	// Mark "a" as limited for a long time.
	kr.markLimited("a", 1*time.Hour)

	// next() should skip "a" and return "b" or "c".
	for range 4 {
		key, ok := kr.next()
		if !ok {
			t.Fatal("expected a key, got none")
		}
		if key == "a" {
			t.Error("got limited key 'a', expected it to be skipped")
		}
	}
}

func TestKeyRotatorAllLimited(t *testing.T) {
	kr := newKeyRotator([]string{"a", "b", "c"})
	kr.keys = []string{"a", "b", "c"}
	kr.current = 0

	kr.markLimited("a", 1*time.Hour)
	kr.markLimited("b", 1*time.Hour)
	kr.markLimited("c", 1*time.Hour)

	key, ok := kr.next()
	if ok {
		t.Errorf("expected no available key, got %q", key)
	}
	if key != "" {
		t.Errorf("expected empty string, got %q", key)
	}
}

func TestKeyRotatorCooldownExpiry(t *testing.T) {
	kr := newKeyRotator([]string{"a", "b"})
	kr.keys = []string{"a", "b"}
	kr.current = 0

	// Mark "a" with a very short cooldown.
	kr.markLimited("a", 50*time.Millisecond)

	// Immediately, "a" should be skipped.
	key, ok := kr.next()
	if !ok {
		t.Fatal("expected a key")
	}
	if key != "b" {
		t.Errorf("expected 'b', got %q", key)
	}

	// Poll until cooldown expires instead of sleeping.
	deadline := time.Now().Add(500 * time.Millisecond)
	var recovered bool
	for time.Now().Before(deadline) {
		kr.mu.Lock()
		kr.current = 0
		kr.mu.Unlock()

		key, ok = kr.next()
		if ok && key == "a" {
			recovered = true
			break
		}
		runtime.Gosched()
	}
	if !recovered {
		t.Fatal("expected key 'a' to recover after cooldown expiry")
	}
}

func TestKeyRotatorEmptyKeys(t *testing.T) {
	kr := newKeyRotator(nil)
	key, ok := kr.next()
	if ok {
		t.Errorf("expected no key from empty rotator, got %q", key)
	}

	kr2 := newKeyRotator([]string{})
	key, ok = kr2.next()
	if ok {
		t.Errorf("expected no key from empty rotator, got %q", key)
	}
}

func TestShortestCooldown(t *testing.T) {
	kr := newKeyRotator([]string{"a", "b", "c"})

	// No limited keys.
	if d := kr.shortestCooldown(); d != 0 {
		t.Errorf("expected 0 cooldown, got %v", d)
	}

	kr.markLimited("a", 10*time.Minute)
	kr.markLimited("b", 1*time.Minute)
	kr.markLimited("c", 5*time.Minute)

	d := kr.shortestCooldown()
	// Shortest should be close to 1 minute (allowing some execution time).
	if d < 50*time.Second || d > 70*time.Second {
		t.Errorf("expected shortest cooldown near 1 minute, got %v", d)
	}
}

func TestAdjustFromHeadersLowRemaining(t *testing.T) {
	rl := newRateLimiter(10.0, nil)

	header := http.Header{}
	header.Set("X-RateLimit-Remaining", "5")
	rl.adjustFromHeaders(header)

	// Rate should be reduced: 5/60.0 ~= 0.0833
	limit := rl.limiter.Limit()
	if limit < 0.08 || limit > 0.09 {
		t.Errorf("expected rate ~0.083, got %f", float64(limit))
	}
}

func TestAdjustFromHeadersZeroRemaining(t *testing.T) {
	rl := newRateLimiter(10.0, nil)

	header := http.Header{}
	header.Set("X-RateLimit-Remaining", "0")
	rl.adjustFromHeaders(header)

	// Rate should be floored at 0.01, not 0.0.
	limit := rl.limiter.Limit()
	if float64(limit) != 0.01 {
		t.Errorf("expected rate 0.01 (floor), got %f", float64(limit))
	}
}

func TestAdjustFromHeadersHealthyRemaining(t *testing.T) {
	rl := newRateLimiter(10.0, nil)

	// First reduce the rate.
	header := http.Header{}
	header.Set("X-RateLimit-Remaining", "5")
	rl.adjustFromHeaders(header)

	// Now restore with healthy remaining.
	header.Set("X-RateLimit-Remaining", "500")
	rl.adjustFromHeaders(header)

	limit := rl.limiter.Limit()
	if float64(limit) != 10.0 {
		t.Errorf("expected default rate 10.0, got %f", float64(limit))
	}
}

func TestAdjustFromHeadersMissing(t *testing.T) {
	rl := newRateLimiter(10.0, nil)
	originalLimit := rl.limiter.Limit()

	// Empty header - should not panic and rate should be unchanged.
	rl.adjustFromHeaders(http.Header{})

	if rl.limiter.Limit() != originalLimit {
		t.Errorf("rate changed on missing header: got %f, want %f",
			float64(rl.limiter.Limit()), float64(originalLimit))
	}
}

func TestAdjustFromHeadersInvalid(t *testing.T) {
	rl := newRateLimiter(10.0, nil)
	originalLimit := rl.limiter.Limit()

	header := http.Header{}
	header.Set("X-RateLimit-Remaining", "not-a-number")
	rl.adjustFromHeaders(header)

	if rl.limiter.Limit() != originalLimit {
		t.Errorf("rate changed on invalid header: got %f, want %f",
			float64(rl.limiter.Limit()), float64(originalLimit))
	}
}

func TestWaitCancelledContext(t *testing.T) {
	rl := newRateLimiter(10.0, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := rl.wait(ctx)
	if err == nil {
		t.Error("expected error from cancelled context, got nil")
	}
}

func TestWaitValidContext(t *testing.T) {
	// Generous rate limit so wait returns immediately.
	rl := newRateLimiter(1000.0, nil)

	ctx := context.Background()
	err := rl.wait(ctx)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestCooldownConstants(t *testing.T) {
	// Verify the cooldown tier constants are defined with expected values.
	if cooldownRateLimit != 30*time.Minute {
		t.Errorf("cooldownRateLimit = %v, want 30m", cooldownRateLimit)
	}
	if cooldownUnavailable != 60*time.Second {
		t.Errorf("cooldownUnavailable = %v, want 60s", cooldownUnavailable)
	}
	if cooldownNetwork != 10*time.Minute {
		t.Errorf("cooldownNetwork = %v, want 10m", cooldownNetwork)
	}
	if cooldownServerError != 5*time.Minute {
		t.Errorf("cooldownServerError = %v, want 5m", cooldownServerError)
	}
}

func TestNewKeyRotatorDoesNotMutateInput(t *testing.T) {
	original := []string{"x", "y", "z"}
	backup := make([]string, len(original))
	copy(backup, original)

	_ = newKeyRotator(original)

	for i, v := range original {
		if v != backup[i] {
			t.Errorf("input slice was mutated at index %d: got %q, want %q", i, v, backup[i])
		}
	}
}

func TestNewRateLimiterMinBurst(t *testing.T) {
	// With rps=0.5, int(0.5)=0, so burst should be 0+1=1.
	rl := newRateLimiter(0.5, nil)
	if rl.burst != 1 {
		t.Errorf("expected burst 1, got %d", rl.burst)
	}

	// With rps=0, burst should still be 1 (minimum).
	rl2 := newRateLimiter(0, nil)
	if rl2.burst != 1 {
		t.Errorf("expected burst 1, got %d", rl2.burst)
	}
}
