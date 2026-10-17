package ble

import (
	"sync/atomic"
	"time"
)

// activityClock records a session's last inbound frame. Notification callbacks
// run on the BLE stack's own goroutine while the keepalive loop reads it on the
// Session goroutine, so the timestamp is atomic: a plain time.Time here is a
// data race (and can tear). nanoseconds since the Unix epoch.
type activityClock struct {
	ns atomic.Int64
}

func newActivityClock() *activityClock {
	a := &activityClock{}
	a.touch()
	return a
}

// touch records "just now".
func (a *activityClock) touch() { a.ns.Store(time.Now().UnixNano()) }

// last returns the recorded timestamp (Unix epoch before the first touch).
func (a *activityClock) last() time.Time { return time.Unix(0, a.ns.Load()) }
