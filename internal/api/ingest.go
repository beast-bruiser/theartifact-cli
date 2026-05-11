package api

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

type UploadRequest struct {
	Count    int    `json:"count"`
	MimeType string `json:"mime_type,omitempty"` // defaults to image/png
}

type UploadToken struct {
	UploadToken string `json:"upload_token"`
	URL         string `json:"url"`
	ExpiresAt   string `json:"expires_at"`
}

type UploadResponse struct {
	Uploads []UploadToken `json:"uploads"`
}

type IngestRequest struct {
	WorkspaceID string   `json:"workspace_id"`
	Sources     []string `json:"sources"`
	WebhookURL  string   `json:"webhook_url,omitempty"` // HTTPS callback on terminal state
}

type IngestResponse struct {
	JobID string `json:"job_id"`
}

// UploadFiles PUTs each file to its presigned URL and returns the upload tokens.
func UploadFiles(uploads []UploadToken, filePaths []string, mimeType string) ([]string, error) {
	if len(uploads) != len(filePaths) {
		return nil, fmt.Errorf("uploads/files length mismatch: %d vs %d", len(uploads), len(filePaths))
	}
	httpClient := &http.Client{}
	tokens := make([]string, 0, len(filePaths))
	for i, path := range filePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("cannot read file %s: %w", path, err)
		}
		req, err := http.NewRequest(http.MethodPut, uploads[i].URL, bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("cannot create request for %s: %w", path, err)
		}
		req.Header.Set("Content-Type", mimeType)
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("upload failed for %s: %w", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("upload rejected for %s (HTTP %d)", path, resp.StatusCode)
		}
		tokens = append(tokens, uploads[i].UploadToken)
	}
	return tokens, nil
}

const maxUploadBatch = 10

// RequestUploadURLs requests presigned upload slots from the API, automatically
// chunking into batches of 10 (the server-side maximum) when count > 10.
func (c *Client) RequestUploadURLs(count int, mimeType string) ([]UploadToken, error) {
	var all []UploadToken
	remaining := count
	for remaining > 0 {
		batch := remaining
		if batch > maxUploadBatch {
			batch = maxUploadBatch
		}
		tokens, err := c.requestUploadURLsBatch(batch, mimeType)
		if err != nil {
			return nil, err
		}
		all = append(all, tokens...)
		remaining -= batch
	}
	return all, nil
}

func (c *Client) requestUploadURLsBatch(count int, mimeType string) ([]UploadToken, error) {
	var upResp UploadResponse
	req := UploadRequest{Count: count, MimeType: mimeType}

	resp, err := c.HTTPClient.R().
		SetBody(req).
		SetResult(&upResp).
		Post("/v1/uploads")

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.IsError() {
		apiErr, ok := resp.Error().(*APIError)
		if ok && apiErr.Message != "" {
			return nil, fmt.Errorf("API error (%s): %s", apiErr.Code, apiErr.Message)
		}
		return nil, fmt.Errorf("API request failed with status: %s", resp.Status())
	}

	return upResp.Uploads, nil
}

func (c *Client) TriggerIngest(req IngestRequest) (string, error) {
	var inResp IngestResponse
	resp, err := c.HTTPClient.R().
		SetBody(req).
		SetResult(&inResp).
		Post("/v1/ingest")

	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}

	if resp.IsError() {
		apiErr, ok := resp.Error().(*APIError)
		if ok && apiErr.Message != "" {
			return "", fmt.Errorf("API error (%s): %s", apiErr.Code, apiErr.Message)
		}
		return "", fmt.Errorf("API request failed with status: %s", resp.Status())
	}

	return inResp.JobID, nil
}
