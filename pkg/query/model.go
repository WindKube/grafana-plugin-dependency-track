package query

import (
	"encoding/json"
	"fmt"
	"regexp"
)

type Type string

const (
	TypePortfolioMetrics Type = "portfolioMetrics"
	TypeProjectMetrics   Type = "projectMetrics"
	TypeFindings         Type = "findings"
	TypeViolations       Type = "violations"
)

type Format string

const (
	FormatTable Format = "table"
	FormatCount Format = "count"
)

var (
	Severities      = []string{"CRITICAL", "HIGH", "MEDIUM", "LOW", "UNASSIGNED"}
	AnalysisStates  = []string{"NOT_SET", "EXPLOITABLE", "IN_TRIAGE", "FALSE_POSITIVE", "NOT_AFFECTED", "RESOLVED"}
	ViolationStates = []string{"FAIL", "WARN", "INFO"}
	RiskTypes       = []string{"SECURITY", "LICENSE", "OPERATIONAL"}
)

type FindingFilter struct {
	Severities        []string `json:"severities,omitempty"`
	AnalysisStates    []string `json:"analysisStates,omitempty"`
	IncludeSuppressed bool     `json:"includeSuppressed,omitempty"`
	CVSSMin           *float64 `json:"cvssMin,omitempty"`
	EPSSMin           *float64 `json:"epssMin,omitempty"`
	KEVOnly           bool     `json:"kevOnly,omitempty"`
}

type ViolationFilter struct {
	States            []string `json:"states,omitempty"`
	RiskTypes         []string `json:"riskTypes,omitempty"`
	Policies          []string `json:"policies,omitempty"`
	IncludeSuppressed bool     `json:"includeSuppressed,omitempty"`
	GroupByPolicy     bool     `json:"groupByPolicy,omitempty"`
}

type Mode string

const (
	ModeCurrent    Mode = "current"
	ModeTimeSeries Mode = "timeseries"
)

type SelectorMode string

const (
	SelectAll        SelectorMode = "all"
	SelectUUID       SelectorMode = "uuid"
	SelectName       SelectorMode = "name"
	SelectTag        SelectorMode = "tag"
	SelectClassifier SelectorMode = "classifier"
)

type ProjectSelector struct {
	Mode            SelectorMode `json:"mode,omitempty"`
	UUIDs           []string     `json:"uuids,omitempty"`
	NameRegex       string       `json:"nameRegex,omitempty"`
	Tag             string       `json:"tag,omitempty"`
	Classifier      string       `json:"classifier,omitempty"`
	OnlyLatest      bool         `json:"onlyLatest,omitempty"`
	IncludeInactive bool         `json:"includeInactive,omitempty"`
}

type Query struct {
	QueryType  Type            `json:"queryType"`
	Mode       Mode            `json:"mode,omitempty"`
	Metrics    []string        `json:"metrics,omitempty"`
	Projects   ProjectSelector `json:"projects"`
	Format     Format          `json:"format,omitempty"`
	Findings   FindingFilter   `json:"findings"`
	Violations ViolationFilter `json:"violations"`
}

var DefaultMetrics = []string{"critical", "high", "medium", "low", "unassigned"}

func Parse(raw json.RawMessage) (*Query, error) {
	q := &Query{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, q); err != nil {
			return nil, fmt.Errorf("invalid query: %w", err)
		}
	}
	if q.QueryType == "" {
		q.QueryType = TypePortfolioMetrics
	}
	if q.Mode == "" {
		q.Mode = ModeCurrent
	}
	if len(q.Metrics) == 0 {
		q.Metrics = DefaultMetrics
	}
	if q.Projects.Mode == "" {
		q.Projects.Mode = SelectAll
	}
	if q.Format == "" {
		q.Format = FormatTable
	}
	return q, q.validate()
}

func (q *Query) validate() error {
	switch q.QueryType {
	case TypePortfolioMetrics, TypeProjectMetrics:
	case TypeFindings, TypeViolations:
		if q.Format != FormatTable && q.Format != FormatCount {
			return fmt.Errorf("unknown format %q", q.Format)
		}
		if err := q.validateFilters(); err != nil {
			return err
		}
		return q.Projects.validate()
	default:
		return fmt.Errorf("unknown query type %q", q.QueryType)
	}
	if q.Mode != ModeCurrent && q.Mode != ModeTimeSeries {
		return fmt.Errorf("unknown mode %q", q.Mode)
	}
	for _, m := range q.Metrics {
		metric, ok := Metrics[m]
		if !ok {
			return fmt.Errorf("unknown metric %q", m)
		}
		if metric.PortfolioOnly && q.QueryType != TypePortfolioMetrics {
			return fmt.Errorf("metric %q is only available for portfolio metrics", m)
		}
	}
	return q.Projects.validate()
}

func (s ProjectSelector) validate() error {
	switch s.Mode {
	case SelectAll:
	case SelectUUID:
		if len(s.UUIDs) == 0 {
			return fmt.Errorf("select at least one project")
		}
	case SelectName:
		if _, err := regexp.Compile(s.NameRegex); err != nil {
			return fmt.Errorf("invalid project name regex: %w", err)
		}
	case SelectTag:
		if s.Tag == "" {
			return fmt.Errorf("tag is required")
		}
	case SelectClassifier:
		if s.Classifier == "" {
			return fmt.Errorf("classifier is required")
		}
	default:
		return fmt.Errorf("unknown project selector %q", s.Mode)
	}
	return nil
}
