package plugin

import (
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func fieldByName(t *testing.T, f *data.Frame, name string) *data.Field {
	t.Helper()
	field, _ := f.FieldByName(name)
	if field == nil {
		t.Fatalf("frame %q has no field %q", f.Name, name)
	}
	return field
}

func TestEventsAllKinds(t *testing.T) {
	res := runQuery(t, fixtureDT(t), map[string]any{"queryType": "events"})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	f := res.Frames[0]
	if f.Rows() != 5+13+8 {
		t.Fatalf("got %d events", f.Rows())
	}
	times := fieldByName(t, f, "time")
	for i := 1; i < f.Rows(); i++ {
		if times.At(i).(time.Time).Before(times.At(i - 1).(time.Time)) {
			t.Fatal("events are not sorted by time")
		}
	}
	kinds := map[string]int{}
	tags := fieldByName(t, f, "tags")
	for i := 0; i < f.Rows(); i++ {
		parts := strings.Split(tags.At(i).(string), ",")
		if parts[0] != "dependency-track" {
			t.Fatalf("tags %q", tags.At(i))
		}
		kinds[parts[1]]++
	}
	if kinds["bom"] != 5 || kinds["finding"] != 13 || kinds["violation"] != 8 {
		t.Fatalf("kinds %v", kinds)
	}
}

func TestEventsUseFindingFilter(t *testing.T) {
	res := runQuery(t, fixtureDT(t), map[string]any{
		"queryType": "events",
		"events":    map[string]any{"kinds": []string{"finding"}},
		"findings":  map[string]any{"severities": []string{"CRITICAL"}},
	})
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	f := res.Frames[0]
	if f.Rows() != 3 {
		t.Fatalf("got %d critical finding events", f.Rows())
	}
	if title := fieldByName(t, f, "title").At(0).(string); !strings.HasPrefix(title, "CRITICAL ") {
		t.Fatalf("title %q", title)
	}
}

func TestEventsOutsideTimeRange(t *testing.T) {
	ds := fixtureDT(t)
	frames, err := ds.events(t.Context(), mustQuery(t, `{"queryType":"events"}`), backend.TimeRange{
		From: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if frames[0].Rows() != 0 {
		t.Fatalf("got %d events", frames[0].Rows())
	}
}

func TestEventsRejectUnknownKind(t *testing.T) {
	res := runQuery(t, fixtureDT(t), map[string]any{"queryType": "events", "events": map[string]any{"kinds": []string{"nope"}}})
	if res.Error == nil {
		t.Fatal("expected an error")
	}
}
