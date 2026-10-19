package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.Ascii)
	os.Exit(m.Run())
}

func on(cfg Config, l, r PodState) Model {
	m := New(cfg)
	m.bt, m.left, m.right = true, l, r
	return m
}

func has(t *testing.T, v tea.View, subs ...string) {
	t.Helper()
	for _, s := range subs {
		if !strings.Contains(v.Content, s) {
			t.Errorf("view missing %q:\n%s", s, v.Content)
		}
	}
}

func lacks(t *testing.T, v tea.View, subs ...string) {
	t.Helper()
	for _, s := range subs {
		if strings.Contains(v.Content, s) {
			t.Errorf("view should not contain %q:\n%s", s, v.Content)
		}
	}
}

func TestViewBTOffHaltsSetup(t *testing.T) {
	m := New(testCfg())
	m.left, m.right = PodConnected, PodConnected // stale poll
	v := m.View()
	has(t, v, "Bluetooth", "turn")
	lacks(t, v, "Left Controller", "key bindings")
}

func TestViewInlinePrompts(t *testing.T) {
	v := on(testCfg(), PodConnected, PodOff).View()
	has(t, v, "Right Controller: not detected → please activate the Right Controller")
	lacks(t, v, "please activate the Left Controller", "disconnected")
	v = on(testCfg(), PodOff, PodOff).View()
	has(t, v, "Left Controller: not detected → please activate the Left Controller",
		"Right Controller: not detected → please activate the Right Controller")
	v = on(testCfg(), PodDetected, PodDetected).View()
	lacks(t, v, "please activate")
}

func TestViewVersion(t *testing.T) {
	// "dev" is the on-disk build stamp; on release main stamps it via
	// -X main.version. The dev build drives it through the Config field; the
	// release binary carries the release tag (e.g. "1.2.3").
	// "dev" is the on-disk build stamp; on release main stamps it via
	// -X main.version. The dev build drives it through the Config field; the
	// release binary carries the release tag (e.g. "1.2.3"). An empty version
	// hides the suffix (BT-off path shows the bare title).
	for _, v := range []string{"", "dev", "1.2.3", "1.2.3.0"} {
		m := on(testCfg(), PodConnected, PodConnected)
		m.cfg.Version = v
		t.Run(v, func(t *testing.T) {
			out := m.View()
			lacks(t, out, "zwiftboarddev") // title not glued to version
			if v == "" {
				lacks(t, out, "zwiftboard dev")
			} else {
				has(t, out, v) // e.g. "zwiftboard ... (version dev)" / "... 1.2.3"
			}
		})
	}
}

func TestViewUpdateChip(t *testing.T) {
	m := on(testCfg(), PodConnected, PodConnected)
	m.cfg.Version = "1.0.1"
	m.latest = "1.0.2"
	v := m.View()
	has(t, v, "1.0.1", "→ version 1.0.2 available") // version and the update notice

	// No update result → no notice, no second version.
	m.latest = ""
	v = m.View()
	has(t, v, "1.0.1", "version 1.0.1")
	lacks(t, v, "→ version", "1.0.2", "available") // table rows always contain a bare "→"
}

func TestViewIntroLine(t *testing.T) {
	v := on(testCfg(), PodOff, PodConnected).View()
	has(t, v, "WAITING", "Both controllers must be ON")
	lacks(t, v, "virtual keyboard", "READY")
	v = on(testCfg(), PodConnected, PodConnected).View()
	has(t, v, "READY", "Both controllers connected", "Only the Right controller is supported as a virtual keyboard")
	lacks(t, v, "Both controllers must be ON", "WAITING")
}

func TestViewTable(t *testing.T) {
	v := on(testCfg(), PodConnected, PodConnected).View()
	has(t, v, "test", "PLUS", "+")
}

func TestProfileMsgUpdatesHeader(t *testing.T) {
	n, _ := New(testCfg()).Update(ProfileMsg{Name: "systm", Index: 2, Bindings: map[string]string{"A": "space"}})
	if !strings.Contains(n.(Model).View().Content, "systm") {
		t.Fatal("profile not shown")
	}
}

