package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	// DefaultTLSAPIURL is the fallback URL when TLS_API_URL env var is not set
	DefaultTLSAPIURL = "http://localhost:8080"

	// DefaultTLSAPITimeout is the default timeout for TLS-API requests
	DefaultTLSAPITimeout = 30 * time.Second

	// TLSAPIRequestPath is the endpoint for making requests
	TLSAPIRequestPath = "/v1/request"
)

// TLSAPIClient handles communication with the TLS-API service
type TLSAPIClient struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewTLSAPIClient creates a new TLS-API client
// Configuration is read from environment variables:
// - TLS_API_URL: Base URL (default: http://localhost:8080)
// - TLS_API_TOKEN: Authorization token (optional)
func NewTLSAPIClient() *TLSAPIClient {
	baseURL := os.Getenv("TLS_API_URL")
	if baseURL == "" {
		baseURL = DefaultTLSAPIURL
	}

	authToken := os.Getenv("TLS_API_TOKEN")

	return &TLSAPIClient{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: DefaultTLSAPITimeout,
		},
	}
}

// NewTLSAPIClientWithConfig creates a TLS-API client with explicit configuration
func NewTLSAPIClientWithConfig(baseURL, authToken string, timeout time.Duration) *TLSAPIClient {
	if baseURL == "" {
		baseURL = DefaultTLSAPIURL
	}
	if timeout == 0 {
		timeout = DefaultTLSAPITimeout
	}

	return &TLSAPIClient{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Request sends an HTTP request through the TLS-API service
func (c *TLSAPIClient) Request(req TLSRequest) (*TLSResponse, error) {
	// Ensure ReturnCookies is set by default
	req.ReturnCookies = true

	// Set default timeout if not specified
	if req.Timeout == 0 {
		req.Timeout = 30 // 30 seconds in ms
	}

	// Marshal request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, NewError(PhaseTLSAPI, "marshal request", err)
	}

	// Create HTTP request
	url := c.baseURL + TLSAPIRequestPath
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, NewError(PhaseTLSAPI, "create http request", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, NewError(PhaseTLSAPI, "execute request", err).WithRetryable(true)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewError(PhaseTLSAPI, "read response body", err)
	}

	// Check for HTTP errors from TLS-API itself
	if resp.StatusCode != http.StatusOK {
		return nil, NewErrorWithStatus(
			PhaseTLSAPI,
			"tls-api returned non-200",
			resp.StatusCode,
			fmt.Errorf("body: %s", string(respBody)),
		)
	}

	// Parse response
	var tlsResp TLSResponse
	if err := json.Unmarshal(respBody, &tlsResp); err != nil {
		return nil, NewError(PhaseTLSAPI, "unmarshal response", err)
	}

	// Check for API-level errors
	if !tlsResp.Success && tlsResp.Error != nil {
		return &tlsResp, c.handleAPIError(tlsResp.Error)
	}

	return &tlsResp, nil
}

// handleAPIError converts a TLS-API error to a SolverError
func (c *TLSAPIClient) handleAPIError(apiErr *TLSAPIError) *SolverError {
	err := &SolverError{
		Phase:    PhaseTLSAPI,
		Step:     apiErr.Type,
		RawError: apiErr.Message,
	}

	// Determine retryability based on error category
	switch apiErr.Category {
	case "TLS":
		err.Retryable = true
	case "PROXY":
		err.Retryable = true
	case "SITE":
		err.Retryable = true
	case "VALIDATION":
		err.Retryable = false
	default:
		err.Retryable = false
	}

	return err
}

// GetStatus returns the status of the request target (not the TLS-API)
func (r *TLSResponse) GetStatus() int {
	if r.Data != nil {
		return r.Data.Status
	}
	return 0
}

// GetBody returns the response body from the target
func (r *TLSResponse) GetBody() string {
	if r.Data != nil {
		return r.Data.Body
	}
	return ""
}

// GetHeaders returns the response headers from the target
func (r *TLSResponse) GetHeaders() map[string]string {
	if r.Data != nil {
		return r.Data.Headers
	}
	return nil
}

// GetCookies returns the cookies from the target response
func (r *TLSResponse) GetCookies() []Cookie {
	if r.Data != nil {
		return r.Data.Cookies
	}
	return nil
}

// IsSuccess checks if the request to the target was successful (2xx status)
func (r *TLSResponse) IsSuccess() bool {
	status := r.GetStatus()
	return status >= 200 && status < 300
}

// Ping checks if the TLS-API service is reachable
func (c *TLSAPIClient) Ping() error {
	// Simple GET to base URL to check connectivity
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return NewError(PhaseTLSAPI, "ping failed", err).WithRetryable(true)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NewErrorWithStatus(PhaseTLSAPI, "health check failed", resp.StatusCode, nil)
	}

	return nil
}

// GetBaseURL returns the configured base URL
func (c *TLSAPIClient) GetBaseURL() string {
	return c.baseURL
}

// HasAuth returns whether authentication is configured
func (c *TLSAPIClient) HasAuth() bool {
	return c.authToken != ""
}
