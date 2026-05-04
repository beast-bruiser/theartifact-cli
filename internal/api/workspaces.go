package api

import "fmt"

type Workspace struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	StyleSummary string `json:"style_summary"`
}

func (c *Client) ListWorkspaces() ([]Workspace, error) {
	var workspaces []Workspace
	resp, err := c.HTTPClient.R().
		SetResult(&workspaces).
		Get("/v1/workspaces")

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

	return workspaces, nil
}