func TestViewFocus(t *testing.T) {
	cfg := testCfg()
	cfg.Focus = "MyWhoosh"
	has(t, on(cfg, PodConnected, PodConnected).View(), "MyWhoosh")
	has(t, on(testCfg(), PodConnected, PodConnected).View(), "off")
}

func TestViewStartProgram(t *testing.T) {
	// The table only renders once Bluetooth is on (the BT-off screen is the
	// minimal "turn it on" prompt), so drive the BT-on view like TestViewFocus.
	cfg := testCfg()
	cfg.StartProgram = "MyWhooshHD.exe"
	has(t, on(cfg, PodConnected, PodConnected).View(), "MyWhooshHD.exe")
	lacks(t, on(testCfg(), PodConnected, PodConnected).View(), "Start program")
}

func TestViewNoMapping(t *testing.T) {
	cfg := testCfg()
	cfg.NoMapping = true
	has(t, on(cfg, PodConnected, PodConnected).View(), "no key mapping")
}

func TestViewZeroSize(t *testing.T) {
	_ = New(testCfg()).View()
	_ = on(testCfg(), PodOff, PodDetected).View()
}

func TestViewLogPanel(t *testing.T) {
	logs := NewLogBuffer(10)
	logs.Write([]byte("level=INFO msg=hello-from-slog\n"))
	cfg := testCfg()
	cfg.Logs = logs
	m := on(cfg, PodConnected, PodConnected)
	lacks(t, m.View(), "hello-from-slog")
	has(t, m.View(), "l logs")
	m.showLogs = true
	has(t, m.View(), "hello-from-slog", "Logs")
}

func TestViewLogPanelWithoutBuffer(t *testing.T) {
	m := on(testCfg(), PodConnected, PodConnected)
	m.showLogs = true
	has(t, m.View(), "no log output")
}

func TestViewNarrowWidthDoesNotPanic(t *testing.T) {
	for _, w := range []int{0, 1, 10, 30, 200} {
		m := on(testCfg(), PodOff, PodOff)
		m.width = w
		m.showLogs = true
		_ = m.View()
	}
}

func focusCfg() Config {
	cfg := testCfg()
	cfg.Focus = "MyWhoosh"
	return cfg
}

func TestViewFocusMissingAlert(t *testing.T) {
	m := on(focusCfg(), PodConnected, PodConnected)
	lacks(t, m.View(), "not found") // nothing reported yet: no false alarm
	m.focusMissing = true
	has(t, m.View(), "ALERT", `"MyWhoosh" not found`, "clicks are dropped")
	cfg := testCfg() // no focus configured: never alert
	m = on(cfg, PodConnected, PodConnected)
	m.focusMissing = true
	lacks(t, m.View(), "ALERT", "not found")
}

func TestViewLogPanelWiderThanCards(t *testing.T) {
	logs := NewLogBuffer(5)
	line := strings.Repeat("x", 130)
	logs.Write([]byte(line + "\n"))
	cfg := testCfg()
	cfg.Logs = logs
	m := on(cfg, PodConnected, PodConnected)
	m.showLogs = true
	m.width = 170
	has(t, m.View(), line)
	m.width = 80
	v := m.View()
	lacks(t, v, line)
	has(t, v, "…")
}

func TestViewFooterOpenConfigHint(t *testing.T) {
	has(t, on(openCfg(func() error { return nil }), PodConnected, PodConnected).View(), "e edit config")
	lacks(t, on(testCfg(), PodConnected, PodConnected).View(), "edit config")
}

func TestViewFooterProfileSwitchHint(t *testing.T) {
	cfg := testCfg()
	cfg.SwitchProfile = func(int) {}
	cfg.Profiles = []string{"a", "b"}
	has(t, on(cfg, PodConnected, PodConnected).View(), "ctrl+←/→ switch profile")

	// A single profile hides the hint even though switching is wired.
	one := testCfg()
	one.SwitchProfile = func(int) {}
	one.Profiles = []string{"a"}
	lacks(t, on(one, PodConnected, PodConnected).View(), "switch profile")

	lacks(t, on(testCfg(), PodConnected, PodConnected).View(), "switch profile")
}

