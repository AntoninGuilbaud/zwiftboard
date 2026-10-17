//go:build !windows

package launch

import "os/exec"

// defaultEnv on non-Windows is inert: the app ships for Windows, and tests
// inject their own Env instead of reaching the real OS.
func defaultEnv() Env {
	return Env{
		Exists:  func(string) bool { return true },
		Running: func(string) bool { return false },
		Start:   func(*exec.Cmd) error { return nil },
	}
}

// resolveExe off Windows only consults PATH: App Paths is a Windows registry
// mechanism. Kept so launch.go compiles on the dev platform.
func resolveExe(name string) (string, bool) {
	p, err := exec.LookPath(name)
	return p, err == nil
}
