package plugin

import (
	"encoding/json"
	"testing"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/query"
)

func mustQuery(t *testing.T, raw string) *query.Query {
	t.Helper()
	q, err := query.Parse(json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func linkURLs(t *testing.T, res map[string]any, uiURL, field string) []string {
	t.Helper()
	ds := fixtureDT(t)
	ds.settings.UIURL = uiURL
	r := runQuery(t, ds, res)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	f, _ := r.Frames[0].FieldByName(field)
	if f == nil || f.Config == nil {
		return nil
	}
	var urls []string
	for _, l := range f.Config.Links {
		urls = append(urls, l.URL)
	}
	return urls
}

func TestFindingTableLinks(t *testing.T) {
	q := map[string]any{"queryType": "findings"}
	got := linkURLs(t, q, "http://dt.example/", "Vulnerability")
	want := []string{
		`http://dt.example/projects/${__data.fields["Project UUID"]}/findings/${__data.fields["Component UUID"]}/${__data.fields["Vulnerability UUID"]}`,
		`http://dt.example/vulnerabilities/${__data.fields["Source"]}/${__value.raw}`,
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("links %v", got)
	}
	if got := linkURLs(t, q, "http://dt.example", "Project"); len(got) != 1 {
		t.Fatalf("project links %v", got)
	}
}

func TestViolationAndInventoryLinks(t *testing.T) {
	if got := linkURLs(t, map[string]any{"queryType": "violations"}, "http://dt.example", "Policy"); len(got) != 1 {
		t.Fatalf("policy links %v", got)
	}
	if got := linkURLs(t, map[string]any{"queryType": "projects"}, "http://dt.example", "Project"); len(got) != 1 {
		t.Fatalf("inventory links %v", got)
	}
}

func TestNoLinksWithoutUIURL(t *testing.T) {
	if got := linkURLs(t, map[string]any{"queryType": "findings"}, "", "Project"); got != nil {
		t.Fatalf("links %v", got)
	}
}

func TestNoLinksOnCountFrames(t *testing.T) {
	ds := fixtureDT(t)
	ds.settings.UIURL = "http://dt.example"
	r := runQuery(t, ds, map[string]any{"queryType": "findings", "format": "count"})
	for _, f := range r.Frames[0].Fields {
		if f.Config != nil && len(f.Config.Links) > 0 {
			t.Fatalf("field %q has links", f.Name)
		}
	}
}
