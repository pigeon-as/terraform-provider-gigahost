package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const DefaultAddress = "https://api.gigahost.no/api/v0"

const (
	defaultTimeout  = 30 * time.Second
	defaultRetryMax = 2
)

var ErrNotFound = errors.New("gigahost: resource not found")

type Config struct {
	Address    string
	Token      string
	HTTPClient *http.Client
	UserAgent  string
}

type Client struct {
	baseURL   *url.URL
	token     string
	http      *retryablehttp.Client
	userAgent string
}

func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = &Config{}
	}

	address := config.Address
	if address == "" {
		address = DefaultAddress
	}

	baseURL, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("parsing Gigahost API address %q: %w", address, err)
	}
	if !strings.HasSuffix(baseURL.Path, "/") {
		baseURL.Path += "/"
	}

	if config.Token == "" {
		return nil, errors.New("a Gigahost API token is required")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	retryClient := &retryablehttp.Client{
		HTTPClient:   httpClient,
		Logger:       nil,
		RetryWaitMin: 100 * time.Millisecond,
		RetryWaitMax: 400 * time.Millisecond,
		RetryMax:     defaultRetryMax,
		CheckRetry:   retryPolicy,
		Backoff:      retryablehttp.DefaultBackoff,
		ErrorHandler: retryablehttp.PassthroughErrorHandler,
	}

	return &Client{
		baseURL:   baseURL,
		token:     config.Token,
		http:      retryClient,
		userAgent: config.UserAgent,
	}, nil
}

func retryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if err != nil {
		return false, err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}
	if resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented &&
		resp.Request != nil && isIdempotentMethod(resp.Request.Method) {
		return true, nil
	}
	return false, nil
}

func isIdempotentMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

func (c *Client) newRequest(ctx context.Context, method, apiPath string, query url.Values, body any) (*retryablehttp.Request, error) {
	ref := &url.URL{Path: apiPath}
	if len(query) > 0 {
		ref.RawQuery = query.Encode()
	}
	endpoint := c.baseURL.ResolveReference(ref)

	var rawBody any
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
		rawBody = encoded
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, method, endpoint.String(), rawBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	return req, nil
}

func (c *Client) sendRequest(req *retryablehttp.Request, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("performing %s %s: %w", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if err := checkResponse(resp.StatusCode, body); err != nil {
		return err
	}

	if out == nil || len(body) == 0 {
		return nil
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("decoding response envelope: %w", err)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("decoding response data: %w", err)
	}

	return nil
}

func checkResponse(statusCode int, body []byte) error {
	if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
		return nil
	}
	message := errorMessage(body)
	if message == "" {
		message = http.StatusText(statusCode)
	}
	return &Error{StatusCode: statusCode, Message: message}
}

func errorMessage(body []byte) string {
	var env envelope
	if err := json.Unmarshal(body, &env); err == nil {
		if env.Meta.Message != "" {
			return env.Meta.Message
		}
		if env.Meta.StatusMessage != "" {
			return env.Meta.StatusMessage
		}
	}
	return ""
}

type meta struct {
	Status        int    `json:"status"`
	StatusMessage string `json:"status_message"`
	Message       string `json:"message"`
}

type envelope struct {
	Meta meta            `json:"meta"`
	Data json.RawMessage `json:"data"`
}

type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("gigahost: HTTP %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("gigahost: HTTP %d", e.StatusCode)
}

func (e *Error) Is(target error) bool {
	return target == ErrNotFound && e.StatusCode == http.StatusNotFound
}
