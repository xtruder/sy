package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the SiYuan API client
type Client struct {
	host       string
	token      string
	httpClient *http.Client
}

// APIResponse is the standard SiYuan API response wrapper
type APIResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// NewClient creates a new SiYuan API client
func NewClient(host, token string) (*Client, error) {
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}

	return &Client{
		host:  host,
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Post makes a POST request to the SiYuan API
func (c *Client) Post(endpoint string, payload interface{}) (*APIResponse, error) {
	url := c.host + endpoint

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Token "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API error: %s", apiResp.Msg)
	}

	return &apiResp, nil
}

// PostData makes a POST request and unmarshals the data field into the result
func (c *Client) PostData(endpoint string, payload interface{}, result interface{}) error {
	resp, err := c.Post(endpoint, payload)
	if err != nil {
		return err
	}

	if result != nil && len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return nil
}
