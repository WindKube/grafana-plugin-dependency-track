package dtrack

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const MinSupportedMajor = 5

var RequiredPermissions = []string{"VIEW_PORTFOLIO", "VIEW_VULNERABILITY", "VIEW_POLICY_VIOLATION"}

func (c *Client) Version(ctx context.Context) (*Version, error) {
	var v Version
	if _, err := c.get(ctx, "/version", nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// MajorVersion parses the major component of a Dependency-Track version such as "5.2.2" or "5.3.0-SNAPSHOT".
func MajorVersion(version string) (int, error) {
	major, _, _ := strings.Cut(strings.TrimPrefix(version, "v"), ".")
	n, err := strconv.Atoi(major)
	if err != nil {
		return 0, fmt.Errorf("dependency-track: unrecognized version %q", version)
	}
	return n, nil
}

func (c *Client) TeamSelf(ctx context.Context) (*Team, error) {
	var t Team
	if _, err := c.get(ctx, "/v1/team/self", nil, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// MissingPermissions returns the required permissions the team does not have.
func (t *Team) MissingPermissions() []string {
	have := make(map[string]bool, len(t.Permissions))
	for _, p := range t.Permissions {
		have[p.Name] = true
	}
	var missing []string
	for _, p := range RequiredPermissions {
		if !have[p] {
			missing = append(missing, p)
		}
	}
	return missing
}

type ProjectFilter struct {
	Name            string
	Tag             string
	Classifier      string
	ExcludeInactive bool
	OnlyRoot        bool
}

func (c *Client) Projects(ctx context.Context, f ProjectFilter) ([]Project, error) {
	q := url.Values{}
	if f.Name != "" {
		q.Set("name", f.Name)
	}
	if f.ExcludeInactive {
		q.Set("excludeInactive", "true")
	}
	if f.OnlyRoot {
		q.Set("onlyRoot", "true")
	}
	path := "/v1/project"
	switch {
	case f.Tag != "":
		path = "/v1/project/tag/" + url.PathEscape(f.Tag)
	case f.Classifier != "":
		path = "/v1/project/classifier/" + url.PathEscape(f.Classifier)
	}
	return getAll[Project](ctx, c, path, q)
}

func (c *Client) Project(ctx context.Context, uuid string) (*Project, error) {
	var p Project
	if _, err := c.get(ctx, "/v1/project/"+url.PathEscape(uuid), nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) Tags(ctx context.Context) ([]Tag, error) {
	return getAll[Tag](ctx, c, "/v1/tag", nil)
}

func (c *Client) PortfolioMetricsCurrent(ctx context.Context) (*PortfolioMetrics, error) {
	var m PortfolioMetrics
	if _, err := c.get(ctx, "/v1/metrics/portfolio/current", nil, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *Client) PortfolioMetricsSince(ctx context.Context, since time.Time) ([]PortfolioMetrics, error) {
	var m []PortfolioMetrics
	if _, err := c.get(ctx, "/v1/metrics/portfolio/since/"+formatDate(since), nil, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *Client) ProjectMetricsCurrent(ctx context.Context, uuid string) (*ProjectMetrics, error) {
	var m ProjectMetrics
	if _, err := c.get(ctx, "/v1/metrics/project/"+url.PathEscape(uuid)+"/current", nil, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *Client) ProjectMetricsSince(ctx context.Context, uuid string, since time.Time) ([]ProjectMetrics, error) {
	var m []ProjectMetrics
	if _, err := c.get(ctx, "/v1/metrics/project/"+url.PathEscape(uuid)+"/since/"+formatDate(since), nil, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *Client) VulnerabilityMetrics(ctx context.Context) ([]VulnerabilityMetrics, error) {
	var m []VulnerabilityMetrics
	if _, err := c.get(ctx, "/v1/metrics/vulnerability", nil, &m); err != nil {
		return nil, err
	}
	return m, nil
}

type FindingFilter struct {
	Suppressed bool
	Source     string
	EPSSFrom   *float64
	EPSSTo     *float64
	IsKEV      bool
}

func (c *Client) ProjectFindings(ctx context.Context, uuid string, f FindingFilter) ([]Finding, error) {
	q := url.Values{}
	setBool(q, "suppressed", f.Suppressed)
	setBool(q, "isKev", f.IsKEV)
	setFloat(q, "epssFrom", f.EPSSFrom)
	setFloat(q, "epssTo", f.EPSSTo)
	if f.Source != "" {
		q.Set("source", f.Source)
	}
	return getAll[Finding](ctx, c, "/v1/finding/project/"+url.PathEscape(uuid), q)
}

type PortfolioFindingFilter struct {
	ShowSuppressed bool
	ShowInactive   bool
	Severity       string
	CVSSv3From     *float64
	CVSSv3To       *float64
	EPSSFrom       *float64
	EPSSTo         *float64
	IsKEV          bool
}

// Findings queries findings across the portfolio. The server caps these queries at 60s.
func (c *Client) Findings(ctx context.Context, f PortfolioFindingFilter) ([]Finding, error) {
	q := url.Values{}
	setBool(q, "showSuppressed", f.ShowSuppressed)
	setBool(q, "showInactive", f.ShowInactive)
	setBool(q, "isKev", f.IsKEV)
	setFloat(q, "cvssv3From", f.CVSSv3From)
	setFloat(q, "cvssv3To", f.CVSSv3To)
	setFloat(q, "epssFrom", f.EPSSFrom)
	setFloat(q, "epssTo", f.EPSSTo)
	if f.Severity != "" {
		q.Set("severity", f.Severity)
	}
	return getAll[Finding](ctx, c, "/v1/finding", q)
}

type ViolationFilter struct {
	Suppressed     bool
	ShowInactive   bool
	ViolationState string
	RiskType       string
	Policy         string
}

// ProjectViolationFilter holds the only filter /v1/violation/project/{uuid} supports; filter
// by state, risk type or policy client-side.
type ProjectViolationFilter struct {
	Suppressed bool
}

func (c *Client) ProjectViolations(ctx context.Context, uuid string, f ProjectViolationFilter) ([]Violation, error) {
	q := url.Values{}
	setBool(q, "suppressed", f.Suppressed)
	return getAll[Violation](ctx, c, "/v1/violation/project/"+url.PathEscape(uuid), q)
}

func (c *Client) Violations(ctx context.Context, f ViolationFilter) ([]Violation, error) {
	q := url.Values{}
	setBool(q, "suppressed", f.Suppressed)
	setBool(q, "showInactive", f.ShowInactive)
	for k, v := range map[string]string{"violationState": f.ViolationState, "riskType": f.RiskType, "policy": f.Policy} {
		if v != "" {
			q.Set(k, v)
		}
	}
	return getAll[Violation](ctx, c, "/v1/violation", q)
}

func setBool(q url.Values, key string, v bool) {
	if v {
		q.Set(key, "true")
	}
}

func setFloat(q url.Values, key string, v *float64) {
	if v != nil {
		q.Set(key, strconv.FormatFloat(*v, 'f', -1, 64))
	}
}

func formatDate(t time.Time) string {
	return t.UTC().Format("20060102")
}

type Policy struct {
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	ViolationState string `json:"violationState"`
}

// Policies requires POLICY_MANAGEMENT; view-only keys get 403.
func (c *Client) Policies(ctx context.Context) ([]Policy, error) {
	return getAll[Policy](ctx, c, "/v1/policy", nil)
}
