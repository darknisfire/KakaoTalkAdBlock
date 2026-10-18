package internal

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestParseLatestTag(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		want   string
		wantOK bool
	}{
		{"valid", `{"tag_name":"2.3.0","name":"release"}`, "2.3.0", true},
		{"rate limit response", `{"message":"API rate limit exceeded for 1.2.3.4.","documentation_url":"https://docs.github.com/rest"}`, "", false},
		{"numeric tag_name", `{"tag_name":123}`, "", false},
		{"null tag_name", `{"tag_name":null}`, "", false},
		{"empty tag_name", `{"tag_name":""}`, "", false},
		{"malformed json", `{"tag_name":`, "", false},
		{"empty body", ``, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseLatestTag([]byte(tt.body))
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("parseLatestTag(%q) = (%q, %v), want (%q, %v)", tt.body, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestHasNewRelease(t *testing.T) {
	tests := []struct {
		name            string
		current, latest string
		want            bool
	}{
		{"same", "2.2.3", "2.2.3", false},
		{"major up", "2.2.3", "3.2.3", true},
		{"major down", "3.2.3", "2.2.3", false},
		{"minor up", "2.2.3", "2.3.3", true},
		{"minor down", "2.3.3", "2.2.3", false},
		{"patch up", "2.2.3", "2.2.4", true},
		{"patch down", "2.2.4", "2.2.3", false},
		{"different length, same prefix", "2.2.3", "2.3", true},
		{"different length, equal prefix", "2.3", "2.3.1", false},
		{"non-numeric latest", "2.2.3", "v2.2.3", false},
		{"non-numeric current", "v2.2.3", "2.2.4", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasNewRelease(tt.current, tt.latest); got != tt.want {
				t.Errorf("hasNewRelease(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}
func TestCurlPath(t *testing.T) {
	p := curlPath()
	if !filepath.IsAbs(p) || !strings.HasSuffix(strings.ToLower(p), `\curl.exe`) {
		t.Fatalf("curlPath() = %q, want absolute path ending in curl.exe", p)
	}
	if _, err := os.Stat(p); err != nil {
		t.Skipf("curl.exe not present: %v", err)
	}
}

func TestCheckLatestVersionMissingCurl(t *testing.T) {
	old := curlExe
	curlExe = filepath.Join(t.TempDir(), "missing-curl.exe")
	t.Cleanup(func() { curlExe = old })

	start := time.Now()
	got, ok := CheckLatestVersion(context.Background(), "2.2.3")
	if got != "2.2.3" || ok {
		t.Errorf("CheckLatestVersion() = (%q, %v), want (\"2.2.3\", false)", got, ok)
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("took %v, want under 1s", d)
	}
}

func TestCheckLatestVersionOnline(t *testing.T) {
	if os.Getenv("KTAB_ONLINE") != "1" {
		t.Skip("set KTAB_ONLINE=1 to query GitHub")
	}
	done := make(chan struct{})
	var tag string
	var ok bool
	go func() {
		defer close(done)
		tag, ok = CheckLatestVersion(context.Background(), "0.0.1")
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("CheckLatestVersion did not return within 10s")
	}
	if ok && !regexp.MustCompile(`^\d+(\.\d+)+$`).MatchString(tag) {
		t.Errorf("CheckLatestVersion() = (%q, true), tag does not look like a version", tag)
	}
}
