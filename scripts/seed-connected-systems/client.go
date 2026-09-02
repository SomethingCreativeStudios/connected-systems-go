package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxErrorBodyBytes = 64 << 10

type APIClient struct {
	baseURL string
	headers map[string]string
	http    *http.Client
}

type APIResponse struct {
	StatusCode int
	Location   string
	Body       []byte
}

func NewAPIClient(cfg Config) (*APIClient, error) {
	base, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	return &APIClient{
		baseURL: strings.TrimRight(base.String(), "/"),
		headers: cfg.HTTP.Headers,
		http:    &http.Client{Timeout: cfg.HTTP.Timeout.Std()},
	}, nil
}

func (c *APIClient) URL(resourcePath string) string {
	if strings.HasPrefix(resourcePath, "http://") || strings.HasPrefix(resourcePath, "https://") {
		return resourcePath
	}
	return c.baseURL + "/" + strings.TrimLeft(resourcePath, "/")
}

func (c *APIClient) Preflight(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodGet, "/conformance", "", "application/json", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return c.statusError(http.MethodGet, "/conformance", resp)
	}
	return nil
}

func (c *APIClient) GetJSON(ctx context.Context, resourcePath string, output any) error {
	resp, err := c.do(ctx, http.MethodGet, resourcePath, "", "application/json", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return c.statusError(http.MethodGet, resourcePath, resp)
	}
	if err := json.Unmarshal(resp.Body, output); err != nil {
		return fmt.Errorf("decode GET %s response: %w", resourcePath, err)
	}
	return nil
}

func (c *APIClient) PostJSON(ctx context.Context, resourcePath string, payload any) (APIResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return APIResponse{}, fmt.Errorf("marshal POST %s payload: %w", resourcePath, err)
	}
	return c.do(ctx, http.MethodPost, resourcePath, "application/json", "application/json", body)
}

func (c *APIClient) Post(ctx context.Context, resourcePath, contentType, accept string, payload any) (APIResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return APIResponse{}, fmt.Errorf("marshal POST %s payload: %w", resourcePath, err)
	}
	return c.do(ctx, http.MethodPost, resourcePath, contentType, accept, body)
}

func (c *APIClient) Put(ctx context.Context, resourcePath, contentType string, payload any) (APIResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return APIResponse{}, fmt.Errorf("marshal PUT %s payload: %w", resourcePath, err)
	}
	return c.do(ctx, http.MethodPut, resourcePath, contentType, contentType, body)
}

func (c *APIClient) do(ctx context.Context, method, resourcePath, contentType, accept string, body []byte) (APIResponse, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.URL(resourcePath), bytes.NewReader(body))
	if err != nil {
		return APIResponse{}, err
	}
	for name, value := range c.headers {
		req.Header.Set(name, value)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return APIResponse{}, fmt.Errorf("%s %s: %w", method, c.URL(resourcePath), err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	if err != nil {
		return APIResponse{}, fmt.Errorf("read %s %s response: %w", method, resourcePath, err)
	}
	return APIResponse{StatusCode: resp.StatusCode, Location: resp.Header.Get("Location"), Body: responseBody}, nil
}

func (c *APIClient) statusError(method, resourcePath string, response APIResponse) error {
	body := strings.TrimSpace(string(response.Body))
	if body == "" {
		body = "<empty response>"
	}
	return fmt.Errorf("%s %s returned HTTP %d: %s", method, resourcePath, response.StatusCode, body)
}

func expectCreated(c *APIClient, resourcePath, resourceName string, response APIResponse) (string, error) {
	if response.StatusCode != http.StatusCreated {
		return "", c.statusError(http.MethodPost, resourcePath, response)
	}
	if response.Location == "" {
		return "", fmt.Errorf("POST %s created %s without a Location header", resourcePath, resourceName)
	}
	id, err := resourceID(response.Location, resourceName)
	if err != nil {
		return "", fmt.Errorf("POST %s returned invalid Location %q: %w", resourcePath, response.Location, err)
	}
	return id, nil
}

func resourceID(location, resourceName string) (string, error) {
	u, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := len(parts) - 2; i >= 0; i-- {
		if parts[i] == resourceName && parts[i+1] != "" {
			return parts[i+1], nil
		}
	}
	return "", fmt.Errorf("no /%s/{id} path segment", resourceName)
}

func collectionID(response APIResponse) (string, error) {
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Body, &result); err != nil {
		return "", err
	}
	if result.ID == "" {
		return "", fmt.Errorf("collection create response has no id")
	}
	return result.ID, nil
}
