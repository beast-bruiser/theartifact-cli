package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
)

// newTestAPIClient creates a test client with a custom base URL.
func newTestAPIClient(baseURL string) *Client {
	httpClient := resty.New()
	httpClient.SetBaseURL(baseURL)
	httpClient.SetAuthToken("test_api_key")
	httpClient.SetHeader("Content-Type", "application/json")
	httpClient.SetError(&APIError{})
	return &Client{HTTPClient: httpClient}
}

// TestGetJob_Queued verifies that a queued job is correctly parsed.
func TestGetJob_Queued(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/jobs/job_test123" {
			t.Errorf("wrong path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "job_test123",
			"status":   "queued",
			"progress": 0,
			"kind":     "generation",
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	job, err := client.GetJob("job_test123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if job.ID != "job_test123" {
		t.Errorf("job ID mismatch: got %q, want %q", job.ID, "job_test123")
	}

	if job.Status != "queued" {
		t.Errorf("job status mismatch: got %q, want %q", job.Status, "queued")
	}

	if job.Progress != 0 {
		t.Errorf("job progress mismatch: got %d, want %d", job.Progress, 0)
	}
}

// TestGetJob_Running verifies job in running state.
func TestGetJob_Running(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "job_running",
			"status":   "running",
			"progress": 45,
			"kind":     "generation",
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	job, err := client.GetJob("job_running")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if job.Status != "running" {
		t.Errorf("job status mismatch: got %q, want %q", job.Status, "running")
	}

	if job.Progress != 45 {
		t.Errorf("job progress mismatch: got %d, want %d", job.Progress, 45)
	}
}

// TestGetJob_Succeeded verifies successful job with results.
func TestGetJob_Succeeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "job_success",
			"status":   "succeeded",
			"progress": 100,
			"result": map[string]interface{}{
				"manifest": map[string]interface{}{
					"prompt": "A futuristic city",
					"model":  "artifact-v1",
					"seed":   12345,
				},
				"assets": []map[string]interface{}{
					{
						"id":         "asset_001",
						"url":        "https://presigned.example.com/asset1.png",
						"expires_at": "2026-05-04T15:00:00Z",
					},
					{
						"id":         "asset_002",
						"url":        "https://presigned.example.com/asset2.png",
						"expires_at": "2026-05-04T15:00:00Z",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	job, err := client.GetJob("job_success")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if job.Status != "succeeded" {
		t.Errorf("job status mismatch: got %q, want %q", job.Status, "succeeded")
	}

	if job.Progress != 100 {
		t.Errorf("job progress mismatch: got %d, want %d", job.Progress, 100)
	}

	if job.Result == nil {
		t.Fatal("job result is nil")
	}

	if job.Result.Manifest.Prompt != "A futuristic city" {
		t.Errorf("manifest prompt mismatch: got %q", job.Result.Manifest.Prompt)
	}

	if len(job.Result.Assets) != 2 {
		t.Errorf("asset count mismatch: got %d, want %d", len(job.Result.Assets), 2)
	}

	if job.Result.Assets[0].ID != "asset_001" {
		t.Errorf("asset ID mismatch: got %q", job.Result.Assets[0].ID)
	}
}

// TestGetJob_Failed verifies failed job with error message.
func TestGetJob_Failed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "job_failed",
			"status":   "failed",
			"progress": 0,
			"error":    "quota_exceeded",
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	job, err := client.GetJob("job_failed")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if job.Status != "failed" {
		t.Errorf("job status mismatch: got %q, want %q", job.Status, "failed")
	}

	if job.Error != "quota_exceeded" {
		t.Errorf("job error mismatch: got %q, want %q", job.Error, "quota_exceeded")
	}
}

// TestGetJob_APIError verifies error handling for API errors.
func TestGetJob_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "not_found",
			"message": "Job not found",
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	job, err := client.GetJob("job_nonexistent")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if job != nil {
		t.Errorf("expected nil job on error, got %v", job)
	}

	// The error should mention that the request failed
	if errMsg := err.Error(); errMsg == "" {
		t.Errorf("error message is empty")
	}
}

// TestJobResult_WithGraphFragment verifies parsing of graph data.
func TestJobResult_WithGraphFragment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "job_with_graph",
			"status":   "succeeded",
			"progress": 100,
			"result": map[string]interface{}{
				"manifest": map[string]interface{}{
					"prompt": "test",
					"model":  "artifact-v1",
				},
				"assets": []map[string]interface{}{},
				"graph_fragment": map[string]interface{}{
					"nodes": []map[string]interface{}{
						{
							"id":    "node_1",
							"kind":  "entity",
							"label": "Forest",
						},
					},
					"edges": []map[string]interface{}{
						{
							"from":  "node_1",
							"to":    "node_2",
							"label": "contains",
						},
					},
					"version": 1,
				},
			},
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	job, err := client.GetJob("job_with_graph")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if job.Result.GraphFragment == nil {
		t.Fatal("graph fragment is nil")
	}

	if len(job.Result.GraphFragment.Nodes) != 1 {
		t.Errorf("node count mismatch: got %d", len(job.Result.GraphFragment.Nodes))
	}

	if job.Result.GraphFragment.Nodes[0].Label != "Forest" {
		t.Errorf("node label mismatch: got %q", job.Result.GraphFragment.Nodes[0].Label)
	}

	if len(job.Result.GraphFragment.Edges) != 1 {
		t.Errorf("edge count mismatch: got %d", len(job.Result.GraphFragment.Edges))
	}

	if job.Result.GraphFragment.Version != 1 {
		t.Errorf("graph version mismatch: got %d", job.Result.GraphFragment.Version)
	}
}

// TestJobResult_WithBrainFiles verifies parsing of brain files.
func TestJobResult_WithBrainFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "job_with_brain",
			"status":   "succeeded",
			"progress": 100,
			"result": map[string]interface{}{
				"manifest": map[string]interface{}{
					"prompt": "test",
				},
				"assets": []map[string]interface{}{},
				"brain_files": []map[string]interface{}{
					{
						"path":    "characters.json",
						"content": `{"name":"Alice"}`,
						"sha":     "abc123def456",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	job, err := client.GetJob("job_with_brain")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(job.Result.BrainFiles) != 1 {
		t.Errorf("brain file count mismatch: got %d", len(job.Result.BrainFiles))
	}

	if job.Result.BrainFiles[0].Path != "characters.json" {
		t.Errorf("brain file path mismatch: got %q", job.Result.BrainFiles[0].Path)
	}
}
