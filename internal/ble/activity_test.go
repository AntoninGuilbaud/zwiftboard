package ble

import (
	"sync"
	"testing"
	"time"
)

// Touch must advance the recorded activity time (the keepalive's idle-reset
// gate compares against it).
func TestActivityClockTouchAdvances(t *testing.T) {
	a := newActivityClock()
	first := a.last()
	time.Sleep(2 * time.Millisecond)
	a.touch()
	if !a.last().After(first) {
		t.Fatal("touch must advance the clock")
	}
}

// The clock is written by WinRT notification callbacks (which run on the BLE
// stack's own goroutine) while the keepalive loop reads it on the Session
// goroutine. A non-atomic time.Time here is a data race and can tear; this
// test reproduces the access pattern and is meaningful under -race.
func TestActivityClockConcurrentTouchAndRead(t *testing.T) {
	a := newActivityClock()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10_000; j++ {
				a.touch()
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 10_000; j++ {
			_ = a.last()
		}
	}()
	wg.Wait()
}
