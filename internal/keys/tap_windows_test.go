//go:build windows

package keys

import "testing"

// findTargetWindow used to register a fresh Windows callback on every call via
// windows.NewCallback. Go's runtime keeps every registration in a fixed
// 2000-entry table that is never freed, so once a focus target is configured
// (the shipped profiles set one) the process aborted with "too many callback
// functions" after ~2000 lookups — reached in ~33 min by the TUI's
// once-a-second presence poll, sooner with key presses. The callback is now
// registered once at package init, so lookups are free.
//
// Against the old code this aborts the test binary (throw, not panic); it must
// therefore be run on Windows, where the bug is observable.
func TestFindTargetWindowDoesNotExhaustCallbacks(t *testing.T) {
	const lookups = 2500 // past the 2000-slot runtime callback table
	for i := 0; i < lookups; i++ {
		hwnd, _, ok := findTargetWindow("definitely-not-a-running-program-xyzzy")
		if ok || hwnd != 0 {
			t.Fatalf("lookup %d matched by accident: hwnd=%d ok=%v", i, hwnd, ok)
		}
	}
}
