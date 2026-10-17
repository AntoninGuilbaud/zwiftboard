package main

import (
	"sync"
	"time"
)

// registryIdleTTL is how long a controller may go unseen before its session
// slot is released. It is far longer than the right pod's ~65s sleep and the
// reconnect window (ble.ReconnectWindow, 30s): a controller still advertising
// during a normal idle is never reaped, only one that is genuinely gone (off,
// out of range, or paired elsewhere). A later sighting re-registers it.
const registryIdleTTL = 10 * time.Minute

// sessionRegistry tracks which controller addresses already own a session
// goroutine, so the scanner never spawns a duplicate reconnect loop for an
// address. Entries are released when a goroutine gives up on a controller that
// has been gone for registryIdleTTL, so the map (and the per-controller 1Hz
// wait loops) do not accumulate for controllers that never come back.
//
// Reaping is only ever done by the goroutine that owns the entry, as it exits:
//
//	register → ... run the reconnect loop ... → forget → return
//
// A fresh sighting after that can register the address again and start a new
// loop. Because the owner deletes only when it is already leaving, there is no
// window in which two goroutines own the same address.
type sessionRegistry struct {
	mu sync.Mutex
	m  map[string]bool
}

func newSessionRegistry() *sessionRegistry {
	return &sessionRegistry{m: map[string]bool{}}
}

// register claims addr, returning isFirst=true and the new count only for the
// first caller; a duplicate sighting of an owned address is refused so exactly
// one goroutine runs per address.
func (r *sessionRegistry) register(addr string) (isFirst bool, count int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.m[addr] {
		return false, len(r.m)
	}
	r.m[addr] = true
	return true, len(r.m)
}

// seen reports whether addr already owns a session goroutine. The scanner's
// dedup callback (Watch's registered) uses this.
func (r *sessionRegistry) seen(addr string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.m[addr]
}

// forget releases addr so a later sighting can register it again. Only the
// owning goroutine calls it, as it exits after idleFor says to give up.
func (r *sessionRegistry) forget(addr string) {
	r.mu.Lock()
	delete(r.m, addr)
	r.mu.Unlock()
}

// idleFor reports whether a controller last seen at lastSeen has been gone for
// the full TTL and its session slot should be released. Extracted so the
// give-up policy is testable without hardware.
func idleFor(lastSeen, now time.Time) bool {
	return now.Sub(lastSeen) >= registryIdleTTL
}
