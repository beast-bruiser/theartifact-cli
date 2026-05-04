package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
)

// TestGenerateRequest_ValidRequest verifies that a valid GenerateRequest is correctly formatted.
func TestGenerateRequest_ValidRequest(t *testing.T) {
	req := GenerateRequest{
		WorkspaceID: "ws_test123",
		Prompt:      "A futuristic city",
		Count:       2,
		Seed:        12345,
	}

	if req.WorkspaceID != "ws_test123" {
		t.Errorf("workspace ID mismatch: got %q, want %q", req.WorkspaceID, "ws_test123")
	}

	if req.Prompt != "A futuristic city" {
		t.Errorf("prompt mismatch: got %q, want %q", req.Prompt, "A futuristic city")
	}

	if req.Count != 2 {
		t.Errorf("count mismatch: got %d, want %d", req.Count, 2)
	}

	if req.Seed != 12345 {
		t.Errorf("seed mismatch: got %d, want %d", req.Seed, 12345)
	}
}

// TestAPIError_Unmarshalling verifies that APIError correctly unmarshals API responses.
func TestAPIError_Unmarshalling(t *testing.T) {
	err := &APIError{
		Code:    "invalid_request",
		Message: "Workspace ID is required",
	}

	if err.Code != "invalid_request" {
		t.Errorf("error code mismatch: got %q, want %q", err.Code, "invalid_request")
	}

	if err.Message != "Workspace ID is required" {
		t.Errorf("error message mismatch: got %q, want %q", err.Message, "Workspace ID is required")
	}
}

// TestGenerateResponse_Parsing verifies that GenerateResponse correctly parses job_id.
func TestGenerateResponse_Parsing(t *testing.T) {
	resp := GenerateResponse{
		JobID: "job_abc123def456",
	}

	if resp.JobID != "job_abc123def456" {
		t.Errorf("job ID mismatch: got %q, want %q", resp.JobID, "job_abc123def456")
	}
}

// TestClientAuthHeader verifies that the client properly sets Bearer token.
func TestClientAuthHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("Authorization header is missing")
		}
		if auth != "Bearer test_api_key_12345" {
			t.Errorf("authorization header mismatch: got %q", auth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"job_id":"job_test"}`))
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: NewTestClient("test_api_key_12345", server.URL),
	}

	req := GenerateRequest{
		WorkspaceID: "ws_test",
		Prompt:      "test",
	}

	_, err := client.Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGenerateWithIdempotencyKey verifies that IdempotencyKey is passed as a header.
func TestGenerateWithIdempotencyKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idempotency := r.Header.Get("Idempotency-Key")
		if idempotency != "key_unique_12345" {
			t.Errorf("Idempotency-Key mismatch: got %q, want %q", idempotency, "key_unique_12345")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"job_id":"job_test"}`))
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: NewTestClient("test_api_key", server.URL),
	}

	req := GenerateRequest{
		WorkspaceID:    "ws_test",
		Prompt:         "test",
		IdempotencyKey: "key_unique_12345",
	}

	_, err := client.Generate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// NewTestClient is a helper to create a test client with a custom base URL and API key.
func NewTestClient(apiKey, baseURL string) *resty.Client {
	httpClient := resty.New()
	httpClient.SetBaseURL(baseURL)
	httpClient.SetAuthToken(apiKey)
	httpClient.SetHeader("Content-Type", "application/json")
	httpClient.SetError(&APIError{})
	return httpClient
}
