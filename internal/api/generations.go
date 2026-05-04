package api

import "fmt"

type GenerateRequest struct {
	WorkspaceID    string   `json:"workspace_id"`
	Prompt         string   `json:"prompt"`
	Count          int      `json:"count,omitempty"`          // 1–4, default 1
	Seed           int64    `json:"seed,omitempty"`           // for reproducibility
	Refs           []string `json:"refs,omitempty"`           // upload_tokens for reference images
	WebhookURL     string   `json:"webhook_url,omitempty"`    // HTTPS callback on terminal state
	IdempotencyKey string   `json:"-"`                        // passed as header, not body
}

type GenerateResponse struct {
	JobID string `json:"job_id"`
}

func (c *Client) Generate(req GenerateRequest) (string, error) {
	var genResp GenerateResponse
	r := c.HTTPClient.R().SetBody(req).SetResult(&genResp)
	if req.IdempotencyKey != "" {
		r = r.SetHeader("Idempotency-Key", req.IdempotencyKey)
	}
	resp, err := r.Post("/v1/generations")

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

	return genResp.JobID, nil
}
