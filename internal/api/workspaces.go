package api

import "fmt"

type Workspace struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	StyleSummary string `json:"style_summary"`
}

type createWorkspaceRequest struct {
	Name string `json:"name"`
}

func (c *Client) CreateWorkspace(name string) (*Workspace, error) {
	var ws Workspace
	resp, err := c.HTTPClient.R().
		SetBody(createWorkspaceRequest{Name: name}).
		SetResult(&ws).
		Post("/v1/workspaces")

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.IsError() {
		if resp.StatusCode() == 401 || resp.StatusCode() == 403 {
			return nil, fmt.Errorf("workspace creation not yet available via API key — create at https://theartifact.art/workspaces/new")
		}
		apiErr, ok := resp.Error().(*APIError)
		if ok && apiErr.Message != "" {
			return nil, fmt.Errorf("API error (%s): %s", apiErr.Code, apiErr.Message)
		}
		return nil, fmt.Errorf("API request failed with status: %s", resp.Status())
	}

	return &ws, nil
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
