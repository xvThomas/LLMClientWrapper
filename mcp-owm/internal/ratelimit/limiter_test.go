package ratelimit

import (
	"context"
	"testing"
)

func TestNewLimiter(t *testing.T) {
	perMinute := 60
	lim := NewLimiter(perMinute)
	if lim == nil {
		t.Fatal("expected non-nil limiter")
	}
	// Burst equals perMinute, so the first Wait should succeed immediately.
	if err := lim.Wait(context.Background()); err != nil {
		t.Errorf("expected first Wait to succeed, got: %v", err)
	}
}

func TestNewLimiter_ZeroAllowsNoRequests(t *testing.T) {
	lim := NewLimiter(0)
	if lim == nil {
		t.Fatal("expected non-nil limiter")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := lim.Wait(ctx)
	if err == nil {
		t.Error("expected error for cancelled context with zero-rate limiter")
	}
}
