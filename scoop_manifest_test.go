package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

func requireJust(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("just"); err != nil {
		t.Skip("just not available")
	}
}

func TestScoopManifestRenders(t *testing.T) {
	requireJust(t)
	const ver = "1.2.3"
	const hash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	out, err := exec.Command("just", "scoop-manifest", ver, hash).Output()
	if err != nil {
		t.Fatalf("script failed: %v", err)
	}
	var m struct {
		Version      string     `json:"version"`
		Description  string     `json:"description"`
		Homepage     string     `json:"homepage"`
		License      string     `json:"license"`
		Bin          string     `json:"bin"`
		Shortcuts    [][]string `json:"shortcuts"`
		Architecture map[string]struct {
			URL  string `json:"url"`
			Hash string `json:"hash"`
		} `json:"architecture"`
		Checkver   string `json:"checkver"`
		Autoupdate struct {
			Architecture map[string]struct {
				URL string `json:"url"`
			} `json:"architecture"`
			Hash struct {
				URL string `json:"url"`
			} `json:"hash"`
		} `json:"autoupdate"`
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("manifest is not valid JSON: %v\n%s", err, out)
	}
	if m.Version != ver {
		t.Errorf("version = %q, want %q", m.Version, ver)
	}
	if m.Description == "" {
		t.Error("description is empty")
	}
	if m.Homepage != "https://github.com/thomaschampagne/zwiftboard" {
		t.Errorf("homepage = %q", m.Homepage)
	}
	if m.License != "MIT" {
		t.Errorf("license = %q, want MIT", m.License)
	}
	if m.Bin != "zwiftboard.exe" {
		t.Errorf("bin = %q, want zwiftboard.exe", m.Bin)
	}
	if len(m.Shortcuts) != 1 || len(m.Shortcuts[0]) != 2 ||
		m.Shortcuts[0][0] != "zwiftboard.exe" || m.Shortcuts[0][1] != "zwiftboard" {
		t.Errorf("shortcuts = %v, want [[zwiftboard.exe zwiftboard]]", m.Shortcuts)
	}
	if m.Checkver != "github" {
		t.Errorf("checkver = %q, want github", m.Checkver)
	}
	arch, ok := m.Architecture["64bit"]
	if !ok {
		t.Fatalf("missing 64bit architecture")
	}
	wantURL := "https://github.com/thomaschampagne/zwiftboard/releases/download/1.2.3/zwiftboard.exe"
	if arch.URL != wantURL {
		t.Errorf("download url = %q, want %q", arch.URL, wantURL)
	}
	if arch.Hash != hash {
		t.Errorf("hash = %q, want %q", arch.Hash, hash)
	}
	if got := m.Autoupdate.Architecture["64bit"].URL; got != "https://github.com/thomaschampagne/zwiftboard/releases/download/$version/zwiftboard.exe" {
		t.Errorf("autoupdate url = %q", got)
	}
	if got := m.Autoupdate.Hash.URL; got != "$baseurl/checksums.txt" {
		t.Errorf("autoupdate hash url = %q", got)
	}
}

func TestScoopManifestValuesAreLiteral(t *testing.T) {
	requireJust(t)
	const ver = "1.2.3-$(echo pwned)"
	const hash = "$(echo pwned)"
	out, err := exec.Command("just", "scoop-manifest", ver, hash).Output()
	if err != nil {
		t.Fatalf("just failed: %v", err)
	}
	var m struct {
		Version      string `json:"version"`
		Architecture map[string]struct {
			Hash string `json:"hash"`
		} `json:"architecture"`
	}
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("manifest is not valid JSON: %v\n%s", err, out)
	}
	if m.Version != ver {
		t.Errorf("version = %q, want the literal %q", m.Version, ver)
	}
	if got := m.Architecture["64bit"].Hash; got != hash {
		t.Errorf("hash = %q, want the literal %q", got, hash)
	}
}

func TestBucketManifestInSync(t *testing.T) {
	requireJust(t)
	raw, err := os.ReadFile("bucket/zwiftboard.json")
	if err != nil {
		t.Fatalf("read bucket/zwiftboard.json: %v", err)
	}
	var m struct {
		Version      string `json:"version"`
		Architecture map[string]struct {
			Hash string `json:"hash"`
		} `json:"architecture"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("bucket manifest is not valid JSON: %v", err)
	}
	hash := m.Architecture["64bit"].Hash
	out, err := exec.Command("just", "scoop-manifest", m.Version, hash).Output()
	if err != nil {
		t.Fatalf("script failed: %v", err)
	}
	if !bytes.Equal(bytes.TrimSpace(out), bytes.TrimSpace(raw)) {
		t.Errorf("bucket/zwiftboard.json is stale; run: just scoop-manifest %s %s > bucket/zwiftboard.json", m.Version, hash)
	}
}

func TestScoopManifestRequiresArgs(t *testing.T) {
	requireJust(t)
	if err := exec.Command("just", "scoop-manifest").Run(); err == nil {
		t.Fatal("missing arguments must fail")
	}
	if err := exec.Command("just", "scoop-manifest", "1.2.3").Run(); err == nil {
		t.Fatal("missing hash argument must fail")
	}
}
