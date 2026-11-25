package plugin

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"
	"github.com/WindKube/grafana-plugin-dependency-track/pkg/query"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"golang.org/x/sync/errgroup"
)

// collectPerProject resolves the selected projects and gathers items for each of them.
// Selecting all projects uses one portfolio-wide query instead of a request per project.
func collectPerProject[T any](
	ctx context.Context,
	d *Datasource,
	sel query.ProjectSelector,
	global func(context.Context) ([]T, error),
	perProject func(context.Context, string) ([]T, error),
	projectOf func(T) string,
) ([]dtrack.Project, map[string][]T, error) {
	projects, err := d.resolveProjects(ctx, sel)
	if err != nil {
		return nil, nil, err
	}
	byProject := make(map[string][]T, len(projects))
	for _, p := range projects {
		byProject[p.UUID] = nil
	}
	if len(projects) == 0 {
		return projects, byProject, nil
	}

	if sel.Mode == query.SelectAll {
		items, err := global(ctx)
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			if uuid := projectOf(it); hasKey(byProject, uuid) {
				byProject[uuid] = append(byProject[uuid], it)
			}
		}
		return projects, byProject, nil
	}

	results := make([][]T, len(projects))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)
	for i, p := range projects {
		g.Go(func() error {
			items, err := perProject(gctx, p.UUID)
			if err != nil {
				return fmt.Errorf("%s %s: %w", p.Name, p.Version, err)
			}
			results[i] = items
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}
	for i, p := range projects {
		byProject[p.UUID] = results[i]
	}
	return projects, byProject, nil
}

func hasKey[K comparable, V any](m map[K]V, k K) bool {
	_, ok := m[k]
	return ok
}

func (d *Datasource) findings(ctx context.Context, q *query.Query) (data.Frames, error) {
	projects, byProject, err := d.collectFindings(ctx, q)
	if err != nil {
		return nil, err
	}
	if q.Format == query.FormatCount {
		return data.Frames{findingCounts(projects, byProject, q.Findings)}, nil
	}
	return data.Frames{findingTable(projects, byProject)}, nil
}

func (d *Datasource) collectFindings(ctx context.Context, q *query.Query) ([]dtrack.Project, map[string][]dtrack.Finding, error) {
	f := q.Findings
	projects, byProject, err := collectPerProject(ctx, d, q.Projects,
		func(ctx context.Context) ([]dtrack.Finding, error) {
			return d.client.Findings(ctx, dtrack.PortfolioFindingFilter{
				ShowSuppressed: f.IncludeSuppressed,
				ShowInactive:   q.Projects.IncludeInactive,
				IsKEV:          f.KEVOnly,
				EPSSFrom:       f.EPSSMin,
			})
		},
		func(ctx context.Context, uuid string) ([]dtrack.Finding, error) {
			return d.client.ProjectFindings(ctx, uuid, dtrack.FindingFilter{Suppressed: f.IncludeSuppressed, IsKEV: f.KEVOnly, EPSSFrom: f.EPSSMin})
		},
		func(fi dtrack.Finding) string { return fi.Component.Project },
	)
	if err != nil {
		return nil, nil, err
	}
	for uuid, items := range byProject {
		byProject[uuid] = slices.DeleteFunc(items, func(fi dtrack.Finding) bool { return !matchFinding(fi, f) })
	}
	return projects, byProject, nil
}

func matchFinding(fi dtrack.Finding, f query.FindingFilter) bool {
	v := fi.Vulnerability
	if !slices.Contains(f.SelectedSeverities(), strings.ToUpper(v.Severity)) {
		return false
	}
	if len(f.AnalysisStates) > 0 && !slices.Contains(f.AnalysisStates, analysisState(fi.Analysis.State)) {
		return false
	}
	if fi.Analysis.IsSuppressed && !f.IncludeSuppressed {
		return false
	}
	if f.KEVOnly && !v.IsKEV {
		return false
	}
	if f.CVSSMin != nil {
		score := cvss(v)
		if score == nil || *score < *f.CVSSMin {
			return false
		}
	}
	if f.EPSSMin != nil && (v.EPSSScore == nil || *v.EPSSScore < *f.EPSSMin) {
		return false
	}
	return true
}

// cvss returns the newest CVSS score available.
func cvss(v dtrack.FindingVulnerability) *float64 {
	for _, s := range []*float64{v.CVSSV4Score, v.CVSSV3BaseScore, v.CVSSV2BaseScore} {
		if s != nil {
			return s
		}
	}
	return nil
}

func analysisState(s string) string {
	if s == "" {
		return "NOT_SET"
	}
	return s
}

func findingTable(projects []dtrack.Project, byProject map[string][]dtrack.Finding) *data.Frame {
	var (
		project, version, tags, component, componentVersion, purl []string
		vulnID, source, severity, state                           []string
		projectUUID, componentUUID, vulnUUID                      []string
		score, epss                                               []*float64
		kev, suppressed                                           []bool
		attributed                                                []time.Time
	)
	for _, p := range projects {
		labels := projectLabels(p)
		for _, fi := range byProject[p.UUID] {
			project = append(project, p.Name)
			version = append(version, p.Version)
			tags = append(tags, labels["tags"])
			component = append(component, componentName(fi.Component.Group, fi.Component.Name))
			componentVersion = append(componentVersion, fi.Component.Version)
			purl = append(purl, fi.Component.PURL)
			vulnID = append(vulnID, fi.Vulnerability.VulnID)
			source = append(source, fi.Vulnerability.Source)
			severity = append(severity, fi.Vulnerability.Severity)
			score = append(score, cvss(fi.Vulnerability))
			epss = append(epss, fi.Vulnerability.EPSSScore)
			kev = append(kev, fi.Vulnerability.IsKEV)
			state = append(state, analysisState(fi.Analysis.State))
			suppressed = append(suppressed, fi.Analysis.IsSuppressed)
			attributed = append(attributed, fi.Attribution.AttributedOn.Time)
			projectUUID = append(projectUUID, p.UUID)
			componentUUID = append(componentUUID, fi.Component.UUID)
			vulnUUID = append(vulnUUID, fi.Vulnerability.UUID)
		}
	}
	frame := data.NewFrame("findings",
		data.NewField("Project", nil, nonNil(project)),
		data.NewField("Version", nil, nonNil(version)),
		data.NewField("Tags", nil, nonNil(tags)),
		data.NewField("Component", nil, nonNil(component)),
		data.NewField("Component version", nil, nonNil(componentVersion)),
		data.NewField("PURL", nil, nonNil(purl)),
		data.NewField("Vulnerability", nil, nonNil(vulnID)),
		data.NewField("Source", nil, nonNil(source)),
		data.NewField("Severity", nil, nonNil(severity)),
		data.NewField("CVSS", nil, nonNil(score)),
		data.NewField("EPSS", nil, nonNil(epss)),
		data.NewField("KEV", nil, nonNil(kev)),
		data.NewField("Analysis", nil, nonNil(state)),
		data.NewField("Suppressed", nil, nonNil(suppressed)),
		data.NewField("Attributed", nil, nonNil(attributed)),
		data.NewField("Project UUID", nil, nonNil(projectUUID)),
		data.NewField("Component UUID", nil, nonNil(componentUUID)),
		data.NewField("Vulnerability UUID", nil, nonNil(vulnUUID)),
	)
	frame.SetMeta(&data.FrameMeta{PreferredVisualization: data.VisTypeTable})
	return frame
}

func findingCounts(projects []dtrack.Project, byProject map[string][]dtrack.Finding, f query.FindingFilter) *data.Frame {
	b := newCountBuilder("severity")
	for _, p := range projects {
		counts := map[string]int64{}
		for _, fi := range byProject[p.UUID] {
			counts[strings.ToUpper(fi.Vulnerability.Severity)]++
		}
		for _, sev := range f.SelectedSeverities() {
			b.add(p, counts[sev], strings.ToLower(sev))
		}
	}
	return b.frame("findings")
}

func componentName(group, name string) string {
	if group == "" {
		return name
	}
	return group + "/" + name
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// countBuilder builds a numeric-long frame: string columns become alert labels, and
// "count" is the single numeric column, so every row is one alert instance.
type countBuilder struct {
	dims    []string
	project []string
	version []string
	tags    []string
	values  [][]string
	count   []int64
}

func newCountBuilder(dims ...string) *countBuilder {
	return &countBuilder{dims: dims, values: make([][]string, len(dims))}
}

func (b *countBuilder) add(p dtrack.Project, count int64, dimValues ...string) {
	labels := projectLabels(p)
	b.project = append(b.project, p.Name)
	b.version = append(b.version, p.Version)
	b.tags = append(b.tags, labels["tags"])
	for i, v := range dimValues {
		b.values[i] = append(b.values[i], v)
	}
	b.count = append(b.count, count)
}

func (b *countBuilder) frame(name string) *data.Frame {
	fields := []*data.Field{
		data.NewField("project", nil, nonNil(b.project)),
		data.NewField("version", nil, nonNil(b.version)),
		data.NewField("tags", nil, nonNil(b.tags)),
	}
	for i, dim := range b.dims {
		fields = append(fields, data.NewField(dim, nil, nonNil(b.values[i])))
	}
	fields = append(fields, data.NewField("count", nil, nonNil(b.count)))
	frame := data.NewFrame(name, fields...)
	frame.SetMeta(&data.FrameMeta{Type: data.FrameTypeNumericLong, TypeVersion: data.FrameTypeVersion{0, 1}})
	return frame
}
