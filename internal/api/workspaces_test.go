package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestListWorkspaces_Empty verifies handling of empty workspace list.
func TestListWorkspaces_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/workspaces" {
			t.Errorf("wrong path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	workspaces, err := client.ListWorkspaces()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(workspaces) != 0 {
		t.Errorf("expected 0 workspaces, got %d", len(workspaces))
	}
}

// TestListWorkspaces_Single verifies parsing a single workspace.
func TestListWorkspaces_Single(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":            "ws_test123",
				"name":          "My Workspace",
				"style_summary": "Dark fantasy aesthetic",
			},
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	workspaces, err := client.ListWorkspaces()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}

	if workspaces[0].ID != "ws_test123" {
		t.Errorf("workspace ID mismatch: got %q, want %q", workspaces[0].ID, "ws_test123")
	}

	if workspaces[0].Name != "My Workspace" {
		t.Errorf("workspace name mismatch: got %q, want %q", workspaces[0].Name, "My Workspace")
	}

	if workspaces[0].StyleSummary != "Dark fantasy aesthetic" {
		t.Errorf("workspace style summary mismatch: got %q", workspaces[0].StyleSummary)
	}
}

// TestListWorkspaces_Multiple verifies parsing multiple workspaces.
func TestListWorkspaces_Multiple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":            "ws_001",
				"name":          "Fantasy World",
				"style_summary": "Medieval fantasy",
			},
			{
				"id":            "ws_002",
				"name":          "Sci-Fi Universe",
				"style_summary": "Cyberpunk aesthetic",
			},
			{
				"id":            "ws_003",
				"name":          "Contemporary",
				"style_summary": "Modern urban",
			},
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	workspaces, err := client.ListWorkspaces()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(workspaces) != 3 {
		t.Fatalf("expected 3 workspaces, got %d", len(workspaces))
	}

	expectedNames := []string{"Fantasy World", "Sci-Fi Universe", "Contemporary"}
	for i, ws := range workspaces {
		if ws.Name != expectedNames[i] {
			t.Errorf("workspace %d name mismatch: got %q, want %q", i, ws.Name, expectedNames[i])
		}
	}
}

// TestListWorkspaces_WithAuthHeader verifies Bearer token is sent.
func TestListWorkspaces_WithAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("Authorization header is missing")
		}
		if auth != "Bearer test_api_key" {
			t.Errorf("authorization header mismatch: got %q", auth)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	_, err := client.ListWorkspaces()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestListWorkspaces_APIError verifies error handling.
func TestListWorkspaces_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "unauthorized",
			"message": "Invalid API key",
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	workspaces, err := client.ListWorkspaces()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if workspaces != nil {
		t.Errorf("expected nil workspaces on error, got %v", workspaces)
	}
}

// TestWorkspaceStructure verifies workspace struct fields.
func TestWorkspaceStructure(t *testing.T) {
	ws := Workspace{
		ID:           "ws_test",
		Name:         "Test Workspace",
		StyleSummary: "Test Style",
	}

	if ws.ID != "ws_test" {
		t.Errorf("workspace ID mismatch")
	}

	if ws.Name != "Test Workspace" {
		t.Errorf("workspace name mismatch")
	}

	if ws.StyleSummary != "Test Style" {
		t.Errorf("workspace style summary mismatch")
	}
}
