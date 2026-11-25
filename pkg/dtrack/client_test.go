package dtrack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.Handler, opts ...func(*Options)) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	o := Options{BaseURL: srv.URL, APIKey: "odt_test", RetryWait: time.Millisecond}
	for _, f := range opts {
		f(&o)
	}
	c, err := New(o)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type item struct {
	ID int `json:"id"`
}

func pagedHandler(t *testing.T, total int, totalType string, requests *atomic.Int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if got := r.Header.Get("X-Api-Key"); got != "odt_test" {
			t.Errorf("X-Api-Key = %q", got)
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("pageNumber"))
		size, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		var items []item
		for i := (page - 1) * size; i < page*size && i < total; i++ {
			items = append(items, item{ID: i})
		}
		if items == nil {
			items = []item{}
		}
		if totalType != "none" {
			reported := total
			if totalType == "BOUNDED" {
				reported = total / 2
			}
			w.Header().Set("X-Total-Count", strconv.Itoa(reported))
			w.Header().Set("X-Total-Count-Type", totalType)
		}
		_ = json.NewEncoder(w).Encode(items)
	}
}

func TestGetAllPaginates(t *testing.T) {
	tests := []struct {
		name      string
		total     int
		totalType string
		requests  int32
	}{
		{"exact total", 250, "EXACT", 3},
		{"exact total on page boundary", 200, "EXACT", 2},
		{"empty", 0, "EXACT", 1},
		{"lower bound total keeps paging", 250, "BOUNDED", 3},
		{"no total header", 250, "none", 3},
		{"no total header on page boundary", 200, "none", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			c := newTestClient(t, pagedHandler(t, tt.total, tt.totalType, &requests))
			items, err := getAll[item](context.Background(), c, "/v1/project", nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != tt.total {
				t.Fatalf("got %d items, want %d", len(items), tt.total)
			}
			for i, it := range items {
				if it.ID != i {
					t.Fatalf("item %d has id %d", i, it.ID)
				}
			}
			if got := requests.Load(); got != tt.requests {
				t.Errorf("made %d requests, want %d", got, tt.requests)
			}
		})
	}
}

