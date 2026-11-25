package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func newTestDatasource(t *testing.T, h http.Handler, apiKey string) *Datasource {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	inst, err := NewDatasource(context.Background(), backend.DataSourceInstanceSettings{
		URL:                     srv.URL,
		JSONData:                []byte(`{}`),
		DecryptedSecureJSONData: map[string]string{"apiKey": apiKey},
	})
	if err != nil {
		t.Fatal(err)
	}
	return inst.(*Datasource)
}

func fakeDT(version string, teamStatus int, permissions ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"application": "Dependency-Track", "version": version})
		case "/api/v1/team/self":
			if r.Header.Get("X-Api-Key") != "odt_good" || teamStatus != http.StatusOK {
				w.WriteHeader(teamStatus)
				return
			}
			perms := []map[string]string{}
			for _, p := range permissions {
				perms = append(perms, map[string]string{"name": p})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "Grafana", "permissions": perms})
		default:
			http.NotFound(w, r)
		}
	})
}

func TestCheckHealth(t *testing.T) {
	all := []string{"VIEW_PORTFOLIO", "VIEW_VULNERABILITY", "VIEW_POLICY_VIOLATION"}
	tests := []struct {
		name       string
		handler    http.Handler
		apiKey     string
		wantStatus backend.HealthStatus
		wantMsg    string
	}{
		{"ok", fakeDT("5.2.2", http.StatusOK, all...), "odt_good", backend.HealthStatusOk, `Connected to Dependency-Track 5.2.2 as team "Grafana"`},
		{"v4 rejected", fakeDT("4.14.7", http.StatusOK, all...), "odt_good", backend.HealthStatusError, "4.14.7 is not supported"},
		{"bad key", fakeDT("5.2.2", http.StatusUnauthorized), "odt_bad", backend.HealthStatusError, "rejected the API key"},
		{"forbidden key", fakeDT("5.2.2", http.StatusForbidden), "odt_bad", backend.HealthStatusError, "rejected the API key"},
		{"missing permissions", fakeDT("5.2.2", http.StatusOK, "VIEW_PORTFOLIO"), "odt_good", backend.HealthStatusError, "missing permissions: VIEW_VULNERABILITY, VIEW_POLICY_VIOLATION"},
		{"team endpoint error", fakeDT("5.2.2", http.StatusInternalServerError), "odt_good", backend.HealthStatusError, "Could not verify the API key: HTTP 500 Internal Server Error"},
		{"not a DT API", http.NotFoundHandler(), "odt_good", backend.HealthStatusError, "does not look like a Dependency-Track API server (HTTP 404 Not Found"},
		{"frontend URL", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("<!DOCTYPE html><html><body>Dependency-Track</body></html>"))
		}), "odt_good", backend.HealthStatusError, "not the frontend"},
		{"no key", fakeDT("5.2.2", http.StatusOK, all...), "", backend.HealthStatusError, "API key is not configured"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := newTestDatasource(t, tt.handler, tt.apiKey)
			res, err := ds.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
			if err != nil {
				t.Fatal(err)
			}
			if res.Status != tt.wantStatus || !strings.Contains(res.Message, tt.wantMsg) {
				t.Fatalf("got %v %q, want %v containing %q", res.Status, res.Message, tt.wantStatus, tt.wantMsg)
			}
		})
	}
}

func TestCheckHealthUnreachableFailsFast(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	inst, err := NewDatasource(context.Background(), backend.DataSourceInstanceSettings{
		URL:                     url,
		DecryptedSecureJSONData: map[string]string{"apiKey": "odt_good"},
	})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	res, _ := inst.(*Datasource).CheckHealth(context.Background(), &backend.CheckHealthRequest{})
	if res.Status != backend.HealthStatusError || !strings.Contains(res.Message, "Cannot reach Dependency-Track") {
		t.Fatalf("got %q", res.Message)
	}
	// Retries would add at least 500ms of backoff.
	if elapsed := time.Since(start); elapsed > 400*time.Millisecond {
		t.Fatalf("health check took %v; it must not retry", elapsed)
	}
}

func TestCheckHealthHidesResponseBody(t *testing.T) {
	body := strings.Repeat("<p>proxy error page</p>", 100)
	ds := newTestDatasource(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, body, http.StatusBadGateway)
	}), "odt_good")
	res, _ := ds.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
	if strings.Contains(res.Message, "proxy error page") || !strings.Contains(res.Message, "HTTP 502 Bad Gateway") {
		t.Fatalf("got %q", res.Message)
	}
}

func TestDisposeClosesIdleConnections(t *testing.T) {
	ds := newTestDatasource(t, fakeDT("5.2.2", http.StatusOK), "odt_good")
	if ds.httpClient == nil {
		t.Fatal("http client is not kept for Dispose")
	}
	ds.Dispose()
}

func TestCheckHealthWithoutURL(t *testing.T) {
	inst, err := NewDatasource(context.Background(), backend.DataSourceInstanceSettings{DecryptedSecureJSONData: map[string]string{"apiKey": "k"}})
	if err != nil {
		t.Fatal(err)
	}
	res, _ := inst.(*Datasource).CheckHealth(context.Background(), &backend.CheckHealthRequest{})
	if res.Status != backend.HealthStatusError || !strings.Contains(res.Message, "URL is not configured") {
		t.Fatalf("got %q", res.Message)
	}
}
