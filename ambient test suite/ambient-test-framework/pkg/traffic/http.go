// pkg/traffic/http.go
package traffic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Endpoint identifies a service endpoint to send traffic to.
type Endpoint struct {
	Host    string
	Port    int
	Path    string
	Scheme  string // "http" or "https"; defaults to "http"
	Headers map[string]string
}

// URL returns the full URL for the endpoint.
func (e Endpoint) URL() string {
	scheme := e.Scheme
	if scheme == "" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s:%d%s", scheme, e.Host, e.Port, e.Path)
}

// HTTPResponse wraps an http.Response with a pre-read body.
type HTTPResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// HTTPGet sends a GET request and returns the response.
func HTTPGet(ctx context.Context, client *http.Client,
	endpoint Endpoint) (*HTTPResponse, error) {

	if client == nil {
		client = defaultHTTPClient()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		endpoint.URL(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	for k, v := range endpoint.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
	}, nil
}

// HTTPGetJSON sends a GET request and JSON-decodes the response body.
func HTTPGetJSON(ctx context.Context, client *http.Client,
	endpoint Endpoint) (map[string]interface{}, error) {

	resp, err := HTTPGet(ctx, client, endpoint)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	return result, nil
}

// HTTPSend sends N requests to an endpoint and returns all responses.
func HTTPSend(ctx context.Context, client *http.Client,
	endpoint Endpoint, count int) ([]*HTTPResponse, error) {

	if client == nil {
		client = defaultHTTPClient()
	}

	responses := make([]*HTTPResponse, 0, count)
	for i := 0; i < count; i++ {
		resp, err := HTTPGet(ctx, client, endpoint)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}
	return responses, nil
}

func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}
