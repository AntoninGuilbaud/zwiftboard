// Package update checks GitHub for a newer zwiftboard release. It is silent by
// design: every failure (offline, rate limit, bad stamp) is the caller's
// "no update" path.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// repo identifies the GitHub project releases are published from.
const repo = "thomaschampagne/zwiftboard"

// latestURL is the unauthenticated latest-release endpoint. The TUI calls this
// once per startup, far under GitHub's 60/h rate limit.
const latestURL = "https://api.github.com/repos/" + repo + "/releases/latest"

// Newer reports whether latest is a strictly newer SemVer than current. Any
// value that fails to parse (the "dev" stamp, empty string, a malformed tag)
// makes the answer false: no chip rather than a wrong one. A pre-release
// latest is also false: strict SemVer ranks 1.0.3-rc.1 above 1.0.2, but a
// pre-release is not something to nudge users toward.
func Newer(current, latest string) bool {
	c := normalize(current)
	l := normalize(latest)
	if !semver.IsValid(c) || !semver.IsValid(l) {
		return false
	}
	if semver.Prerelease(l) != "" {
		return false
	}
	return semver.Compare(l, c) > 0
}

// normalize prepends the "v" x/mod/semver requires; release tags are bare
// ("1.0.2") per semantic-release's tagFormat.
func normalize(v string) string {
	v = strings.TrimSpace(v)
	if v != "" && !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// Latest returns the newest release tag (bare, "v" stripped) from GitHub.
func Latest(ctx context.Context) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	return latestFrom(ctx, client, latestURL)
}

// latestFrom fetches and decodes one GitHub latest-release document. Split out
// so tests can point it at an httptest server.
func latestFrom(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	// GitHub rejects UA-less clients.
	req.Header.Set("User-Agent", "zwiftboard")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("update check: %s", resp.Status)
	}
	var doc struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", err
	}
	if doc.TagName == "" {
		return "", fmt.Errorf("update check: empty tag_name")
	}
	return strings.TrimPrefix(doc.TagName, "v"), nil
}
