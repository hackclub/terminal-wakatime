package tracker

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hackclub/terminal-wakatime/pkg/config"
)

func TestSplitCommandLine(t *testing.T) {
	tests := []struct {
		line string
		want []string
	}{
		{"vim main.go", []string{"vim main.go"}},
		{"cd api && vim main.go", []string{"cd api", "vim main.go"}},
		{"make || echo failed", []string{"make", "echo failed"}},
		{"echo first; echo second", []string{"echo first", "echo second"}},
		{"ls -la | wc -l", []string{"ls -la", "wc -l"}},
		{"make 2>&1 | tee build.log", []string{"make 2>&1", "tee build.log"}},
		{"npm run dev &> dev.log &", []string{"npm run dev &> dev.log"}},
		{"(cd api; make)", []string{"cd api", "make"}},
		{"{ cd api; make; }", []string{"cd api", "make"}},
		{"for f in *.go; do vim $f; done", []string{"vim $f"}},
		{"if make; then ./run; fi", []string{"make", "./run"}},
		{"while true; do curl localhost; sleep 1; done", []string{"true", "curl localhost", "sleep 1"}},
		{`echo "a && b; c"`, []string{`echo "a && b; c"`}},
		{`echo 'a | b'`, []string{`echo 'a | b'`}},
		{`echo a\;b`, []string{`echo a\;b`}},
		{"git add .\ngit commit", []string{"git add .", "git commit"}},
		{"", nil},
		{" ; ", nil},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := splitCommandLine(tt.line); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitCommandLine(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestParseCommandLineWithSeveralCommands(t *testing.T) {
	cfg := &config.Config{Project: "test-project", DisableEditorSuggestions: true}
	tracker := NewTracker(cfg)

	home := t.TempDir()
	t.Setenv("HOME", home)

	root := t.TempDir()
	api := filepath.Join(root, "api")
	if err := os.MkdirAll(api, 0755); err != nil {
		t.Fatal(err)
	}
	mainFile := filepath.Join(api, "main.go")
	if err := os.WriteFile(mainFile, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		line       string
		entity     string
		entityType ActivityType
	}{
		// The editor wins over cd, and opens the file relative to the new directory
		{"cd api && vim main.go", mainFile, ActivityFile},
		{"(cd api; vim main.go)", mainFile, ActivityFile},
		{"cd " + api + " && vim main.go", mainFile, ActivityFile},
		// Commands after the first are no longer ignored
		{"cd api && make build", "make build", ActivityApp},
		{"cd api && ls", "ls", ActivityApp},
		{"ls && git status", "git status", ActivityApp},
		// A line with a single command behaves as before
		{"cd api", api, ActivityFile},
		{"cd ~", home, ActivityFile},
		{"cd ~/api", filepath.Join(home, "api"), ActivityFile},
		{"git push", "git push", ActivityApp},
		// On a tie, the first command is used
		{"make build && make test", "make build", ActivityApp},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			activity := tracker.parseCommandToSingleActivity(tt.line, root)
			if activity == nil {
				t.Fatal("expected an activity")
			}
			if activity.Entity != tt.entity || activity.EntityType != tt.entityType {
				t.Errorf("got %s %q, want %s %q", activity.EntityType, activity.Entity, tt.entityType, tt.entity)
			}
		})
	}
}