func TestRetriesTransientErrors(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"application":"Dependency-Track","version":"5.2.2"}`))
	}))
	v, err := c.Version(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != "5.2.2" || calls.Load() != 3 {
		t.Fatalf("version %q after %d calls", v.Version, calls.Load())
	}
}

func TestDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	_, err := c.TeamSelf(context.Background())
	if !IsStatus(err, http.StatusUnauthorized) {
		t.Fatalf("expected 401 APIError, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("made %d calls, want 1", calls.Load())
	}
}

func TestGivesUpAfterMaxRetries(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}), func(o *Options) { o.MaxRetries = 2 })
	_, err := c.Version(context.Background())
	if !IsStatus(err, http.StatusTooManyRequests) {
		t.Fatalf("expected 429, got %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("made %d calls, want 3", calls.Load())
	}
}

func TestContextCancelStopsRetries(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}), func(o *Options) { o.RetryWait = time.Hour })
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Version(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestBaseURLNormalization(t *testing.T) {
	for _, base := range []string{"https://dt.example.com", "https://dt.example.com/", "https://dt.example.com/api", "https://dt.example.com/api/"} {
		c, err := New(Options{BaseURL: base})
		if err != nil {
			t.Fatal(err)
		}
		if c.BaseURL() != "https://dt.example.com" {
			t.Errorf("%s normalized to %s", base, c.BaseURL())
		}
	}
	c, _ := New(Options{BaseURL: "https://example.com/dtrack/api"})
	if c.BaseURL() != "https://example.com/dtrack" {
		t.Errorf("sub-path normalized to %s", c.BaseURL())
	}
	for _, bad := range []string{"", "ftp://dt", "://nope"} {
		if _, err := New(Options{BaseURL: bad}); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestMajorVersion(t *testing.T) {
	for in, want := range map[string]int{"5.2.2": 5, "4.14.7": 4, "5.3.0-SNAPSHOT": 5, "v5.0.0": 5} {
		got, err := MajorVersion(in)
		if err != nil || got != want {
			t.Errorf("MajorVersion(%q) = %d, %v", in, got, err)
		}
	}
	if _, err := MajorVersion("garbage"); err == nil {
		t.Error("expected error")
	}
}

func TestMissingPermissions(t *testing.T) {
	team := Team{Permissions: []Permission{{Name: "VIEW_PORTFOLIO"}, {Name: "BOM_UPLOAD"}}}
	got := team.MissingPermissions()
	if fmt.Sprint(got) != "[VIEW_VULNERABILITY VIEW_POLICY_VIOLATION]" {
		t.Fatalf("got %v", got)
	}
}

func TestCWEUnmarshal(t *testing.T) {
	var cwes []CWE
	if err := json.Unmarshal([]byte(`[79, {"cweId": 89, "name": "SQL Injection"}]`), &cwes); err != nil {
		t.Fatal(err)
	}
	if cwes[0].CWEID != 79 || cwes[1].CWEID != 89 || cwes[1].Name != "SQL Injection" {
		t.Fatalf("got %+v", cwes)
	}
}

const (
	fixtureProject = "193995f2-2205-4b1b-ac0b-4e84e16eda35"
	// web-frontend 2.3.0, which has two FAIL violations, one with an APPROVED analysis.
	fixtureViolationProject = "51abccf1-b95d-49c6-b791-32fce5fd1ffe"
)

func fixtureServer(t *testing.T) *Client {
	t.Helper()
	routes := map[string]string{
		"/api/version":                             "version",
		"/api/v1/team/self":                        "team_self",
		"/api/v1/project":                          "projects",
		"/api/v1/tag":                              "tags",
		"/api/v1/metrics/portfolio/current":        "portfolio_current",
		"/api/v1/metrics/portfolio/since/20261026": "portfolio_days",
		"/api/v1/metrics/project/" + fixtureProject + "/current":        "project_current",
		"/api/v1/metrics/project/" + fixtureProject + "/since/20261026": "project_days",
		"/api/v1/finding/project/" + fixtureProject:                     "project_findings",
		"/api/v1/finding":   "findings",
		"/api/v1/violation": "violations",
		"/api/v1/violation/project/" + fixtureViolationProject: "project_violations",
	}
	return newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, ok := routes[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		body, err := os.ReadFile("testdata/" + name + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var raw []json.RawMessage
		if json.Unmarshal(body, &raw) == nil {
			w.Header().Set("X-Total-Count", strconv.Itoa(len(raw)))
			w.Header().Set("X-Total-Count-Type", "EXACT")
		}
		_, _ = w.Write(body)
	}))
}

func TestFixtures(t *testing.T) {
	ctx := context.Background()
	c := fixtureServer(t)
	since := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

	v, err := c.Version(ctx)
	must(t, err)
	if major, _ := MajorVersion(v.Version); major != 5 {
		t.Errorf("version %q", v.Version)
	}

	team, err := c.TeamSelf(ctx)
	must(t, err)
	if missing := team.MissingPermissions(); len(missing) != 0 {
		t.Errorf("seeded team should have all permissions, missing %v", missing)
	}

	projects, err := c.Projects(ctx, ProjectFilter{})
	must(t, err)
	if len(projects) != 5 {
		t.Fatalf("got %d projects", len(projects))
	}
	var latest int
	for _, p := range projects {
		if p.IsLatest {
			latest++
		}
		if p.LastBomImport == nil || p.LastBomImport.IsZero() || len(p.Tags) == 0 || p.Metrics == nil {
			t.Errorf("project %s %s is missing fields: %+v", p.Name, p.Version, p)
		}
	}
	if latest != 4 {
		t.Errorf("got %d latest projects, want 4", latest)
	}

	tags, err := c.Tags(ctx)
	must(t, err)
	if len(tags) == 0 || tags[0].Name == "" {
		t.Errorf("tags: %+v", tags)
	}

	pm, err := c.PortfolioMetricsCurrent(ctx)
	must(t, err)
	if pm.Projects != 5 || pm.Critical != 3 || pm.High != 4 || pm.PolicyViolationsFail != 2 || pm.LastOccurrence.IsZero() {
		t.Errorf("portfolio metrics: %+v", pm)
	}

	series, err := c.PortfolioMetricsSince(ctx, since)
	must(t, err)
	if len(series) != 30 || !series[0].FirstOccurrence.Before(series[29].FirstOccurrence.Time) {
		t.Errorf("portfolio series has %d points", len(series))
	}

	prm, err := c.ProjectMetricsCurrent(ctx, fixtureProject)
	must(t, err)
	if prm.Critical != 1 || prm.High != 2 {
		t.Errorf("project metrics: %+v", prm)
	}
	prs, err := c.ProjectMetricsSince(ctx, fixtureProject, since)
	must(t, err)
	if len(prs) == 0 {
		t.Error("project series is empty")
	}

	findings, err := c.ProjectFindings(ctx, fixtureProject, FindingFilter{})
	must(t, err)
	if len(findings) != 4 {
		t.Fatalf("got %d project findings", len(findings))
	}
	all, err := c.Findings(ctx, PortfolioFindingFilter{})
	must(t, err)
	if len(all) != 13 {
		t.Fatalf("got %d findings", len(all))
	}
	for _, f := range all {
		if f.Component.ProjectName == "" || f.Vulnerability.Severity == "" || f.Vulnerability.CVSSV3BaseScore == nil || f.Attribution.AttributedOn.IsZero() {
			t.Errorf("finding is missing fields: %+v", f)
		}
	}

	violations, err := c.Violations(ctx, ViolationFilter{})
	must(t, err)
	if len(violations) != 8 {
		t.Fatalf("got %d violations", len(violations))
	}
	states := map[string]int{}
	for _, v := range violations {
		states[v.PolicyCondition.Policy.ViolationState]++
		if v.Project.Name == "" || v.Component.Name == "" || v.Timestamp.IsZero() {
			t.Errorf("violation is missing fields: %+v", v)
		}
	}
	if states["FAIL"] != 2 || states["WARN"] != 1 || states["INFO"] != 5 {
		t.Errorf("violation states: %v", states)
	}

	pv, err := c.ProjectViolations(ctx, fixtureViolationProject, ProjectViolationFilter{})
	must(t, err)
	if len(pv) != 2 {
		t.Fatalf("got %d project violations", len(pv))
	}
	var analysed int
	for _, v := range pv {
		if v.PolicyCondition.Policy.ViolationState != "FAIL" || v.Project.UUID != fixtureViolationProject {
			t.Errorf("unexpected violation %+v", v)
		}
		if v.Analysis != nil {
			analysed++
			if v.Analysis.State != "APPROVED" || v.Analysis.IsSuppressed {
				t.Errorf("analysis %+v", v.Analysis)
			}
		}
	}
	if analysed != 1 {
		t.Errorf("got %d analysed violations, want 1", analysed)
	}
}

// The dev instance has no vulnerability mirror, so this payload follows the documented
// /v1/metrics/vulnerability shape instead of a recording.
func TestVulnerabilityMetricsDecoding(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"year":2026,"month":10,"count":3512,"measuredAt":1795611053247},{"year":2026,"month":11,"count":1204,"measuredAt":1795611053247}]`))
	}))
	m, err := c.VulnerabilityMetrics(context.Background())
	must(t, err)
	if len(m) != 2 || m[0].Year != 2026 || m[0].Month != 10 || m[0].Count != 3512 || m[1].Measured.IsZero() {
		t.Fatalf("got %+v", m)
	}
}

