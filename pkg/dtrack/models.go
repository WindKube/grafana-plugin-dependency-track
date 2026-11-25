package dtrack

import (
	"bytes"
	"encoding/json"
	"strconv"
	"time"
)

// Millis is a Unix timestamp in milliseconds, as used throughout the Dependency-Track API.
type Millis struct{ time.Time }

func (m *Millis) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		m.Time = time.Time{}
		return nil
	}
	ms, err := strconv.ParseInt(string(bytes.Trim(b, `"`)), 10, 64)
	if err != nil {
		return err
	}
	m.Time = time.UnixMilli(ms).UTC()
	return nil
}

func (m Millis) MarshalJSON() ([]byte, error) {
	if m.IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(m.UnixMilli(), 10)), nil
}

type Version struct {
	Application string `json:"application"`
	Version     string `json:"version"`
	Timestamp   string `json:"timestamp"`
	UUID        string `json:"uuid"`
}

type Permission struct {
	Name string `json:"name"`
}

type Team struct {
	UUID        string       `json:"uuid"`
	Name        string       `json:"name"`
	Permissions []Permission `json:"permissions"`
}

type Tag struct {
	Name         string `json:"name"`
	ProjectCount int    `json:"projectCount,omitempty"`
}

type Project struct {
	UUID                      string          `json:"uuid"`
	Name                      string          `json:"name"`
	Version                   string          `json:"version"`
	Group                     string          `json:"group,omitempty"`
	Classifier                string          `json:"classifier"`
	Active                    bool            `json:"active"`
	IsLatest                  bool            `json:"isLatest"`
	Tags                      []Tag           `json:"tags"`
	LastBomImport             *Millis         `json:"lastBomImport,omitempty"`
	LastBomImportFormat       string          `json:"lastBomImportFormat,omitempty"`
	LastVulnerabilityAnalysis *Millis         `json:"lastVulnerabilityAnalysis,omitempty"`
	LastInheritedRiskScore    float64         `json:"lastInheritedRiskScore"`
	Metrics                   *ProjectMetrics `json:"metrics,omitempty"`
}

func (p Project) TagNames() []string {
	names := make([]string, 0, len(p.Tags))
	for _, t := range p.Tags {
		names = append(names, t.Name)
	}
	return names
}

// Metrics holds the counters shared by portfolio and project metrics.
type Metrics struct {
	Critical                             int     `json:"critical"`
	High                                 int     `json:"high"`
	Medium                               int     `json:"medium"`
	Low                                  int     `json:"low"`
	Unassigned                           int     `json:"unassigned"`
	KEV                                  int     `json:"kev"`
	Vulnerabilities                      int     `json:"vulnerabilities"`
	Components                           int     `json:"components"`
	VulnerableComponents                 int     `json:"vulnerableComponents"`
	Suppressed                           int     `json:"suppressed"`
	FindingsTotal                        int     `json:"findingsTotal"`
	FindingsAudited                      int     `json:"findingsAudited"`
	FindingsUnaudited                    int     `json:"findingsUnaudited"`
	InheritedRiskScore                   float64 `json:"inheritedRiskScore"`
	PolicyViolationsFail                 int     `json:"policyViolationsFail"`
	PolicyViolationsWarn                 int     `json:"policyViolationsWarn"`
	PolicyViolationsInfo                 int     `json:"policyViolationsInfo"`
	PolicyViolationsTotal                int     `json:"policyViolationsTotal"`
	PolicyViolationsAudited              int     `json:"policyViolationsAudited"`
	PolicyViolationsUnaudited            int     `json:"policyViolationsUnaudited"`
	PolicyViolationsSecurityTotal        int     `json:"policyViolationsSecurityTotal"`
	PolicyViolationsSecurityAudited      int     `json:"policyViolationsSecurityAudited"`
	PolicyViolationsSecurityUnaudited    int     `json:"policyViolationsSecurityUnaudited"`
	PolicyViolationsLicenseTotal         int     `json:"policyViolationsLicenseTotal"`
	PolicyViolationsLicenseAudited       int     `json:"policyViolationsLicenseAudited"`
	PolicyViolationsLicenseUnaudited     int     `json:"policyViolationsLicenseUnaudited"`
	PolicyViolationsOperationalTotal     int     `json:"policyViolationsOperationalTotal"`
	PolicyViolationsOperationalAudited   int     `json:"policyViolationsOperationalAudited"`
	PolicyViolationsOperationalUnaudited int     `json:"policyViolationsOperationalUnaudited"`
	FirstOccurrence                      Millis  `json:"firstOccurrence"`
	LastOccurrence                       Millis  `json:"lastOccurrence"`
}

