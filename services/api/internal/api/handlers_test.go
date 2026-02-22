package api

import (
	"testing"
	"time"
)

func TestComputeRequestSignatureDeterministic(t *testing.T) {
	a := computeRequestSignature("POST", "/v1/samples/upload-url", "1700000000", `{"catId":"c1"}`, "token")
	b := computeRequestSignature("POST", "/v1/samples/upload-url", "1700000000", `{"catId":"c1"}`, "token")
	if a == "" {
		t.Fatalf("expected signature")
	}
	if a != b {
		t.Fatalf("signature should be deterministic")
	}
}

func TestRequestLimiterAllow(t *testing.T) {
	limiter := newRequestLimiter()
	now := time.Unix(1700000000, 0)
	if !limiter.Allow("u1", 2, now) {
		t.Fatalf("first request should pass")
	}
	if !limiter.Allow("u1", 2, now) {
		t.Fatalf("second request should pass")
	}
	if limiter.Allow("u1", 2, now) {
		t.Fatalf("third request should be limited")
	}
	if !limiter.Allow("u1", 2, now.Add(time.Minute)) {
		t.Fatalf("new window should pass")
	}
}

func TestDailyQuotaLimiter(t *testing.T) {
	limiter := newDailyQuotaLimiter()
	now := time.Unix(1700000000, 0)
	if !limiter.Allow("u1", 2, now) {
		t.Fatalf("first quota should pass")
	}
	if !limiter.Allow("u1", 2, now) {
		t.Fatalf("second quota should pass")
	}
	if limiter.Allow("u1", 2, now) {
		t.Fatalf("third quota should be blocked")
	}
	if !limiter.Allow("u1", 2, now.Add(24*time.Hour)) {
		t.Fatalf("next day should reset quota")
	}
}
