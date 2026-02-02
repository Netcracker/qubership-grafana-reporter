package main

import (
	"context"
	"testing"
	"time"
)

func TestShutdown_Success(t *testing.T) {
	releaseFunc := func(ctx context.Context) {
		// Simulate quick release
		time.Sleep(10 * time.Millisecond)
	}

	ctx := context.Background()
	timeout := 100 * time.Millisecond

	err := Shutdown(ctx, timeout, releaseFunc)
	if err != nil {
		t.Errorf("Shutdown should succeed, but got error: %v", err)
	}
}

func TestShutdown_Timeout(t *testing.T) {
	releaseFunc := func(ctx context.Context) {
		// Simulate slow release that exceeds timeout
		time.Sleep(200 * time.Millisecond)
	}

	ctx := context.Background()
	timeout := 50 * time.Millisecond

	err := Shutdown(ctx, timeout, releaseFunc)
	if err == nil {
		t.Error("Shutdown should timeout, but succeeded")
	}
	if err.Error() != "failed to release resources in time" {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestShutdown_MultipleFinalizers(t *testing.T) {
	finalizer1 := func(ctx context.Context) {
		time.Sleep(10 * time.Millisecond)
	}
	finalizer2 := func(ctx context.Context) {
		time.Sleep(20 * time.Millisecond)
	}

	ctx := context.Background()
	timeout := 100 * time.Millisecond

	err := Shutdown(ctx, timeout, finalizer1, finalizer2)
	if err != nil {
		t.Errorf("Shutdown with multiple finalizers should succeed, but got error: %v", err)
	}
}
