package plugin

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"
	"github.com/WindKube/grafana-plugin-dependency-track/pkg/query"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type event struct {
	time    time.Time
	title   string
	text    string
	tags    []string
	project dtrack.Project
}

// events returns a frame Grafana maps to annotations by field name: time, title, text and
// comma-separated tags. Dependency-Track keeps only the last BOM import per project, so
// older imports cannot be shown.
func (d *Datasource) events(ctx context.Context, q *query.Query, tr backend.TimeRange) (data.Frames, error) {
	kinds := q.Events.SelectedKinds()
	var events []event

	if slices.Contains(kinds, query.EventBOMImport) {
		projects, err := d.resolveProjects(ctx, q.Projects)
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			if p.LastBomImport == nil || p.LastBomImport.IsZero() {
				continue
			}
			events = append(events, event{
				time:    p.LastBomImport.Time,
				title:   fmt.Sprintf("BOM imported: %s %s", p.Name, p.Version),
				text:    strings.TrimSpace("Format " + p.LastBomImportFormat),
				tags:    []string{"bom"},
				project: p,
			})
		}
	}

	if slices.Contains(kinds, query.EventFinding) {
		projects, byProject, err := d.collectFindings(ctx, q)
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			for _, fi := range byProject[p.UUID] {
				v := fi.Vulnerability
				events = append(events, event{
					time:    fi.Attribution.AttributedOn.Time,
					title:   fmt.Sprintf("%s %s in %s %s", strings.ToUpper(v.Severity), v.VulnID, p.Name, p.Version),
					text:    fmt.Sprintf("%s %s", componentName(fi.Component.Group, fi.Component.Name), fi.Component.Version),
					tags:    []string{"finding", strings.ToLower(v.Severity)},
					project: p,
				})
			}
		}
	}

	if slices.Contains(kinds, query.EventViolation) {
		projects, byProject, err := d.collectViolations(ctx, q)
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			for _, v := range byProject[p.UUID] {
				policy := v.PolicyCondition.Policy
				events = append(events, event{
					time:    v.Timestamp.Time,
					title:   fmt.Sprintf("%s policy violation in %s %s: %s", policy.ViolationState, p.Name, p.Version, policy.Name),
					text:    fmt.Sprintf("%s %s", componentName(v.Component.Group, v.Component.Name), v.Component.Version),
					tags:    []string{"violation", strings.ToLower(policy.ViolationState)},
					project: p,
				})
			}
		}
	}

	events = slices.DeleteFunc(events, func(e event) bool { return e.time.Before(tr.From) || e.time.After(tr.To) })
	slices.SortStableFunc(events, func(a, b event) int { return a.time.Compare(b.time) })
	return data.Frames{eventFrame(events)}, nil
}

func eventFrame(events []event) *data.Frame {
	var (
		times                               []time.Time
		title, text, tags, project, version []string
	)
	for _, e := range events {
		times = append(times, e.time)
		title = append(title, e.title)
		text = append(text, e.text)
		tags = append(tags, strings.Join(append([]string{"dependency-track"}, e.tags...), ","))
		project = append(project, e.project.Name)
		version = append(version, e.project.Version)
	}
	return data.NewFrame("events",
		data.NewField("time", nil, nonNil(times)),
		data.NewField("title", nil, nonNil(title)),
		data.NewField("text", nil, nonNil(text)),
		data.NewField("tags", nil, nonNil(tags)),
		data.NewField("project", nil, nonNil(project)),
		data.NewField("version", nil, nonNil(version)),
	)
}
