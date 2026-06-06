package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const DefaultAddress = "https://api.gigahost.no/api/v0"

const defaultTimeout = 30 * time.Second

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

	if config.Token == "" {
		return nil, errors.New("a Gigahost API token is required")
	}

	retryClient := retryablehttp.NewClient()
	retryClient.Logger = nil
	if config.HTTPClient != nil {
		retryClient.HTTPClient = config.HTTPClient
	} else {
		retryClient.HTTPClient = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{
		baseURL:   baseURL,
		token:     config.Token,
		http:      retryClient,
		userAgent: config.UserAgent,
	}, nil
}

func (c *Client) newRequest(ctx context.Context, method, apiPath string, body any) (*retryablehttp.Request, error) {
	endpoint := *c.baseURL
	reqPath, rawQuery, _ := strings.Cut(apiPath, "?")
	endpoint.Path = path.Join(endpoint.Path, reqPath)
	endpoint.RawQuery = rawQuery

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

	var env envelope
	_ = json.Unmarshal(body, &env)

	message := env.Meta.Message
	if message == "" {
		if raw := strings.TrimSpace(string(body)); raw != "" {
			message = raw
		} else {
			message = env.Meta.StatusMessage
		}
	}

	return &Error{StatusCode: statusCode, Message: message}
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
