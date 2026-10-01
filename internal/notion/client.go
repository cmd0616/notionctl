// Package notion provides a thin HTTP client for the Notion API.
package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL       = "https://api.notion.com/v1"
	apiVersion    = "2022-06-28"
	defaultTimeout = 30 * time.Second
)

// Client is a Notion API client.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Notion API client with the given integration token.
func NewClient(token string) *Client {
	return &Client{
		token:   token,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// NewClientWithBase creates a client with a custom base URL (for testing).
func NewClientWithBase(base, token string) *Client {
	return &Client{
		token:   token,
		baseURL: base,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// CreateDatabase creates a new database in Notion.
func (c *Client) CreateDatabase(parentPageID, title string, properties map[string]interface{}) (string, error) {
	body := map[string]interface{}{
		"parent": map[string]interface{}{
			"type":    "page_id",
			"page_id": parentPageID,
		},
		"title": []map[string]interface{}{
			{
				"type": "text",
				"text": map[string]interface{}{
					"content": title,
				},
			},
		},
		"properties": properties,
	}

	resp, err := c.do("POST", "/databases", body)
	if err != nil {
		return "", fmt.Errorf("creating database %q: %w", title, err)
	}

	id, ok := resp["id"].(string)
	if !ok {
		return "", fmt.Errorf("creating database %q: missing id in response", title)
	}
	return id, nil
}

// UpdateDatabase updates an existing database's properties.
func (c *Client) UpdateDatabase(databaseID string, properties map[string]interface{}) error {
	body := map[string]interface{}{
		"properties": properties,
	}

	_, err := c.do("PATCH", "/databases/"+databaseID, body)
	if err != nil {
		return fmt.Errorf("updating database %s: %w", databaseID, err)
	}
	return nil
}

// GetDatabase retrieves a database by ID.
func (c *Client) GetDatabase(databaseID string) (map[string]interface{}, error) {
	resp, err := c.do("GET", "/databases/"+databaseID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting database %s: %w", databaseID, err)
	}
	return resp, nil
}

func (c *Client) do(method, path string, body interface{}) (map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", apiVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respData))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return result, nil
}
