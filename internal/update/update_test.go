package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"1.0.2", "1.0.3", true},       // patch bump
		{"1.0.2", "1.1.0", true},       // minor bump
		{"1.0.2", "2.0.0", true},       // major bump
		{"1.0.2", "1.0.2", false},      // same
		{"1.0.3", "1.0.2", false},      // older remote
		{"dev", "1.0.3", false},        // non-semver local stamp
		{"1.0.2", "dev", false},        // non-semver remote
		{"", "1.0.3", false},           // empty local
		{"v1.0.2", "1.0.3", true},      // v-prefixed local tolerated
		{"1.0.2", "1.0.3-rc.1", false}, // pre-release is LOWER than the release
		{"1.0.3-rc.1", "1.0.3", true},
	}
	for _, c := range cases {
		if got := Newer(c.current, c.latest); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v1.0.3"})
	}))
	defer srv.Close()

	// Latest() against a real GitHub URL is untestable without an injectable
	// base; test the JSON parsing path via latestFrom.
	got, err := latestFrom(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.0.3" {
		t.Errorf("got %q, want %q", got, "1.0.3")
	}
}

func TestLatestFromErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer srv.Close()
	if _, err := latestFrom(context.Background(), srv.Client(), srv.URL); err == nil {
		t.Error("want error on non-200, got nil")
	}

	// context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := latestFrom(ctx, http.DefaultClient, "https://api.github.com/repos/thomaschampagne/zwiftboard/releases/latest"); err == nil {
		t.Error("want error on cancelled context, got nil")
	}
}
