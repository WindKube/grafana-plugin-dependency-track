package dtrack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	DefaultPageSize   = 100
	DefaultMaxRetries = 3
	// DefaultTimeout stays above the 60s cap Dependency-Track puts on portfolio finding queries.
	DefaultTimeout       = 90 * time.Second
	DefaultMaxConcurrent = 10
	DefaultCacheMaxBytes = 64 << 20
	maxRetryWait         = 30 * time.Second
	maxPages             = 10000
	maxErrorBodyBytes    = 4096
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
	// MaxConcurrent caps requests in flight to Dependency-Track across all queries.
	MaxConcurrent int
	// CacheTTL enables the response cache; zero disables it.
	CacheTTL      time.Duration
	CacheMaxBytes int
}

type Client struct {
	baseURL    *url.URL
	apiKey     string
	http       *http.Client
	pageSize   int
	maxRetries int
	retryWait  time.Duration
	userAgent  string
	sem        chan struct{}
	cache      *responseCache
	flight     singleflight.Group
}

// ErrUnexpectedResponse marks a 2xx response whose body is not the expected JSON, e.g. an HTML page.
var ErrUnexpectedResponse = errors.New("unexpected response body")

type noRetriesKey struct{}

// WithoutRetries makes requests made with ctx fail on the first error, e.g. for health checks.
func WithoutRetries(ctx context.Context) context.Context {
	return context.WithValue(ctx, noRetriesKey{}, true)
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
	if opts.MaxConcurrent <= 0 {
		opts.MaxConcurrent = DefaultMaxConcurrent
	}
	c.sem = make(chan struct{}, opts.MaxConcurrent)
	if opts.CacheTTL > 0 {
		if opts.CacheMaxBytes <= 0 {
			opts.CacheMaxBytes = DefaultCacheMaxBytes
		}
		c.cache = newResponseCache(opts.CacheTTL, opts.CacheMaxBytes)
	}
	return c, nil
}

func (c *Client) BaseURL() string {
	return c.baseURL.String()
}

// get performs a GET against a path relative to /api and decodes the JSON body into out.
// Identical concurrent requests share one round trip, and successful responses are cached
// when a TTL is configured.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) (http.Header, error) {
	key := path + "?" + query.Encode()
	useCache := c.cache != nil && !cacheDisabled(ctx)

	var res *cachedResponse
	if useCache {
		if hit, ok := c.cache.get(key); ok {
			cacheRequests.WithLabelValues("hit").Inc()
			res = hit
		}
	}
	if res == nil {
		ch := c.flight.DoChan(key, func() (any, error) {
			// Shared by every waiter, so one cancelled caller must not fail the others.
			// The HTTP client timeout still bounds the request.
			body, header, err := c.fetch(context.WithoutCancel(ctx), path, query)
			if err != nil {
				return nil, err
			}
			if useCache {
				c.cache.set(key, body, header)
			}
			return &cachedResponse{body: body, header: header}, nil
		})
		var r singleflight.Result
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r = <-ch:
		}
		if useCache {
			cacheRequests.WithLabelValues(map[bool]string{true: "shared", false: "miss"}[r.Shared]).Inc()
		}
		if r.Err != nil {
			return nil, r.Err
		}
		res = r.Val.(*cachedResponse)
	}

	if out != nil {
		if err := json.Unmarshal(res.body, out); err != nil {
			return nil, fmt.Errorf("dependency-track: decoding %s: %w: %w", path, ErrUnexpectedResponse, err)
		}
	}
	return res.header, nil
}

func (c *Client) fetch(ctx context.Context, path string, query url.Values) ([]byte, http.Header, error) {
	start := time.Now()
	status := "error"
	defer func() {
		requestDuration.WithLabelValues(endpoint(path), status).Observe(time.Since(start).Seconds())
	}()

	resp, err := c.do(ctx, path, query)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			status = strconv.Itoa(apiErr.StatusCode)
		}
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	status = strconv.Itoa(resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("dependency-track: reading %s: %w", path, err)
	}
	return body, resp.Header, nil
}

func (c *Client) roundTrip(req *http.Request) (*http.Response, error) {
	select {
	case c.sem <- struct{}{}:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
	inFlight.Inc()
	defer func() {
		inFlight.Dec()
		<-c.sem
	}()
	return c.http.Do(req)
}

func (c *Client) do(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	// path is already escaped by the callers; keep both forms so url.URL doesn't escape it again.
	u := *c.baseURL
	u.RawPath = c.baseURL.EscapedPath() + "/api" + path
	unescaped, err := url.PathUnescape(u.RawPath)
	if err != nil {
		return nil, fmt.Errorf("dependency-track: invalid path %q: %w", path, err)
	}
	u.Path = unescaped
	u.RawQuery = query.Encode()

	var lastErr error
	maxRetries := c.maxRetries
	if v, _ := ctx.Value(noRetriesKey{}).(bool); v {
		maxRetries = 0
	}
	for attempt := 0; attempt <= maxRetries; attempt++ {
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

		resp, err := c.roundTrip(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("dependency-track: GET %s: %w", path, err)
			// A timed-out query is likely to time out again and only adds load on the server.
			if isTimeout(err) {
				return nil, lastErr
			}
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

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout())
}

func (c *Client) backoff(attempt int) time.Duration {
	d := min(c.retryWait<<min(attempt-1, 10), maxRetryWait)
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
