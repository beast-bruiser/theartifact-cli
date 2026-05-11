package cmd

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"theartifact-cli/internal/api"
	"theartifact-cli/internal/config"
)

func testClient(baseURL string) *api.Client {
	httpClient := resty.New()
	httpClient.SetBaseURL(baseURL)
	httpClient.SetAuthToken("test_key")
	httpClient.SetHeader("Content-Type", "application/json")
	httpClient.SetError(&api.APIError{})
	return &api.Client{HTTPClient: httpClient}
}

// runWorkspaceGate: --no-prompt without --name/--link should fail fast.
func TestWorkspaceGate_NoPromptRequiresInput(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(""))
	cli := testClient("http://unused")
	_, err := runWorkspaceGate(scanner, cli, &config.ProjectConfig{}, "/tmp/foo", workspaceGateOptions{NoPrompt: true})
	if err == nil {
		t.Fatal("expected error when --no-prompt has no --name or --link, got nil")
	}
	if !strings.Contains(err.Error(), "--no-prompt") {
		t.Errorf("error should mention --no-prompt, got: %v", err)
	}
}

// runWorkspaceGate: --link binds directly without calling the API.
func TestWorkspaceGate_LinkBindsDirectly(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(""))
	// Use a server that fails every call — confirms --link doesn't hit the API.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("--link path must not call API; got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	cli := testClient(server.URL)
	out, err := runWorkspaceGate(scanner, cli, &config.ProjectConfig{}, "/tmp/foo", workspaceGateOptions{LinkID: "ws_existing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.WorkspaceID != "ws_existing" {
		t.Errorf("expected WorkspaceID=ws_existing, got %q", out.WorkspaceID)
	}
}

// runWorkspaceGate: --no-prompt --name with zero existing workspaces auto-creates.
func TestWorkspaceGate_NoPromptCreate(t *testing.T) {
	var created bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/workspaces":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/workspaces":
			created = true
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["name"] != "ci-project" {
				t.Errorf("expected name=ci-project, got %q", body["name"])
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"id":   "ws_new",
				"name": "ci-project",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	scanner := bufio.NewScanner(strings.NewReader(""))
	cli := testClient(server.URL)
	out, err := runWorkspaceGate(scanner, cli, &config.ProjectConfig{}, "/tmp/foo", workspaceGateOptions{
		Name:     "ci-project",
		NoPrompt: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected POST /v1/workspaces to be called")
	}
	if out.WorkspaceID != "ws_new" || out.WorkspaceName != "ci-project" {
		t.Errorf("unexpected projCfg: %+v", out)
	}
}

// runWorkspaceGate: --no-prompt --name with existing workspaces still creates
// (link-existing branch only fires interactively).
func TestWorkspaceGate_NoPromptCreate_WithExisting(t *testing.T) {
	var created bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/workspaces":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]string{
				{"id": "ws_old", "name": "old"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/workspaces":
			created = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"id":   "ws_brand_new",
				"name": "brand-new",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	scanner := bufio.NewScanner(strings.NewReader(""))
	cli := testClient(server.URL)
	out, err := runWorkspaceGate(scanner, cli, &config.ProjectConfig{}, "/tmp/foo", workspaceGateOptions{
		Name:     "brand-new",
		NoPrompt: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected workspace creation even when others exist")
	}
	if out.WorkspaceID != "ws_brand_new" {
		t.Errorf("expected ws_brand_new, got %q", out.WorkspaceID)
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{2048, "2.0 KB"},
		{1024 * 1024 * 5, "5.0 MB"},
	}
	for _, tc := range cases {
		if got := formatBytes(tc.in); got != tc.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
