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
)

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
	QueryType Type            `json:"queryType"`
	Mode      Mode            `json:"mode,omitempty"`
	Metrics   []string        `json:"metrics,omitempty"`
	Projects  ProjectSelector `json:"projects"`
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
	return q, q.validate()
}

func (q *Query) validate() error {
	switch q.QueryType {
	case TypePortfolioMetrics, TypeProjectMetrics:
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
