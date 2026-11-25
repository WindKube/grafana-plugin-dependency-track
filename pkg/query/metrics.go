package query

import "github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"

type Metric struct {
	Name          string
	Display       string
	Severity      string
	PortfolioOnly bool
	Value         func(m dtrack.PortfolioMetrics) float64
}

func count(f func(m dtrack.PortfolioMetrics) int) func(m dtrack.PortfolioMetrics) float64 {
	return func(m dtrack.PortfolioMetrics) float64 { return float64(f(m)) }
}

var metricList = []Metric{
	{Name: "critical", Display: "Critical", Severity: "critical", Value: count(func(m dtrack.PortfolioMetrics) int { return m.Critical })},
	{Name: "high", Display: "High", Severity: "high", Value: count(func(m dtrack.PortfolioMetrics) int { return m.High })},
	{Name: "medium", Display: "Medium", Severity: "medium", Value: count(func(m dtrack.PortfolioMetrics) int { return m.Medium })},
	{Name: "low", Display: "Low", Severity: "low", Value: count(func(m dtrack.PortfolioMetrics) int { return m.Low })},
	{Name: "unassigned", Display: "Unassigned", Severity: "unassigned", Value: count(func(m dtrack.PortfolioMetrics) int { return m.Unassigned })},
	{Name: "vulnerabilities", Display: "Vulnerabilities", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.Vulnerabilities })},
	{Name: "kev", Display: "Known exploited (KEV)", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.KEV })},
	{Name: "components", Display: "Components", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.Components })},
	{Name: "vulnerableComponents", Display: "Vulnerable components", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.VulnerableComponents })},
	{Name: "suppressed", Display: "Suppressed", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.Suppressed })},
	{Name: "findingsTotal", Display: "Findings", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.FindingsTotal })},
	{Name: "findingsAudited", Display: "Findings audited", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.FindingsAudited })},
	{Name: "findingsUnaudited", Display: "Findings unaudited", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.FindingsUnaudited })},
	{Name: "inheritedRiskScore", Display: "Inherited risk score", Severity: "", Value: func(m dtrack.PortfolioMetrics) float64 { return m.InheritedRiskScore }},
	{Name: "policyViolationsFail", Display: "Policy violations (fail)", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.PolicyViolationsFail })},
	{Name: "policyViolationsWarn", Display: "Policy violations (warn)", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.PolicyViolationsWarn })},
	{Name: "policyViolationsInfo", Display: "Policy violations (info)", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.PolicyViolationsInfo })},
	{Name: "policyViolationsTotal", Display: "Policy violations", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.PolicyViolationsTotal })},
	{Name: "policyViolationsUnaudited", Display: "Policy violations unaudited", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.PolicyViolationsUnaudited })},
	{Name: "policyViolationsSecurityTotal", Display: "Security violations", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.PolicyViolationsSecurityTotal })},
	{Name: "policyViolationsLicenseTotal", Display: "License violations", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.PolicyViolationsLicenseTotal })},
	{Name: "policyViolationsOperationalTotal", Display: "Operational violations", Severity: "", Value: count(func(m dtrack.PortfolioMetrics) int { return m.PolicyViolationsOperationalTotal })},
	{Name: "projects", Display: "Projects", PortfolioOnly: true, Value: count(func(m dtrack.PortfolioMetrics) int { return m.Projects })},
	{Name: "vulnerableProjects", Display: "Vulnerable projects", PortfolioOnly: true, Value: count(func(m dtrack.PortfolioMetrics) int { return m.VulnerableProjects })},
}

var Metrics = func() map[string]Metric {
	m := make(map[string]Metric, len(metricList))
	for _, metric := range metricList {
		m[metric.Name] = metric
	}
	return m
}()
