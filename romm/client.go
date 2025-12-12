package romm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sonh/qs"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
}

type QueryParam interface {
	Valid() bool
}

type ClientOption func(*Client)

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

func WithBasicAuth(username, password string) ClientOption {
	return func(c *Client) {
		c.username = username
		c.password = password
	}
}

func NewClient(baseURL string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) doRequest(method string, path string, queryParams QueryParam, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	u := c.baseURL + path

	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if queryParams != nil && queryParams.Valid() {
		values, err := qs.NewEncoder().Values(queryParams)
		if err == nil {
			req.URL.RawQuery = values.Encode()
		}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) doRequestRaw(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	u := c.baseURL + path
	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return bodyBytes, nil
}

func (c *Client) doMultipartRequest(method, path string, queryParams QueryParam, body io.Reader, contentType string, result interface{}) error {
	u := c.baseURL + path
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	if queryParams != nil && queryParams.Valid() {
		values, err := qs.NewEncoder().Values(queryParams)
		if err == nil {
			req.URL.RawQuery = values.Encode()
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func buildQueryString(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	values := url.Values{}
	for k, v := range params {
		if v != "" {
			values.Add(k, v)
		}
	}

	query := values.Encode()
	if query != "" {
		return "?" + query
	}
	return ""
}

func (c *Client) DownloadFile(downloadURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}
