package plugin

import (
	"context"
	"time"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"
	"github.com/WindKube/grafana-plugin-dependency-track/pkg/query"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func (d *Datasource) projectInventory(ctx context.Context, q *query.Query) (data.Frames, error) {
	projects, err := d.resolveProjects(ctx, q.Projects)
	if err != nil {
		return nil, err
	}
	now := d.now()
	if q.Format == query.FormatBOMAge {
		return data.Frames{bomAgeFrame(projects, now)}, nil
	}
	return data.Frames{inventoryTable(projects, now)}, nil
}

func bomAge(p dtrack.Project, now time.Time) *float64 {
	if p.LastBomImport == nil || p.LastBomImport.IsZero() {
		return nil
	}
	age := now.Sub(p.LastBomImport.Time).Seconds()
	return &age
}

func optionalTime(m *dtrack.Millis) *time.Time {
	if m == nil || m.IsZero() {
		return nil
	}
	t := m.Time
	return &t
}

// bomAgeFrame is a numeric-long frame for "no BOM uploaded in N days" alerts. Projects
// that never received a BOM have a null age.
func bomAgeFrame(projects []dtrack.Project, now time.Time) *data.Frame {
	var name, version, tags, classifier []string
	var ages []*float64
	for _, p := range projects {
		name = append(name, p.Name)
		version = append(version, p.Version)
		tags = append(tags, projectLabels(p)["tags"])
		classifier = append(classifier, p.Classifier)
		ages = append(ages, bomAge(p, now))
	}
	ageField := data.NewField("bom_age_seconds", nil, nonNil(ages))
	ageField.Config = &data.FieldConfig{Unit: "s", DisplayName: "BOM age"}
	frame := data.NewFrame("bomAge",
		data.NewField("project", nil, nonNil(name)),
		data.NewField("version", nil, nonNil(version)),
		data.NewField("tags", nil, nonNil(tags)),
		data.NewField("classifier", nil, nonNil(classifier)),
		ageField,
	)
	frame.SetMeta(&data.FrameMeta{Type: data.FrameTypeNumericLong, TypeVersion: data.FrameTypeVersion{0, 1}})
	return frame
}

func inventoryTable(projects []dtrack.Project, now time.Time) *data.Frame {
	var (
		name, version, classifier, tags, bomFormat, uuid []string
		active, latest                                   []bool
		lastImport, lastAnalysis                         []*time.Time
		age, risk                                        []*float64
		critical, high, medium, low, unassigned, fail    []*int64
	)
	metric := func(p dtrack.Project, f func(dtrack.Metrics) int) *int64 {
		if p.Metrics == nil {
			return nil
		}
		v := int64(f(p.Metrics.Metrics))
		return &v
	}
	for _, p := range projects {
		name = append(name, p.Name)
		version = append(version, p.Version)
		classifier = append(classifier, p.Classifier)
		tags = append(tags, projectLabels(p)["tags"])
		active = append(active, p.Active)
		latest = append(latest, p.IsLatest)
		lastImport = append(lastImport, optionalTime(p.LastBomImport))
		bomFormat = append(bomFormat, p.LastBomImportFormat)
		age = append(age, bomAge(p, now))
		lastAnalysis = append(lastAnalysis, optionalTime(p.LastVulnerabilityAnalysis))
		r := p.LastInheritedRiskScore
		risk = append(risk, &r)
		critical = append(critical, metric(p, func(m dtrack.Metrics) int { return m.Critical }))
		high = append(high, metric(p, func(m dtrack.Metrics) int { return m.High }))
		medium = append(medium, metric(p, func(m dtrack.Metrics) int { return m.Medium }))
		low = append(low, metric(p, func(m dtrack.Metrics) int { return m.Low }))
		unassigned = append(unassigned, metric(p, func(m dtrack.Metrics) int { return m.Unassigned }))
		fail = append(fail, metric(p, func(m dtrack.Metrics) int { return m.PolicyViolationsFail }))
		uuid = append(uuid, p.UUID)
	}
	ageField := data.NewField("BOM age", nil, nonNil(age))
	ageField.Config = &data.FieldConfig{Unit: "s"}
	frame := data.NewFrame("projects",
		data.NewField("Project", nil, nonNil(name)),
		data.NewField("Version", nil, nonNil(version)),
		data.NewField("Classifier", nil, nonNil(classifier)),
		data.NewField("Tags", nil, nonNil(tags)),
		data.NewField("Active", nil, nonNil(active)),
		data.NewField("Latest", nil, nonNil(latest)),
		data.NewField("Last BOM import", nil, nonNil(lastImport)),
		data.NewField("BOM format", nil, nonNil(bomFormat)),
		ageField,
		data.NewField("Last analysis", nil, nonNil(lastAnalysis)),
		data.NewField("Risk score", nil, nonNil(risk)),
		data.NewField("Critical", nil, nonNil(critical)),
		data.NewField("High", nil, nonNil(high)),
		data.NewField("Medium", nil, nonNil(medium)),
		data.NewField("Low", nil, nonNil(low)),
		data.NewField("Unassigned", nil, nonNil(unassigned)),
		data.NewField("Policy violations (fail)", nil, nonNil(fail)),
		data.NewField("Project UUID", nil, nonNil(uuid)),
	)
	frame.SetMeta(&data.FrameMeta{PreferredVisualization: data.VisTypeTable})
	return frame
}

func (d *Datasource) vulnerabilityMetrics(ctx context.Context, tr backend.TimeRange) (data.Frames, error) {
	rows, err := d.client.VulnerabilityMetrics(ctx)
	if err != nil {
		return nil, err
	}
	var times []time.Time
	var counts []float64
	for _, r := range rows {
		ts := time.Date(r.Year, time.Month(r.Month), 1, 0, 0, 0, 0, time.UTC)
		if !ts.AddDate(0, 1, 0).After(tr.From) || ts.After(tr.To) {
			continue
		}
		times = append(times, ts)
		counts = append(counts, float64(r.Count))
	}
	value := data.NewField("vulnerabilities", data.Labels{"metric": "vulnerabilities"}, nonNil(counts))
	value.Config = &data.FieldConfig{DisplayNameFromDS: "Vulnerabilities published"}
	frame := data.NewFrame("vulnerabilityMetrics", data.NewField("time", nil, nonNil(times)), value)
	frame.SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesMulti, TypeVersion: data.FrameTypeVersion{0, 1}})
	return data.Frames{frame}, nil
}
