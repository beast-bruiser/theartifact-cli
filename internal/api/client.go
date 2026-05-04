package api

import (
	"fmt"
	"os"

	"github.com/go-resty/resty/v2"
	"theartifact-cli/internal/config"
)

const DefaultBaseURL = "https://api.theartifact.art"

type Client struct {
	HTTPClient *resty.Client
}

// APIError represents the error structure returned by TheArtifact API.
// Spec envelope: { "code": "...", "message": "..." }
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// resolveBaseURL returns the API base URL from: env var > config > default.
func resolveBaseURL(cfg *config.Config) string {
	if envURL := os.Getenv("ARTIFACT_API_URL"); envURL != "" {
		return envURL
	}
	if cfg.BaseURL != "" {
		return cfg.BaseURL
	}
	return DefaultBaseURL
}

func NewClient() (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is not set. Please run 'theartifact login --key <your-key>'")
	}

	baseURL := resolveBaseURL(cfg)

	httpClient := resty.New()
	httpClient.SetBaseURL(baseURL)
	httpClient.SetAuthToken(cfg.APIKey)
	httpClient.SetHeader("Content-Type", "application/json")

	// Set a common error struct for resty to automatically unmarshal into on error (400-599 HTTP status codes)
	httpClient.SetError(&APIError{})

	return &Client{
		HTTPClient: httpClient,
	}, nil
}
