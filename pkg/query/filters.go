package query

import (
	"fmt"
	"slices"
)

func (q *Query) validateFilters() error {
	checks := []struct {
		name    string
		values  []string
		allowed []string
	}{
		{"severity", q.Findings.Severities, Severities},
		{"analysis state", q.Findings.AnalysisStates, AnalysisStates},
		{"violation state", q.Violations.States, ViolationStates},
		{"risk type", q.Violations.RiskTypes, RiskTypes},
	}
	for _, c := range checks {
		for _, v := range c.values {
			if !slices.Contains(c.allowed, v) {
				return fmt.Errorf("unknown %s %q", c.name, v)
			}
		}
	}
	return nil
}

// orAll returns values, or all allowed values when none are selected.
func orAll(values, all []string) []string {
	if len(values) == 0 {
		return all
	}
	return values
}

func (f FindingFilter) SelectedSeverities() []string { return orAll(f.Severities, Severities) }

func (f ViolationFilter) SelectedStates() []string { return orAll(f.States, ViolationStates) }

func (f ViolationFilter) SelectedRiskTypes() []string { return orAll(f.RiskTypes, RiskTypes) }
