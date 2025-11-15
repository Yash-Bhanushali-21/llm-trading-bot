package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"llm-trading-bot/internal/logger"
	"net/http"
	"time"
)

// Client represents an HTTP client with common configuration and utilities
type Client struct {
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
	useLogging bool
}

// logDebug logs debug messages using the global logger
func (c *Client) logDebug(ctx context.Context, msg string, args ...interface{}) {
	if c.useLogging {
		logger.Debug(ctx, msg, args...)
	}
}

// logInfo logs info messages using the global logger
func (c *Client) logInfo(ctx context.Context, msg string, args ...interface{}) {
	if c.useLogging {
		logger.Info(ctx, msg, args...)
	}
}

// logWarn logs warning messages using the global logger
func (c *Client) logWarn(ctx context.Context, msg string, args ...interface{}) {
	if c.useLogging {
		logger.Warn(ctx, msg, args...)
	}
}

// logError logs error messages using the global logger
func (c *Client) logError(ctx context.Context, msg string, args ...interface{}) {
	if c.useLogging {
		logger.Error(ctx, msg, args...)
	}
}

// ClientOption configures the API client
type ClientOption func(*Client)

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithBaseURL sets the base URL for all requests
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithHeader sets a default header for all requests
func WithHeader(key, value string) ClientOption {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithLogging enables logging for the API client
func WithLogging(enabled bool) ClientOption {
	return func(c *Client) {
		c.useLogging = enabled
	}
}

// NewClient creates a new API client with the given options
func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers:    make(map[string]string),
		useLogging: false, // Default: logging disabled for performance
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Request represents an HTTP request configuration
type Request struct {
	Method  string
	URL     string
	Body    interface{}
	Headers map[string]string
	ctx     context.Context
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// NewRequest creates a new request
func NewRequest(method, url string) *Request {
	return &Request{
		Method:  method,
		URL:     url,
		Headers: make(map[string]string),
		ctx:     context.Background(),
	}
}

// WithContext sets the context for the request
func (r *Request) WithContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// WithBody sets the request body (will be JSON encoded)
func (r *Request) WithBody(body interface{}) *Request {
	r.Body = body
	return r
}

// WithHeader sets a request-specific header
func (r *Request) WithHeader(key, value string) *Request {
	r.Headers[key] = value
	return r
}

// Do executes the HTTP request
func (c *Client) Do(req *Request) (*Response, error) {
	// Build full URL
	url := req.URL
	if c.baseURL != "" {
		url = c.baseURL + req.URL
	}

	// Encode body if present
	var bodyReader io.Reader
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			c.logError(req.ctx, "Failed to marshal request body", "error", err)
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(req.ctx, req.Method, url, bodyReader)
	if err != nil {
		c.logError(req.ctx, "Failed to create HTTP request", "error", err)
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set default headers
	for key, value := range c.headers {
		httpReq.Header.Set(key, value)
	}

	// Set request-specific headers (override defaults)
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// If body is JSON, ensure Content-Type is set
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Log request
	c.logDebug(req.ctx, "HTTP Request", "method", req.Method, "url", url)

	// Execute request
	startTime := time.Now()
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logError(req.ctx, "HTTP request failed", "method", req.Method, "url", url, "error", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		c.logError(req.ctx, "Failed to read response body", "error", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response
	duration := time.Since(startTime)
	c.logDebug(req.ctx, "HTTP Response",
		"method", req.Method,
		"url", url,
		"status", httpResp.StatusCode,
		"duration", duration,
		"bodySize", len(body))

	// Check for error status codes
	if httpResp.StatusCode >= 400 {
		c.logWarn(req.ctx, "HTTP error response",
			"method", req.Method,
			"url", url,
			"status", httpResp.StatusCode,
			"body", string(body))
		return nil, fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(body))
	}

	return &Response{
		StatusCode: httpResp.StatusCode,
		Body:       body,
		Headers:    httpResp.Header,
	}, nil
}

// GET performs a GET request
func (c *Client) GET(ctx context.Context, url string, headers ...map[string]string) (*Response, error) {
	req := NewRequest(http.MethodGet, url).WithContext(ctx)

	// Add headers if provided
	if len(headers) > 0 {
		for key, value := range headers[0] {
			req.WithHeader(key, value)
		}
	}

	return c.Do(req)
}

// POST performs a POST request
func (c *Client) POST(ctx context.Context, url string, body interface{}, headers ...map[string]string) (*Response, error) {
	req := NewRequest(http.MethodPost, url).
		WithContext(ctx).
		WithBody(body)

	// Add headers if provided
	if len(headers) > 0 {
		for key, value := range headers[0] {
			req.WithHeader(key, value)
		}
	}

	return c.Do(req)
}

// PUT performs a PUT request
func (c *Client) PUT(ctx context.Context, url string, body interface{}, headers ...map[string]string) (*Response, error) {
	req := NewRequest(http.MethodPut, url).
		WithContext(ctx).
		WithBody(body)

	// Add headers if provided
	if len(headers) > 0 {
		for key, value := range headers[0] {
			req.WithHeader(key, value)
		}
	}

	return c.Do(req)
}

// PATCH performs a PATCH request
func (c *Client) PATCH(ctx context.Context, url string, body interface{}, headers ...map[string]string) (*Response, error) {
	req := NewRequest(http.MethodPatch, url).
		WithContext(ctx).
		WithBody(body)

	// Add headers if provided
	if len(headers) > 0 {
		for key, value := range headers[0] {
			req.WithHeader(key, value)
		}
	}

	return c.Do(req)
}

// DELETE performs a DELETE request
func (c *Client) DELETE(ctx context.Context, url string, headers ...map[string]string) (*Response, error) {
	req := NewRequest(http.MethodDelete, url).WithContext(ctx)

	// Add headers if provided
	if len(headers) > 0 {
		for key, value := range headers[0] {
			req.WithHeader(key, value)
		}
	}

	return c.Do(req)
}

// ParseJSON parses the response body as JSON into the given struct
func (r *Response) ParseJSON(v interface{}) error {
	if err := json.Unmarshal(r.Body, v); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w", err)
	}
	return nil
}

// String returns the response body as a string
func (r *Response) String() string {
	return string(r.Body)
}

// Common header presets for different APIs

// BrowserHeaders returns common browser headers to mimic a real browser request
func BrowserHeaders() map[string]string {
	return map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":          "application/json, text/plain, */*",
		"Accept-Language": "en-US,en;q=0.9",
	}
}

// YahooFinanceHeaders returns headers for Yahoo Finance API
func YahooFinanceHeaders() map[string]string {
	return map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":          "application/json",
		"Accept-Language": "en-US,en;q=0.9",
		"Referer":         "https://finance.yahoo.com/",
	}
}

// NSEHeaders returns headers for NSE India API
func NSEHeaders() map[string]string {
	return map[string]string{
		"User-Agent":       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Accept":           "application/json",
		"Accept-Language":  "en-US,en;q=0.9",
		"Referer":          "https://www.nseindia.com/",
		"X-Requested-With": "XMLHttpRequest",
	}
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 3,
		InitialWait: 1 * time.Second,
		MaxWait:     5 * time.Second,
	}
}

// DoWithRetry executes a request with retry logic
func (c *Client) DoWithRetry(req *Request, config *RetryConfig) (*Response, error) {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	waitTime := config.InitialWait

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		c.logDebug(req.ctx, "Request attempt", "attempt", attempt, "maxAttempts", config.MaxAttempts)

		resp, err := c.Do(req)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		c.logWarn(req.ctx, "Request failed, retrying", "attempt", attempt, "error", err, "waitTime", waitTime)

		// Don't wait after the last attempt
		if attempt < config.MaxAttempts {
			time.Sleep(waitTime)
			// Exponential backoff
			waitTime = waitTime * 2
			if waitTime > config.MaxWait {
				waitTime = config.MaxWait
			}
		}
	}

	c.logError(req.ctx, "All retry attempts failed", "maxAttempts", config.MaxAttempts, "error", lastErr)
	return nil, fmt.Errorf("all %d retry attempts failed: %w", config.MaxAttempts, lastErr)
}
