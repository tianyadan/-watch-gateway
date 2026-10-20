package bus

import (
	"testing"
	"time"
)

func TestRateLimiter_AllowsFourThenBlocks(t *testing.T) {
	limiter := NewRateLimiter(4, time.Minute)
	now := time.Unix(1_700_000_000, 0)
	key := "1.2.3.4|abcd1234"

	for i := 0; i < 4; i++ {
		if !limiter.Allow(key, now.Add(time.Duration(i)*time.Second)) {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if limiter.Allow(key, now.Add(5*time.Second)) {
		t.Fatal("5th request within window should be blocked")
	}
}

func TestRateLimiter_ResetsAfterWindow(t *testing.T) {
	limiter := NewRateLimiter(4, time.Minute)
	now := time.Unix(1_700_000_000, 0)
	key := "1.2.3.4|abcd1234"

	for i := 0; i < 4; i++ {
		if !limiter.Allow(key, now) {
			t.Fatalf("warmup %d failed", i+1)
		}
	}
	if limiter.Allow(key, now.Add(30*time.Second)) {
		t.Fatal("should still be blocked inside window")
	}
	if !limiter.Allow(key, now.Add(61*time.Second)) {
		t.Fatal("should allow after window expired")
	}
}

func TestRateLimiter_DifferentKeysIndependent(t *testing.T) {
	limiter := NewRateLimiter(4, time.Minute)
	now := time.Unix(1_700_000_000, 0)

	for i := 0; i < 4; i++ {
		if !limiter.Allow("ip1|h1", now) {
			t.Fatalf("h1 request %d blocked", i+1)
		}
	}
	if !limiter.Allow("ip1|h2", now) {
		t.Fatal("different hash should not share limit")
	}
	if !limiter.Allow("ip2|h1", now) {
		t.Fatal("different ip should not share limit")
	}
}
