package dtrack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultPageSize   = 100
	DefaultMaxRetries = 3
	DefaultTimeout    = 30 * time.Second
	maxPages          = 10000
	maxErrorBodyBytes = 4096
)

type Options struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	PageSize   int
	// MaxRetries defaults to DefaultMaxRetries; a negative value disables retries.
	MaxRetries int
	RetryWait  time.Duration
	UserAgent  string
}

type Client struct {
	baseURL    *url.URL
	apiKey     string
	http       *http.Client
	pageSize   int
	maxRetries int
	retryWait  time.Duration
	userAgent  string
}

type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Body       string
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("dependency-track: %s %s: %d %s", e.Method, e.Path, e.StatusCode, http.StatusText(e.StatusCode))
	if e.Body != "" {
		msg += ": " + e.Body
	}
	return msg
}

func IsStatus(err error, code int) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == code
}

func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, errors.New("dependency-track: base URL is required")
	}
	u, err := url.Parse(strings.TrimRight(opts.BaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("dependency-track: invalid base URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("dependency-track: base URL must use http or https, got %q", opts.BaseURL)
	}
	u.Path = strings.TrimSuffix(u.Path, "/api")

	c := &Client{
		baseURL:    u,
		apiKey:     opts.APIKey,
		http:       opts.HTTPClient,
		pageSize:   opts.PageSize,
		maxRetries: opts.MaxRetries,
		retryWait:  opts.RetryWait,
		userAgent:  opts.UserAgent,
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: DefaultTimeout}
	}
	if c.pageSize <= 0 {
		c.pageSize = DefaultPageSize
	}
	switch {
	case c.maxRetries == 0:
		c.maxRetries = DefaultMaxRetries
	case c.maxRetries < 0:
		c.maxRetries = 0
	}
	if c.retryWait <= 0 {
		c.retryWait = 500 * time.Millisecond
	}
	return c, nil
}

func (c *Client) BaseURL() string {
	return c.baseURL.String()
}

// get performs a GET against a path relative to /api and decodes the JSON body into out.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) (http.Header, error) {
	resp, err := c.do(ctx, path, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return nil, fmt.Errorf("dependency-track: decoding %s: %w", path, err)
		}
	}
	return resp.Header, nil
}

func (c *Client) do(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	u := *c.baseURL
	u.Path = u.Path + "/api" + path
	u.RawQuery = query.Encode()

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleep(ctx, c.backoff(attempt)); err != nil {
				return nil, err
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		if c.apiKey != "" {
			req.Header.Set("X-Api-Key", c.apiKey)
		}
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("dependency-track: GET %s: %w", path, err)
			continue
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		_ = resp.Body.Close()
		lastErr = &APIError{StatusCode: resp.StatusCode, Method: http.MethodGet, Path: path, Body: strings.TrimSpace(string(body))}
		if !retryable(resp.StatusCode) {
			return nil, lastErr
		}
	}
	return nil, lastErr
}

func retryable(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func (c *Client) backoff(attempt int) time.Duration {
	d := c.retryWait << (attempt - 1)
	return d + rand.N(d/2+1)
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// getAll pages through a list endpoint until X-Total-Count items have been read.
// Dependency-Track 5 enforces pagination, so a single request silently truncates results.
func getAll[T any](ctx context.Context, c *Client, path string, query url.Values) ([]T, error) {
	q := url.Values{}
	for k, v := range query {
		q[k] = v
	}
	q.Set("pageSize", strconv.Itoa(c.pageSize))

	var all []T
	for page := 1; page <= maxPages; page++ {
		q.Set("pageNumber", strconv.Itoa(page))
		var items []T
		header, err := c.get(ctx, path, q, &items)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)

		total, hasTotal := totalCount(header)
		if len(items) == 0 || (hasTotal && len(all) >= total) || (!hasTotal && len(items) < c.pageSize) {
			return all, nil
		}
	}
	return nil, fmt.Errorf("dependency-track: %s: more than %d pages", path, maxPages)
}

// totalCount only trusts exact totals; v5 may report a lower bound for expensive queries.
func totalCount(h http.Header) (int, bool) {
	if t := h.Get("X-Total-Count-Type"); t != "" && !strings.EqualFold(t, "EXACT") {
		return 0, false
	}
	v := h.Get("X-Total-Count")
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}
