package client

import (
	"bytes"
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
	baseURL    *url.URL
	token      string
	httpClient *http.Client
	userAgent  string
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

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{
		baseURL:    baseURL,
		token:      config.Token,
		httpClient: httpClient,
		userAgent:  config.UserAgent,
	}, nil
}

func (c *Client) newRequest(ctx context.Context, method, apiPath string, body any) (*http.Request, error) {
	endpoint := *c.baseURL
	endpoint.Path = path.Join(endpoint.Path, apiPath)

	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bodyReader)
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

func (c *Client) sendRequest(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
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