type PortfolioMetrics struct {
	Metrics
	Projects           int `json:"projects"`
	VulnerableProjects int `json:"vulnerableProjects"`
}

type ProjectMetrics struct {
	Metrics
}

type VulnerabilityMetrics struct {
	Year     int    `json:"year"`
	Month    int    `json:"month"`
	Count    int    `json:"count"`
	Measured Millis `json:"measuredAt"`
}

type FindingComponent struct {
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	Group          string `json:"group,omitempty"`
	Version        string `json:"version"`
	PURL           string `json:"purl,omitempty"`
	Project        string `json:"project"`
	ProjectName    string `json:"projectName"`
	ProjectVersion string `json:"projectVersion"`
	LatestVersion  string `json:"latestVersion,omitempty"`
}

type FindingVulnerability struct {
	UUID            string   `json:"uuid"`
	Source          string   `json:"source"`
	VulnID          string   `json:"vulnId"`
	Title           string   `json:"title,omitempty"`
	Severity        string   `json:"severity"`
	SeverityRank    int      `json:"severityRank"`
	CVSSV2BaseScore *float64 `json:"cvssV2BaseScore,omitempty"`
	CVSSV3BaseScore *float64 `json:"cvssV3BaseScore,omitempty"`
	CVSSV4Score     *float64 `json:"cvssV4Score,omitempty"`
	EPSSScore       *float64 `json:"epssScore,omitempty"`
	EPSSPercentile  *float64 `json:"epssPercentile,omitempty"`
	IsKEV           bool     `json:"isKev"`
	Published       *Millis  `json:"published,omitempty"`
	CWEs            []CWE    `json:"cwes,omitempty"`
	Aliases         []Alias  `json:"aliases,omitempty"`
}

// CWE tolerates both the v5 object form and a bare integer ID.
type CWE struct {
	CWEID int    `json:"cweId"`
	Name  string `json:"name,omitempty"`
}

func (c *CWE) UnmarshalJSON(b []byte) error {
	if id, err := strconv.Atoi(string(bytes.TrimSpace(b))); err == nil {
		c.CWEID = id
		return nil
	}
	type plain CWE
	return json.Unmarshal(b, (*plain)(c))
}

type Alias struct {
	CVEID  string `json:"cveId,omitempty"`
	GHSAID string `json:"ghsaId,omitempty"`
	OSVID  string `json:"osvId,omitempty"`
	SNYKID string `json:"snykId,omitempty"`
}

type FindingAnalysis struct {
	State        string `json:"state,omitempty"`
	IsSuppressed bool   `json:"isSuppressed"`
}

type FindingAttribution struct {
	AnalyzerIdentity string `json:"analyzerIdentity"`
	AttributedOn     Millis `json:"attributedOn"`
}

type Finding struct {
	Component     FindingComponent     `json:"component"`
	Vulnerability FindingVulnerability `json:"vulnerability"`
	Analysis      FindingAnalysis      `json:"analysis"`
	Attribution   FindingAttribution   `json:"attribution"`
	Matrix        string               `json:"matrix"`
}

type ViolationComponent struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Group   string `json:"group,omitempty"`
	Version string `json:"version"`
	PURL    string `json:"purl,omitempty"`
}

type ViolationPolicy struct {
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	ViolationState string `json:"violationState"`
}

type PolicyCondition struct {
	UUID          string          `json:"uuid"`
	Subject       string          `json:"subject"`
	Operator      string          `json:"operator"`
	Value         string          `json:"value"`
	ViolationType string          `json:"violationType"`
	Policy        ViolationPolicy `json:"policy"`
}

type ViolationAnalysis struct {
	State        string `json:"analysisState,omitempty"`
	IsSuppressed bool   `json:"isSuppressed"`
}

type Violation struct {
	UUID            string             `json:"uuid"`
	Type            string             `json:"type"`
	Timestamp       Millis             `json:"timestamp"`
	Project         Project            `json:"project"`
	Component       ViolationComponent `json:"component"`
	PolicyCondition PolicyCondition    `json:"policyCondition"`
	Analysis        *ViolationAnalysis `json:"analysis,omitempty"`
}
