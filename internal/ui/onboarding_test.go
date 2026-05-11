package ui

import (
	"bufio"
	"strings"
	"testing"
)

func TestDeriveWorkspaceName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"/home/me/cyberpunk-art", "cyberpunk-art"},
		{"/tmp/My Project", "My Project"},
		{"/tmp/dark_fantasy_v2", "dark-fantasy-v2"},
		{"/tmp/🚀-rockets", "rockets"},
		{"/tmp/...", "my-workspace"},
		{"/tmp/", "tmp"},
		{"/var/folders/x/tmp.XyZ123", "tmp-XyZ123"},
		{"/tmp/with.dots.everywhere", "with-dots-everywhere"},
	}
	for _, tc := range cases {
		got := DeriveWorkspaceName(tc.in)
		if got != tc.want {
			t.Errorf("DeriveWorkspaceName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPromptWorkspaceName_EnterKeepsSuggested(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("\n"))
	got, err := PromptWorkspaceName(scanner, "my-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "my-project" {
		t.Errorf("expected suggested name kept, got %q", got)
	}
}

func TestPromptWorkspaceName_UserOverride(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("custom-name\n"))
	got, err := PromptWorkspaceName(scanner, "my-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "custom-name" {
		t.Errorf("expected user override, got %q", got)
	}
}

func TestPromptCreateOrLink(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"1\n", true},   // create new
		{"\n", true},    // default = create new
		{"2\n", false},  // link existing
		{"x\n", true},   // anything not "2" = create new
	}
	for _, tc := range cases {
		scanner := bufio.NewScanner(strings.NewReader(tc.input))
		got := PromptCreateOrLink(scanner, "folder")
		if got != tc.want {
			t.Errorf("PromptCreateOrLink(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
