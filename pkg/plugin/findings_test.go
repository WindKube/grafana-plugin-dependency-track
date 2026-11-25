package plugin

import (
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// countRows returns "project version dims..." -> count for a numeric-long count frame.
func countRows(t *testing.T, f *data.Frame) map[string]int64 {
	t.Helper()
	if f.Meta == nil || f.Meta.Type != data.FrameTypeNumericLong {
		t.Fatalf("expected numeric-long frame, got %+v", f.Meta)
	}
	numeric := 0
	for _, field := range f.Fields {
		if field.Type().Numeric() {
			numeric++
		}
	}
	if numeric != 1 {
		t.Fatalf("count frames must have exactly one numeric column, got %d", numeric)
	}
	out := map[string]int64{}
	for i := 0; i < f.Rows(); i++ {
		var key []string
		for _, field := range f.Fields {
			if field.Type() == data.FieldTypeString && field.Name != "tags" {
				key = append(key, field.At(i).(string))
			}
		}
		k := strings.Join(key, " ")
		if _, dup := out[k]; dup {
			t.Fatalf("duplicate row %q", k)
		}
		out[k] = f.Fields[len(f.Fields)-1].At(i).(int64)
	}
	return out
}

func TestFindingsTable(t *testing.T) {
	ds := fixtureDT(t)
	res := runQuery(t, ds, map[string]any{"queryType": "findings"})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	f := res.Frames[0]
	if f.Rows() != 13 {
		t.Fatalf("got %d findings", f.Rows())
	}
	if f.Fields[0].Name != "Project" || f.Meta.PreferredVisualization != data.VisTypeTable {
		t.Errorf("unexpected frame %v %+v", f.Fields[0].Name, f.Meta)
	}

	res = runQuery(t, ds, map[string]any{"queryType": "findings", "findings": map[string]any{"severities": []string{"CRITICAL"}, "cvssMin": 9.9}})
	if rows := res.Frames[0].Rows(); rows != 1 {
		t.Errorf("critical with CVSS >= 9.9: %d rows", rows)
	}
}

func TestFindingsCount(t *testing.T) {
	for _, mode := range []string{"all", "name"} {
		t.Run(mode, func(t *testing.T) {
			ds := fixtureDT(t)
			res := runQuery(t, ds, map[string]any{
				"queryType": "findings", "format": "count",
				"projects": map[string]any{"mode": mode, "nameRegex": ".*", "onlyLatest": true},
			})
			if res.Error != nil {
				t.Fatal(res.Error)
			}
			rows := countRows(t, res.Frames[0])
			if len(rows) != 4*5 {
				t.Fatalf("got %d rows, want 4 projects x 5 severities", len(rows))
			}
			want := map[string]int64{
				"inventory-service 3.1.0 critical": 2,
				"inventory-service 3.1.0 medium":   1,
				"payments-api 1.5.0 critical":      0,
				"web-frontend 2.3.0 high":          1,
				"report-worker 0.9.2 low":          1,
			}
			for k, v := range want {
				if rows[k] != v {
					t.Errorf("%s = %d, want %d", k, rows[k], v)
				}
			}
		})
	}
}

func TestViolations(t *testing.T) {
	ds := fixtureDT(t)

	res := runQuery(t, ds, map[string]any{"queryType": "violations", "violations": map[string]any{"states": []string{"FAIL"}}})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	if rows := res.Frames[0].Rows(); rows != 2 {
		t.Errorf("FAIL violations: %d", rows)
	}

	res = runQuery(t, ds, map[string]any{"queryType": "violations", "format": "count", "violations": map[string]any{"riskTypes": []string{"OPERATIONAL"}}})
	rows := countRows(t, res.Frames[0])
	if len(rows) != 5*3 {
		t.Fatalf("got %d rows, want 5 projects x 3 states", len(rows))
	}
	for k, v := range map[string]int64{
		"web-frontend 2.3.0 FAIL OPERATIONAL":      2,
		"inventory-service 3.1.0 WARN OPERATIONAL": 1,
		"report-worker 0.9.2 INFO OPERATIONAL":     5,
		"payments-api 1.4.0 FAIL OPERATIONAL":      0,
	} {
		if rows[k] != v {
			t.Errorf("%s = %d, want %d", k, rows[k], v)
		}
	}

	res = runQuery(t, ds, map[string]any{
		"queryType": "violations", "format": "count",
		"projects":   map[string]any{"mode": "tag", "tag": "frontend"},
		"violations": map[string]any{"groupByPolicy": true},
	})
	rows = countRows(t, res.Frames[0])
	if fmt.Sprint(rows) != "map[web-frontend 2.3.0 Banned packages FAIL OPERATIONAL:2]" {
		t.Errorf("grouped by policy: %v", rows)
	}
}

func TestFindingFilterValidation(t *testing.T) {
	ds := fixtureDT(t)
	for _, q := range []map[string]any{
		{"queryType": "findings", "format": "nope"},
		{"queryType": "findings", "findings": map[string]any{"severities": []string{"SEVERE"}}},
		{"queryType": "violations", "violations": map[string]any{"states": []string{"ERROR"}}},
	} {
		if res := runQuery(t, ds, q); res.Status != backend.StatusBadRequest {
			t.Errorf("%v: status %v", q, res.Status)
		}
	}
}
