package plugin

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"
	"github.com/WindKube/grafana-plugin-dependency-track/pkg/query"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func (d *Datasource) violations(ctx context.Context, q *query.Query) (data.Frames, error) {
	f := q.Violations
	projects, byProject, err := collectPerProject(ctx, d, q.Projects,
		func(ctx context.Context) ([]dtrack.Violation, error) {
			return d.client.Violations(ctx, dtrack.ViolationFilter{Suppressed: f.IncludeSuppressed, ShowInactive: q.Projects.IncludeInactive})
		},
		func(ctx context.Context, uuid string) ([]dtrack.Violation, error) {
			return d.client.ProjectViolations(ctx, uuid, dtrack.ViolationFilter{Suppressed: f.IncludeSuppressed})
		},
		func(v dtrack.Violation) string { return v.Project.UUID },
	)
	if err != nil {
		return nil, err
	}

	for uuid, items := range byProject {
		byProject[uuid] = slices.DeleteFunc(items, func(v dtrack.Violation) bool { return !matchViolation(v, f) })
	}
	if q.Format == query.FormatCount {
		return data.Frames{violationCounts(projects, byProject, f)}, nil
	}
	return data.Frames{violationTable(projects, byProject)}, nil
}

func matchViolation(v dtrack.Violation, f query.ViolationFilter) bool {
	if !slices.Contains(f.SelectedStates(), v.PolicyCondition.Policy.ViolationState) {
		return false
	}
	if !slices.Contains(f.SelectedRiskTypes(), v.Type) {
		return false
	}
	if len(f.Policies) > 0 && !slices.Contains(f.Policies, v.PolicyCondition.Policy.Name) {
		return false
	}
	if !f.IncludeSuppressed && v.Analysis != nil && v.Analysis.IsSuppressed {
		return false
	}
	return true
}

func violationTable(projects []dtrack.Project, byProject map[string][]dtrack.Violation) *data.Frame {
	var (
		project, version, tags, component, componentVersion, purl []string
		policy, state, riskType, condition, analysis              []string
		projectUUID, componentUUID                                []string
		suppressed                                                []bool
		occurred                                                  []time.Time
	)
	for _, p := range projects {
		labels := projectLabels(p)
		for _, v := range byProject[p.UUID] {
			c := v.PolicyCondition
			project = append(project, p.Name)
			version = append(version, p.Version)
			tags = append(tags, labels["tags"])
			component = append(component, componentName(v.Component.Group, v.Component.Name))
			componentVersion = append(componentVersion, v.Component.Version)
			purl = append(purl, v.Component.PURL)
			policy = append(policy, c.Policy.Name)
			state = append(state, c.Policy.ViolationState)
			riskType = append(riskType, v.Type)
			condition = append(condition, fmt.Sprintf("%s %s %s", c.Subject, c.Operator, c.Value))
			a, s := "NOT_SET", false
			if v.Analysis != nil {
				a, s = analysisState(v.Analysis.State), v.Analysis.IsSuppressed
			}
			analysis = append(analysis, a)
			suppressed = append(suppressed, s)
			occurred = append(occurred, v.Timestamp.Time)
			projectUUID = append(projectUUID, p.UUID)
			componentUUID = append(componentUUID, v.Component.UUID)
		}
	}
	frame := data.NewFrame("violations",
		data.NewField("Project", nil, nonNil(project)),
		data.NewField("Version", nil, nonNil(version)),
		data.NewField("Tags", nil, nonNil(tags)),
		data.NewField("Component", nil, nonNil(component)),
		data.NewField("Component version", nil, nonNil(componentVersion)),
		data.NewField("PURL", nil, nonNil(purl)),
		data.NewField("Policy", nil, nonNil(policy)),
		data.NewField("State", nil, nonNil(state)),
		data.NewField("Risk type", nil, nonNil(riskType)),
		data.NewField("Condition", nil, nonNil(condition)),
		data.NewField("Analysis", nil, nonNil(analysis)),
		data.NewField("Suppressed", nil, nonNil(suppressed)),
		data.NewField("Occurred", nil, nonNil(occurred)),
		data.NewField("Project UUID", nil, nonNil(projectUUID)),
		data.NewField("Component UUID", nil, nonNil(componentUUID)),
	)
	frame.SetMeta(&data.FrameMeta{PreferredVisualization: data.VisTypeTable})
	return frame
}

// violationCounts is zero-filled per project, state and risk type so alert instances are
// stable; with GroupByPolicy only policies that have violations appear.
func violationCounts(projects []dtrack.Project, byProject map[string][]dtrack.Violation, f query.ViolationFilter) *data.Frame {
	if f.GroupByPolicy {
		b := newCountBuilder("policy", "state", "type")
		for _, p := range projects {
			counts := map[[3]string]int64{}
			for _, v := range byProject[p.UUID] {
				counts[[3]string{v.PolicyCondition.Policy.Name, v.PolicyCondition.Policy.ViolationState, v.Type}]++
			}
			keys := make([][3]string, 0, len(counts))
			for k := range counts {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool {
				return keys[i][0]+keys[i][1]+keys[i][2] < keys[j][0]+keys[j][1]+keys[j][2]
			})
			for _, k := range keys {
				b.add(p, counts[k], k[0], k[1], k[2])
			}
		}
		return b.frame("violations")
	}

	b := newCountBuilder("state", "type")
	for _, p := range projects {
		counts := map[[2]string]int64{}
		for _, v := range byProject[p.UUID] {
			counts[[2]string{v.PolicyCondition.Policy.ViolationState, v.Type}]++
		}
		for _, state := range f.SelectedStates() {
			for _, riskType := range f.SelectedRiskTypes() {
				b.add(p, counts[[2]string{state, riskType}], state, riskType)
			}
		}
	}
	return b.frame("violations")
}
