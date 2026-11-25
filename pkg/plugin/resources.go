package plugin

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sort"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"
	"github.com/WindKube/grafana-plugin-dependency-track/pkg/query"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

var Classifiers = []string{
	"APPLICATION", "FRAMEWORK", "LIBRARY", "CONTAINER", "OPERATING_SYSTEM", "DEVICE", "FIRMWARE",
	"FILE", "PLATFORM", "DEVICE_DRIVER", "MACHINE_LEARNING_MODEL", "DATA",
}

type projectOption struct {
	UUID     string   `json:"uuid"`
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	IsLatest bool     `json:"isLatest"`
	Tags     []string `json:"tags"`
}

func (d *Datasource) newResourceHandler() backend.CallResourceHandler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /projects", d.handleProjects)
	mux.HandleFunc("GET /versions", d.handleVersions)
	mux.HandleFunc("GET /tags", d.handleTags)
	mux.HandleFunc("GET /classifiers", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, Classifiers) })
	mux.HandleFunc("GET /policies", d.handlePolicies)
	return httpadapter.New(mux)
}

func (d *Datasource) handleProjects(w http.ResponseWriter, r *http.Request) {
	if !d.ready(w) {
		return
	}
	sel := query.ProjectSelector{Mode: query.SelectAll, OnlyLatest: r.URL.Query().Get("onlyLatest") == "true"}
	if name := r.URL.Query().Get("name"); name != "" {
		sel = query.ProjectSelector{Mode: query.SelectName, NameRegex: "^" + regexp.QuoteMeta(name) + "$", OnlyLatest: sel.OnlyLatest}
	}
	projects, err := d.resolveProjects(r.Context(), sel)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]projectOption, 0, len(projects))
	for _, p := range projects {
		out = append(out, projectOption{UUID: p.UUID, Name: p.Name, Version: p.Version, IsLatest: p.IsLatest, Tags: p.TagNames()})
	}
	writeJSON(w, out)
}

func (d *Datasource) handleVersions(w http.ResponseWriter, r *http.Request) {
	if !d.ready(w) {
		return
	}
	name := r.URL.Query().Get("project")
	if name == "" {
		http.Error(w, "project is required", http.StatusBadRequest)
		return
	}
	projects, err := d.resolveProjects(r.Context(), query.ProjectSelector{Mode: query.SelectName, NameRegex: "^" + regexp.QuoteMeta(name) + "$"})
	if err != nil {
		writeError(w, err)
		return
	}
	versions := make([]string, 0, len(projects))
	for _, p := range projects {
		versions = append(versions, p.Version)
	}
	writeJSON(w, versions)
}

func (d *Datasource) handleTags(w http.ResponseWriter, r *http.Request) {
	if !d.ready(w) {
		return
	}
	tags, err := d.client.Tags(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}
	sort.Strings(names)
	writeJSON(w, names)
}

// handlePolicies lists policy names. Listing policies needs POLICY_MANAGEMENT, so a
// view-only key falls back to the policies referenced by violations.
func (d *Datasource) handlePolicies(w http.ResponseWriter, r *http.Request) {
	if !d.ready(w) {
		return
	}
	set := map[string]bool{}
	policies, err := d.client.Policies(r.Context())
	switch {
	case err == nil:
		for _, p := range policies {
			set[p.Name] = true
		}
	case dtrack.IsStatus(err, http.StatusForbidden):
		violations, verr := d.client.Violations(r.Context(), dtrack.ViolationFilter{})
		if verr != nil {
			writeError(w, verr)
			return
		}
		for _, v := range violations {
			set[v.PolicyCondition.Policy.Name] = true
		}
	default:
		writeError(w, err)
		return
	}
	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	writeJSON(w, names)
}

func (d *Datasource) ready(w http.ResponseWriter) bool {
	if d.settingsErr != nil {
		http.Error(w, d.settingsErr.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	var apiErr *dtrack.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode < 500 {
		status = apiErr.StatusCode
	}
	http.Error(w, err.Error(), status)
}
