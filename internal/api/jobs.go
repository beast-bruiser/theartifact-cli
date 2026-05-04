package api

import "fmt"

type Job struct {
	ID       string     `json:"id"`
	Status   string     `json:"status"` // queued, running, succeeded, failed
	Kind     string     `json:"kind"`
	Progress int        `json:"progress"`
	Result   *JobResult `json:"result,omitempty"`
	Error    string     `json:"error,omitempty"`
}

type JobResult struct {
	Manifest struct {
		Prompt    string `json:"prompt"`
		Model     string `json:"model"`
		Seed      int64  `json:"seed,omitempty"`
		AssetID   string `json:"asset_id,omitempty"`
		CreatedAt string `json:"created_at,omitempty"`
	} `json:"manifest"`
	Assets []struct {
		ID        string `json:"id"`
		URL       string `json:"url"`
		ExpiresAt string `json:"expires_at,omitempty"`
		Kind      string `json:"kind,omitempty"`
	} `json:"assets"`
	GraphFragment *GraphFragment `json:"graph_fragment,omitempty"`
	BrainFiles    []BrainFile    `json:"brain_files,omitempty"`
}

type GraphFragment struct {
	Nodes   []GraphNode `json:"nodes"`
	Edges   []GraphEdge `json:"edges"`
	Version int         `json:"version"`
}

type GraphNode struct {
	ID    string            `json:"id"`
	Kind  string            `json:"kind"`
	Label string            `json:"label"`
	Attrs map[string]string `json:"attrs,omitempty"`
}

type GraphEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
}

type BrainFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	SHA     string `json:"sha"`
}

func (c *Client) GetJob(id string) (*Job, error) {
	var job Job
	resp, err := c.HTTPClient.R().
		SetResult(&job).
		Get(fmt.Sprintf("/v1/jobs/%s", id))

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

	return &job, nil
}
