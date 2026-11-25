package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var testRange = backend.TimeRange{
	From: time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC),
	To:   time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC),
}

func runQuery(t *testing.T, ds *Datasource, q map[string]any) backend.DataResponse {
	t.Helper()
	raw, _ := json.Marshal(q)
	resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{{RefID: "A", JSON: raw, TimeRange: testRange}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp.Responses["A"]
}

func valueField(t *testing.T, f *data.Frame) *data.Field {
	t.Helper()
	for _, field := range f.Fields {
		if field.Type().Numeric() {
			return field
		}
	}
	t.Fatalf("frame %q has no numeric field", f.Name)
	return nil
}

func assertUniqueLabels(t *testing.T, frames data.Frames) {
	t.Helper()
	seen := map[string]bool{}
	for _, f := range frames {
		key := valueField(t, f).Labels.String()
		if seen[key] {
			t.Fatalf("duplicate series labels %s", key)
		}
		seen[key] = true
	}
}

func TestPortfolioMetricsCurrent(t *testing.T) {
	res := runQuery(t, fixtureDT(t), map[string]any{"queryType": "portfolioMetrics"})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	if len(res.Frames) != 5 {
		t.Fatalf("got %d frames", len(res.Frames))
	}
	assertUniqueLabels(t, res.Frames)
	critical := res.Frames[0]
	if critical.Meta.Type != data.FrameTypeNumericMulti || critical.RefID != "A" {
		t.Fatalf("meta %+v refId %q", critical.Meta, critical.RefID)
	}
	f := valueField(t, critical)
	if v, _ := f.FloatAt(0); v != 3 {
		t.Errorf("critical = %v", v)
	}
	if f.Labels["metric"] != "critical" || f.Labels["severity"] != "critical" {
		t.Errorf("labels %v", f.Labels)
	}
}

func TestPortfolioOnlyMetric(t *testing.T) {
	ds := fixtureDT(t)
	res := runQuery(t, ds, map[string]any{"queryType": "portfolioMetrics", "metrics": []string{"projects", "inheritedRiskScore"}})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	if v, _ := valueField(t, res.Frames[0]).FloatAt(0); v != 5 {
		t.Errorf("projects = %v", v)
	}
	if _, ok := valueField(t, res.Frames[1]).Labels["severity"]; ok {
		t.Error("risk score must not have a severity label")
	}

	res = runQuery(t, ds, map[string]any{"queryType": "projectMetrics", "metrics": []string{"projects"}})
	if res.Status != backend.StatusBadRequest {
		t.Errorf("expected bad request, got %v", res.Status)
	}
}

func TestPortfolioMetricsTimeSeries(t *testing.T) {
	res := runQuery(t, fixtureDT(t), map[string]any{"queryType": "portfolioMetrics", "mode": "timeseries", "metrics": []string{"critical"}})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	f := res.Frames[0]
	if f.Meta.Type != data.FrameTypeTimeSeriesMulti || f.Fields[0].Type() != data.FieldTypeTime {
		t.Fatalf("unexpected frame %+v", f.Meta)
	}
	if f.Rows() == 0 {
		t.Fatal("no rows in range")
	}
	for i := 0; i < f.Rows(); i++ {
		ts := f.Fields[0].At(i).(time.Time)
		if ts.Before(testRange.From) || ts.After(testRange.To) {
			t.Errorf("row %d at %v is outside the time range", i, ts)
		}
	}
}

func TestProjectMetricsCurrent(t *testing.T) {
	ds := fixtureDT(t)
	res := runQuery(t, ds, map[string]any{"queryType": "projectMetrics", "metrics": []string{"critical", "policyViolationsFail"}})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	if len(res.Frames) != 10 {
		t.Fatalf("got %d frames, want 5 projects x 2 metrics", len(res.Frames))
	}
	assertUniqueLabels(t, res.Frames)

	f := valueField(t, res.Frames[0])
	want := data.Labels{"metric": "critical", "severity": "critical", "project": "inventory-service", "version": "3.1.0", "tags": "backend,staging,team-logistics"}
	if f.Labels.String() != want.String() {
		t.Errorf("labels %v, want %v", f.Labels, want)
	}
	if v, _ := f.FloatAt(0); v != 2 {
		t.Errorf("inventory-service critical = %v", v)
	}
}

func TestProjectSelectors(t *testing.T) {
	ds := fixtureDT(t)
	tests := []struct {
		name     string
		selector map[string]any
		projects int
	}{
		{"only latest", map[string]any{"mode": "all", "onlyLatest": true}, 4},
		{"name regex", map[string]any{"mode": "name", "nameRegex": "^payments"}, 2},
		{"name regex latest", map[string]any{"mode": "name", "nameRegex": "^payments", "onlyLatest": true}, 1},
		{"tag", map[string]any{"mode": "tag", "tag": "team-logistics"}, 2},
		{"uuid", map[string]any{"mode": "uuid", "uuids": []string{"193995f2-2205-4b1b-ac0b-4e84e16eda35"}}, 1},
		{"no match", map[string]any{"mode": "name", "nameRegex": "^nothing$"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := runQuery(t, ds, map[string]any{"queryType": "projectMetrics", "metrics": []string{"critical"}, "projects": tt.selector})
			if res.Error != nil {
				t.Fatal(res.Error)
			}
			got := len(res.Frames)
			if tt.projects == 0 {
				if got != 1 || len(res.Frames[0].Fields) != 0 {
					t.Fatalf("expected a single empty frame, got %d frames", got)
				}
				return
			}
			if got != tt.projects {
				t.Fatalf("got %d series, want %d", got, tt.projects)
			}
		})
	}
}

func TestProjectMetricsTimeSeries(t *testing.T) {
	res := runQuery(t, fixtureDT(t), map[string]any{
		"queryType": "projectMetrics", "mode": "timeseries", "metrics": []string{"critical"},
		"projects": map[string]any{"mode": "uuid", "uuids": []string{"193995f2-2205-4b1b-ac0b-4e84e16eda35"}},
	})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	f := res.Frames[0]
	if f.Meta.Type != data.FrameTypeTimeSeriesMulti || f.Rows() != 2 {
		t.Fatalf("type %s with %d rows", f.Meta.Type, f.Rows())
	}
	if v, _ := valueField(t, f).FloatAt(1); v != 1 {
		t.Errorf("last critical = %v", v)
	}
	if valueField(t, f).Labels["project"] != "payments-api" {
		t.Errorf("labels %v", valueField(t, f).Labels)
	}
}

func TestInvalidQueries(t *testing.T) {
	ds := fixtureDT(t)
	for _, q := range []map[string]any{
		{"queryType": "nope"},
		{"queryType": "portfolioMetrics", "mode": "nope"},
		{"queryType": "portfolioMetrics", "metrics": []string{"nope"}},
		{"queryType": "projectMetrics", "projects": map[string]any{"mode": "name", "nameRegex": "("}},
		{"queryType": "projectMetrics", "projects": map[string]any{"mode": "tag"}},
		{"queryType": "projectMetrics", "projects": map[string]any{"mode": "uuid"}},
	} {
		if res := runQuery(t, ds, q); res.Status != backend.StatusBadRequest {
			t.Errorf("%v: status %v", q, res.Status)
		}
	}
}

func TestQueryWithoutConfiguration(t *testing.T) {
	ds := &Datasource{settingsErr: errNotConfigured}
	res := runQuery(t, ds, map[string]any{})
	if res.Status != backend.StatusBadRequest {
		t.Errorf("status %v", res.Status)
	}
}

var errNotConfigured = errors.New("API key is not configured")