func TestPathSegmentsAreEscapedOnce(t *testing.T) {
	var got string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.EscapedPath()
		w.Header().Set("X-Total-Count", "0")
		_, _ = w.Write([]byte(`[]`))
	}))
	_, err := c.Projects(context.Background(), ProjectFilter{Tag: "team a/b%"})
	must(t, err)
	if want := "/api/v1/project/tag/team%20a%2Fb%25"; got != want {
		t.Fatalf("requested %s, want %s", got, want)
	}
}

func TestTimeoutsAreNotRetried(t *testing.T) {
	var requests atomic.Int32
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(200 * time.Millisecond)
	}), func(o *Options) { o.HTTPClient = &http.Client{Timeout: 20 * time.Millisecond} })
	if _, err := c.Version(context.Background()); err == nil {
		t.Fatal("expected a timeout")
	}
	if requests.Load() != 1 {
		t.Fatalf("got %d requests, want 1", requests.Load())
	}
}

func TestBackoffIsCapped(t *testing.T) {
	c, err := New(Options{BaseURL: "http://dt", MaxRetries: 40, RetryWait: time.Second})
	must(t, err)
	for attempt := 1; attempt <= 40; attempt++ {
		if d := c.backoff(attempt); d <= 0 || d > maxRetryWait*3/2+1 {
			t.Fatalf("backoff(%d) = %v", attempt, d)
		}
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
