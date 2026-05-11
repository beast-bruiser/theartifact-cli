package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestCreateWorkspace_Success verifies a 201 response is parsed correctly.
func TestCreateWorkspace_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/workspaces" {
			t.Errorf("wrong path: %s", r.URL.Path)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["name"] != "cyberpunk-art" {
			t.Errorf("expected name=cyberpunk-art, got %q", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":            "ws_abc123",
			"name":          "cyberpunk-art",
			"style_summary": "",
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	ws, err := client.CreateWorkspace("cyberpunk-art")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.ID != "ws_abc123" {
		t.Errorf("workspace ID mismatch: got %q", ws.ID)
	}
	if ws.Name != "cyberpunk-art" {
		t.Errorf("workspace name mismatch: got %q", ws.Name)
	}
}

// TestCreateWorkspace_Unauthorized verifies 401 maps to a helpful error.
func TestCreateWorkspace_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "unauthorized",
			"message": "API key auth not supported on this endpoint",
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	ws, err := client.CreateWorkspace("anything")

	if err == nil {
		t.Fatal("expected error on 401, got nil")
	}
	if ws != nil {
		t.Errorf("expected nil workspace on error, got %v", ws)
	}
}

// TestCreateWorkspace_QuotaExceeded verifies 429 error envelope is surfaced.
func TestCreateWorkspace_QuotaExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "quota_exceeded",
			"message": "Workspace quota reached",
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	_, err := client.CreateWorkspace("anything")

	if err == nil {
		t.Fatal("expected error on 429, got nil")
	}
	if !strings.Contains(err.Error(), "quota_exceeded") {
		t.Errorf("expected quota_exceeded in error, got: %v", err)
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
