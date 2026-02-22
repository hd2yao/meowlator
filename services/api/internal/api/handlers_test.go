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
