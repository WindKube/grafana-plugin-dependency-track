package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func callResource(t *testing.T, ds *Datasource, path string, out any) int {
	t.Helper()
	var resp *backend.CallResourceResponse
	route, _, _ := strings.Cut(path, "?")
	err := ds.CallResource(context.Background(), &backend.CallResourceRequest{Method: http.MethodGet, Path: route, URL: path},
		backend.CallResourceResponseSenderFunc(func(r *backend.CallResourceResponse) error {
			resp = r
			return nil
		}))
	if err != nil {
		t.Fatal(err)
	}
	if out != nil && resp.Status == http.StatusOK {
		if err := json.Unmarshal(resp.Body, out); err != nil {
			t.Fatalf("%s: %v: %s", path, err, resp.Body)
		}
	}
	return resp.Status
}

func TestResources(t *testing.T) {
	ds := fixtureDT(t)

	var projects []projectOption
	if status := callResource(t, ds, "projects", &projects); status != http.StatusOK || len(projects) != 5 {
		t.Fatalf("projects: %d %v", status, projects)
	}
	callResource(t, ds, "projects?onlyLatest=true", &projects)
	if len(projects) != 4 {
		t.Errorf("latest projects: %d", len(projects))
	}

	var versions []string
	callResource(t, ds, "versions?project=payments-api", &versions)
	if len(versions) != 2 || versions[0] != "1.4.0" || versions[1] != "1.5.0" {
		t.Errorf("versions: %v", versions)
	}
	if status := callResource(t, ds, "versions", nil); status != http.StatusBadRequest {
		t.Errorf("versions without project: %d", status)
	}

	var tags []string
	callResource(t, ds, "tags", &tags)
	if len(tags) != 8 || tags[0] != "backend" {
		t.Errorf("tags: %v", tags)
	}

	var classifiers []string
	callResource(t, ds, "classifiers", &classifiers)
	if len(classifiers) == 0 || classifiers[0] != "APPLICATION" {
		t.Errorf("classifiers: %v", classifiers)
	}

	var policies []string
	callResource(t, ds, "policies", &policies)
	if len(policies) != 3 || policies[0] != "Banned packages" {
		t.Errorf("policies: %v", policies)
	}

	if status := callResource(t, ds, "nope", nil); status != http.StatusNotFound {
		t.Errorf("unknown route: %d", status)
	}
}

func TestResourcesWithoutConfiguration(t *testing.T) {
	ds := &Datasource{settingsErr: errNotConfigured}
	ds.CallResourceHandler = ds.newResourceHandler()
	if status := callResource(t, ds, "tags", nil); status != http.StatusBadRequest {
		t.Errorf("status %d", status)
	}
}
