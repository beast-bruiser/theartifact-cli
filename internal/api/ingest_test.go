package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRequestUploadURLs_SingleFile verifies requesting upload URL for a single file.
func TestRequestUploadURLs_SingleFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/uploads" {
			t.Errorf("wrong path: %s", r.URL.Path)
		}

		// Verify request body
		var uploadReq UploadRequest
		if err := json.NewDecoder(r.Body).Decode(&uploadReq); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if uploadReq.Count != 1 {
			t.Errorf("count mismatch: got %d, want %d", uploadReq.Count, 1)
		}
		if uploadReq.MimeType != "image/png" {
			t.Errorf("mime type mismatch: got %q, want %q", uploadReq.MimeType, "image/png")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uploads": []map[string]interface{}{
				{
					"upload_token": "token_abc123",
					"url":          "https://s3.presigned.example.com/upload1?sig=xyz",
					"expires_at":   "2026-05-04T12:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	tokens, err := client.RequestUploadURLs(1, "image/png")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}

	if tokens[0].UploadToken != "token_abc123" {
		t.Errorf("upload token mismatch: got %q", tokens[0].UploadToken)
	}

	if tokens[0].URL != "https://s3.presigned.example.com/upload1?sig=xyz" {
		t.Errorf("URL mismatch: got %q", tokens[0].URL)
	}

	if tokens[0].ExpiresAt != "2026-05-04T12:00:00Z" {
		t.Errorf("expires at mismatch: got %q", tokens[0].ExpiresAt)
	}
}

// TestRequestUploadURLs_MultipleFiles verifies requesting upload URLs for multiple files.
func TestRequestUploadURLs_MultipleFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var uploadReq UploadRequest
		json.NewDecoder(r.Body).Decode(&uploadReq)

		if uploadReq.Count != 3 {
			t.Errorf("count mismatch: got %d, want %d", uploadReq.Count, 3)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uploads": []map[string]interface{}{
				{
					"upload_token": "token_001",
					"url":          "https://s3.example.com/upload1",
					"expires_at":   "2026-05-04T12:00:00Z",
				},
				{
					"upload_token": "token_002",
					"url":          "https://s3.example.com/upload2",
					"expires_at":   "2026-05-04T12:00:00Z",
				},
				{
					"upload_token": "token_003",
					"url":          "https://s3.example.com/upload3",
					"expires_at":   "2026-05-04T12:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	tokens, err := client.RequestUploadURLs(3, "image/jpeg")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}

	expectedTokens := []string{"token_001", "token_002", "token_003"}
	for i, token := range tokens {
		if token.UploadToken != expectedTokens[i] {
			t.Errorf("token %d mismatch: got %q, want %q", i, token.UploadToken, expectedTokens[i])
		}
	}
}

// TestRequestUploadURLs_CustomMimeType verifies custom MIME types are sent.
func TestRequestUploadURLs_CustomMimeType(t *testing.T) {
	customMimeType := "image/webp"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var uploadReq UploadRequest
		json.NewDecoder(r.Body).Decode(&uploadReq)

		if uploadReq.MimeType != customMimeType {
			t.Errorf("mime type mismatch: got %q, want %q", uploadReq.MimeType, customMimeType)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uploads": []map[string]interface{}{
				{
					"upload_token": "token_custom",
					"url":          "https://s3.example.com/upload",
					"expires_at":   "2026-05-04T12:00:00Z",
				},
			},
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	tokens, err := client.RequestUploadURLs(1, customMimeType)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
}

// TestRequestUploadURLs_APIError verifies error handling.
func TestRequestUploadURLs_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "invalid_request",
			"message": "Count must be between 1 and 10",
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	tokens, err := client.RequestUploadURLs(100, "image/png")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if tokens != nil {
		t.Errorf("expected nil tokens on error, got %v", tokens)
	}
}

// TestTriggerIngest verifies the ingest trigger call.
func TestTriggerIngest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/ingest" {
			t.Errorf("wrong path: %s", r.URL.Path)
		}

		// Verify request body
		var ingestReq IngestRequest
		if err := json.NewDecoder(r.Body).Decode(&ingestReq); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if ingestReq.WorkspaceID != "ws_test" {
			t.Errorf("workspace ID mismatch: got %q", ingestReq.WorkspaceID)
		}
		if len(ingestReq.Sources) != 2 {
			t.Errorf("sources count mismatch: got %d", len(ingestReq.Sources))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id": "job_ingest_123",
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	jobID, err := client.TriggerIngest(IngestRequest{
		WorkspaceID: "ws_test",
		Sources:     []string{"token_001", "token_002"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if jobID != "job_ingest_123" {
		t.Errorf("job ID mismatch: got %q, want %q", jobID, "job_ingest_123")
	}
}

// TestTriggerIngest_WithWebhook verifies webhook URL in request.
func TestTriggerIngest_WithWebhook(t *testing.T) {
	webhookURL := "https://example.com/webhook"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ingestReq IngestRequest
		json.NewDecoder(r.Body).Decode(&ingestReq)

		if ingestReq.WebhookURL != webhookURL {
			t.Errorf("webhook URL mismatch: got %q, want %q", ingestReq.WebhookURL, webhookURL)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id": "job_webhook",
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	jobID, err := client.TriggerIngest(IngestRequest{
		WorkspaceID: "ws_test",
		Sources:     []string{"token_1"},
		WebhookURL:  webhookURL,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if jobID != "job_webhook" {
		t.Errorf("job ID mismatch: got %q", jobID)
	}
}

// TestTriggerIngest_APIError verifies error handling.
func TestTriggerIngest_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    "invalid_request",
			"message": "Workspace not found",
		})
	}))
	defer server.Close()

	client := newTestAPIClient(server.URL)
	jobID, err := client.TriggerIngest(IngestRequest{
		WorkspaceID: "ws_nonexistent",
		Sources:     []string{"token_1"},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if jobID != "" {
		t.Errorf("expected empty job ID on error, got %q", jobID)
	}
}

// TestUploadToken_Structure verifies UploadToken fields.
func TestUploadToken_Structure(t *testing.T) {
	token := UploadToken{
		UploadToken: "token_xyz",
		URL:         "https://presigned.example.com",
		ExpiresAt:   "2026-05-04T12:00:00Z",
	}

	if token.UploadToken != "token_xyz" {
		t.Errorf("upload token mismatch")
	}

	if token.URL != "https://presigned.example.com" {
		t.Errorf("URL mismatch")
	}

	if token.ExpiresAt != "2026-05-04T12:00:00Z" {
		t.Errorf("expires at mismatch")
	}
}

// TestIngestRequest_Structure verifies IngestRequest fields.
func TestIngestRequest_Structure(t *testing.T) {
	req := IngestRequest{
		WorkspaceID: "ws_test",
		Sources:     []string{"token_1", "token_2"},
		WebhookURL:  "https://example.com/webhook",
	}

	if req.WorkspaceID != "ws_test" {
		t.Errorf("workspace ID mismatch")
	}

	if len(req.Sources) != 2 {
		t.Errorf("sources count mismatch: got %d, want %d", len(req.Sources), 2)
	}

	if req.WebhookURL != "https://example.com/webhook" {
		t.Errorf("webhook URL mismatch")
	}
}
