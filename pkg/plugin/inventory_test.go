package plugin

import (
	"math"
	"testing"
	"time"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Two days after inventory-service's recorded BOM import.
var fixtureNow = time.UnixMilli(1795611053247).Add(48 * time.Hour)

func fixtureAt(t *testing.T) *Datasource {
	ds := fixtureDT(t)
	ds.now = func() time.Time { return fixtureNow }
	return ds
}

func TestProjectInventoryTable(t *testing.T) {
	res := runQuery(t, fixtureAt(t), map[string]any{"queryType": "projects"})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	f := res.Frames[0]
	if f.Rows() != 5 {
		t.Fatalf("got %d rows", f.Rows())
	}
	idx := map[string]int{}
	for i, field := range f.Fields {
		idx[field.Name] = i
	}
	for _, name := range []string{"Project", "Version", "Last BOM import", "BOM age", "Critical", "Project UUID"} {
		if _, ok := idx[name]; !ok {
			t.Fatalf("missing column %q", name)
		}
	}
	for row := 0; row < f.Rows(); row++ {
		if f.Fields[idx["Project"]].At(row) != "inventory-service" {
			continue
		}
		if got := *f.Fields[idx["Critical"]].At(row).(*int64); got != 2 {
			t.Fatalf("inventory-service critical = %d", got)
		}
		age := *f.Fields[idx["BOM age"]].At(row).(*float64)
		if age != 48*3600 {
			t.Fatalf("inventory-service BOM age = %v", age)
		}
		return
	}
	t.Fatal("inventory-service not found")
}

func TestProjectBOMAge(t *testing.T) {
	res := runQuery(t, fixtureAt(t), map[string]any{
		"queryType": "projects",
		"format":    "bomAge",
		"projects":  map[string]any{"mode": "all", "onlyLatest": true},
	})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	f := res.Frames[0]
	if f.Meta.Type != data.FrameTypeNumericLong {
		t.Fatalf("frame type %s", f.Meta.Type)
	}
	if f.Rows() != 4 {
		t.Fatalf("got %d rows", f.Rows())
	}
	numeric := 0
	for _, field := range f.Fields {
		if field.Type().Numeric() {
			numeric++
		}
	}
	if numeric != 1 {
		t.Fatalf("got %d numeric columns", numeric)
	}
	seen := map[string]bool{}
	for row := 0; row < f.Rows(); row++ {
		key := f.Fields[0].At(row).(string) + "@" + f.Fields[1].At(row).(string)
		if seen[key] {
			t.Fatalf("duplicate row %s", key)
		}
		seen[key] = true
		if math.IsInf(f.Fields[4].At(row).(float64), 1) {
			t.Fatalf("%s has no BOM age", key)
		}
	}
	if !seen["payments-api@1.5.0"] || seen["payments-api@1.4.0"] {
		t.Fatalf("unexpected rows %v", seen)
	}
}

func TestBOMAgeWithoutImport(t *testing.T) {
	if v := bomAge(dtrack.Project{Name: "empty"}, fixtureNow); v != nil {
		t.Fatalf("inventory age should be null, got %v", *v)
	}
	f := bomAgeFrame([]dtrack.Project{{Name: "empty"}}, fixtureNow)
	if v := f.Fields[4].At(0).(float64); !math.IsInf(v, 1) {
		t.Fatalf("alerting age = %v, want +Inf so any stale threshold fires", v)
	}
}

func TestVulnerabilityMetrics(t *testing.T) {
	res := runQuery(t, fixtureDT(t), map[string]any{"queryType": "vulnerabilityMetrics"})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	f := res.Frames[0]
	if f.Meta.Type != data.FrameTypeTimeSeriesMulti {
		t.Fatalf("frame type %s", f.Meta.Type)
	}
	if f.Rows() != 2 {
		t.Fatalf("got %d rows, want October and November", f.Rows())
	}
	if got := f.Fields[1].At(0).(float64); got != 3500 {
		t.Fatalf("october count = %v", got)
	}
}
