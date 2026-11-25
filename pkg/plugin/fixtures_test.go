package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("../dtrack/testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// fixtureDT serves the recorded Dependency-Track 5.2.2 responses.
func fixtureDT(t *testing.T) *Datasource {
	t.Helper()
	var projects []map[string]any
	if err := json.Unmarshal(readFixture(t, "projects"), &projects); err != nil {
		t.Fatal(err)
	}
	writeList := func(w http.ResponseWriter, items []map[string]any) {
		w.Header().Set("X-Total-Count", strconv.Itoa(len(items)))
		_ = json.NewEncoder(w).Encode(items)
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/api")
		switch {
		case p == "/v1/project":
			writeList(w, projects)
		case strings.HasPrefix(p, "/v1/project/tag/"):
			tag := strings.TrimPrefix(p, "/v1/project/tag/")
			var out []map[string]any
			for _, pr := range projects {
				for _, tg := range pr["tags"].([]any) {
					if tg.(map[string]any)["name"] == tag {
						out = append(out, pr)
					}
				}
			}
			writeList(w, out)
		case strings.HasPrefix(p, "/v1/project/"):
			uuid := strings.TrimPrefix(p, "/v1/project/")
			for _, pr := range projects {
				if pr["uuid"] == uuid {
					_ = json.NewEncoder(w).Encode(pr)
					return
				}
			}
			http.NotFound(w, r)
		case p == "/v1/metrics/portfolio/current":
			_, _ = w.Write(readFixture(t, "portfolio_current"))
		case strings.HasPrefix(p, "/v1/metrics/portfolio/since/"):
			_, _ = w.Write(readFixture(t, "portfolio_days"))
		case strings.HasPrefix(p, "/v1/metrics/project/") && strings.Contains(p, "/since/"):
			_, _ = w.Write(readFixture(t, "project_days"))
		case strings.HasPrefix(p, "/v1/metrics/project/") && strings.HasSuffix(p, "/current"):
			_, _ = w.Write(readFixture(t, "project_current"))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	inst, err := NewDatasource(context.Background(), backend.DataSourceInstanceSettings{
		URL:                     srv.URL,
		DecryptedSecureJSONData: map[string]string{"apiKey": "odt_test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return inst.(*Datasource)
}
