package wakatime

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hackclub/terminal-wakatime/pkg/config"
)

func TestParseVersion(t *testing.T) {
	tests := map[string]string{
		"v2.27.13\n":             "v2.27.13", // current wakatime-cli
		"wakatime-cli v1.73.0\n": "v1.73.0",  // older wakatime-cli
	}
	for output, want := range tests {
		if got, err := parseVersion(output); err != nil || got != want {
			t.Errorf("parseVersion(%q) = %q, %v; want %q", output, got, err, want)
		}
	}

	if _, err := parseVersion("garbage\n"); err == nil {
		t.Error("expected an error for unparsable output")
	}
}

// updateCheckFixture is a CLI whose binary prints installedVersion and whose
// release lookups go to a local server answering with latestStatus/latestTag.
func updateCheckFixture(t *testing.T, installedVersion string, latestStatus int, latestTag string) (*CLI, *int32) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script as a fake wakatime-cli")
	}

	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(latestStatus)
		fmt.Fprintf(w, `{"tag_name": %q, "assets": []}`, latestTag)
	}))
	t.Cleanup(server.Close)

	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg.WakaTimeDir(), 0755); err != nil {
		t.Fatal(err)
	}

	bin := filepath.Join(cfg.WakaTimeDir(), "wakatime-cli")
	script := fmt.Sprintf("#!/bin/sh\n[ \"$1\" = \"--version\" ] && echo %q\nexit 0\n", installedVersion)
	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	// Last checked two days ago, so a check is due.
	stale := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	if err := os.WriteFile(filepath.Join(cfg.WakaTimeDir(), "last_update_check"), []byte(stale), 0644); err != nil {
		t.Fatal(err)
	}

	return &CLI{config: cfg, binPath: bin, releasesURL: server.URL}, &requests
}

func TestUpdateCheckRunsOncePerInterval(t *testing.T) {
	tests := []struct {
		name         string
		installed    string
		latestStatus int
	}{
		{"current version format", "v2.27.13", http.StatusOK},
		{"old version format", "wakatime-cli v2.27.13", http.StatusOK},
		{"GitHub unavailable", "v2.27.13", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli, requests := updateCheckFixture(t, tt.installed, tt.latestStatus, "v2.27.13")

			for i := 0; i < 3; i++ {
				if err := cli.checkForUpdates(); err != nil {
					t.Fatalf("checkForUpdates() failed: %v", err)
				}
			}

			if got := atomic.LoadInt32(requests); got != 1 {
				t.Errorf("3 commands made %d release lookups, want 1 per day", got)
			}
			if time.Since(cli.getLastUpdateCheck()) > time.Minute {
				t.Errorf("last update check was not recorded: %v", cli.getLastUpdateCheck())
			}
		})
	}
}

func TestUpdateCheckUpdatesOutdatedVersion(t *testing.T) {
	cli, requests := updateCheckFixture(t, "v2.20.0", http.StatusOK, "v2.27.13")

	if err := cli.checkForUpdates(); err != nil {
		t.Fatalf("checkForUpdates() failed: %v", err)
	}

	// One lookup for the check, and a second from install() fetching the
	// release to download, which shows the outdated version was detected.
	if got := atomic.LoadInt32(requests); got != 2 {
		t.Errorf("made %d release lookups, want 2 (check + install)", got)
	}
}
