package provider

import (
	"context"
	"testing"
	"time"

	"github.com/wasilak/nim/pkg/resource"
)

// blockingProvider simulates a provider that blocks until context cancellation.
type blockingProvider struct{}

func (b *blockingProvider) Name() string              { return "blocking" }
func (b *blockingProvider) Available() (bool, string) { return true, "ok" }
func (b *blockingProvider) Reconcile(ctx context.Context, desired []resource.ResourceGroup[any], state []ResourceState) GroupPlan {
	// Block until canceled, then return an empty plan.
	select {
	case <-ctx.Done():
		return GroupPlan{}
	case <-time.After(1 * time.Second):
		// If not canceled within 1s, return empty plan to avoid long test hangs
		return GroupPlan{}
	}
}

func (b *blockingProvider) Apply(ctx context.Context, plan GroupPlan) error {
	// Block until canceled and then return ctx.Err()
	<-ctx.Done()
	return ctx.Err()
}

func (b *blockingProvider) Import(ctx context.Context, group string) (ResourceState, error) {
	return ResourceState{}, nil
}

func (b *blockingProvider) ImportItem(ctx context.Context, group string, item string) (ResourceState, error) {
	return ResourceState{}, nil
}

func TestBlockingProvider_ApplyCancellation(t *testing.T) {
	p := &blockingProvider{}
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel shortly after starting
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := p.Apply(ctx, GroupPlan{})
	dur := time.Since(start)

	if err == nil {
		t.Fatalf("expected error from canceled context, got nil")
	}
	if err != context.Canceled && err != context.DeadlineExceeded {
		t.Fatalf("expected context.Canceled or DeadlineExceeded, got: %v", err)
	}
	if dur > 500*time.Millisecond {
		t.Fatalf("Apply did not return promptly after cancel; duration=%s", dur)
	}
}

func TestBlockingProvider_ReconcileReturnsOnCancel(t *testing.T) {
	p := &blockingProvider{}
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel shortly after starting
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	plan := p.Reconcile(ctx, []resource.ResourceGroup[any]{}, []ResourceState{})
	dur := time.Since(start)

	_ = plan // just ensure we returned
	if dur > 500*time.Millisecond {
		t.Fatalf("Reconcile did not return promptly after cancel; duration=%s", dur)
	}
}
