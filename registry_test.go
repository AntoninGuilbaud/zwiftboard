package main

import (
	"testing"
	"time"
)

func TestSessionRegistryClaimsOnce(t *testing.T) {
	r := newSessionRegistry()
	if first, n := r.register("AA"); !first || n != 1 {
		t.Fatalf("first register: got first=%v n=%d, want true/1", first, n)
	}
	if first, n := r.register("AA"); first || n != 1 {
		t.Fatalf("duplicate register: got first=%v n=%d, want false/1", first, n)
	}
}

func TestSessionRegistrySeen(t *testing.T) {
	r := newSessionRegistry()
	if r.seen("AA") {
		t.Fatal("an unregistered address must not be reported as seen")
	}
	r.register("AA")
	if !r.seen("AA") {
		t.Fatal("a registered address must be reported as seen")
	}
}

func TestSessionRegistryForgetAllowsReRegister(t *testing.T) {
	r := newSessionRegistry()
	r.register("AA")
	r.forget("AA")
	if r.seen("AA") {
		t.Fatal("a forgotten address must not be reported as seen")
	}
	if first, _ := r.register("AA"); !first {
		t.Fatal("re-registering after forget must succeed")
	}
}

func TestSessionRegistryForgetUnknownIsNoop(t *testing.T) {
	r := newSessionRegistry()
	r.forget("nobody") // must not panic
	if len(r.m) != 0 {
		t.Fatalf("forget added an entry: %v", r.m)
	}
}

func TestIdleFor(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	if idleFor(now.Add(-time.Minute), now) {
		t.Fatal("a controller seen a minute ago must not be reaped")
	}
	if !idleFor(now.Add(-registryIdleTTL), now) {
		t.Fatal("a controller unseen for the full TTL must be reaped")
	}
	if !idleFor(now.Add(-2*registryIdleTTL), now) {
		t.Fatal("a controller unseen well past the TTL must be reaped")
	}
}
