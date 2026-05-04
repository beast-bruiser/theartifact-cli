package api

import "fmt"

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

func (c *Client) RequestUploadURLs(count int, mimeType string) ([]UploadToken, error) {
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