func TestViewFooterActionHints(t *testing.T) {
	cfg := testCfg()
	cfg.Reload = func() {}
	cfg.LaunchProgram = func() error { return nil }
	cfg.StartProgram = "MyWhooshHD.exe"
	v := on(cfg, PodConnected, PodConnected).View()
	has(t, v, "r reload config", "s start program")

	// The reload hint is hidden without the callback, the start hint without a
	// program on the active profile.
	lacks(t, on(testCfg(), PodConnected, PodConnected).View(), "reload config", "start program")
	cfg.StartProgram = ""
	lacks(t, on(cfg, PodConnected, PodConnected).View(), "start program")
}

// The header carries the four-dot pod mark in front of the title, all on one
// line: the app icon's dots (cyan, orange, green, magenta).
func TestViewHeaderPodLogo(t *testing.T) {
	m := on(testCfg(), PodConnected, PodConnected)
	h := m.header()
	if strings.Contains(h, "\n") {
		t.Fatalf("header should be a single line:\n%q", h)
	}
	if n := strings.Count(h, "●"); n != 4 {
		t.Errorf("header has %d pod dots, want 4:\n%s", n, h)
	}
	if !strings.HasPrefix(h, "● ● ● ● zwiftboard") {
		t.Errorf("header = %q, want four dots before the title", h)
	}
}

func TestViewHeaderProfileIndex(t *testing.T) {
	cfg := testCfg()
	cfg.Profile = "systm"
	cfg.Profiles = []string{"a", "systm", "c"}
	cfg.ProfileIndex = 1
	has(t, on(cfg, PodConnected, PodConnected).View(), "systm 2/3")
}

func TestViewShowsStatus(t *testing.T) {
	m := on(testCfg(), PodConnected, PodConnected)
	m.status = "Opened config.yaml"
	has(t, m.View(), "Opened config.yaml")
	m.bt = false // also visible while Bluetooth is off
	has(t, m.View(), "Opened config.yaml")
}

func TestViewLogPanelDoubleWidth(t *testing.T) {
	logs := NewLogBuffer(5)
	line := strings.Repeat("y", 290)
	logs.Write([]byte(line + "\n"))
	cfg := testCfg()
	cfg.Logs = logs
	m := on(cfg, PodConnected, PodConnected)
	m.showLogs = true
	m.width = 320
	has(t, m.View(), line)
}

func TestViewKeyStates(t *testing.T) {
	m := on(testCfg(), PodConnected, PodConnected)
	m.clock = func() time.Time { return time.Now() }
	next, _ := m.Update(KeyStateMsg{Button: "A", Down: true})
	has(t, next.(Model).View(), "▶ pressed")
	lacks(t, next.(Model).View(), "released")

	// Force the hold phase by backdating the press.
	m2 := next.(Model)
	s := m2.keys["A"]
	s.at = time.Now().Add(-holdAfter - time.Millisecond)
	k := make(map[string]keyState, 1)
	k["A"] = s
	m2.keys = k
	has(t, m2.View(), "hold")

	// Released: normal keycap, muted word.
	m3 := m2
	k2 := make(map[string]keyState, 1)
	k2["A"] = keyState{down: false, at: time.Now()}
	m3.keys = k2
	v := m3.View()
	has(t, v, "released")
	lacks(t, v, "▶ pressed")
}

// padRight must pad to a display-cell width, not a byte count: the status
// labels carry multibyte glyphs (▶, ○, the braille pulse), so byte padding left
// the status column misaligned. "▶ pressed" is 11 bytes but 9 cells.
func TestPadRightUsesDisplayWidth(t *testing.T) {
	for _, s := range []string{"▶ pressed", "○ released", "pressed", "backspace"} {
		if w := lipgloss.Width(padRight(s, 12)); w != 12 {
			t.Errorf("padRight(%q) display width = %d, want 12", s, w)
		}
	}
}
