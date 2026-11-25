package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type healthDetails struct {
	Version            string   `json:"version,omitempty"`
	Team               string   `json:"team,omitempty"`
	Permissions        []string `json:"permissions,omitempty"`
	MissingPermissions []string `json:"missingPermissions,omitempty"`
}

func (d *Datasource) CheckHealth(ctx context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if d.settingsErr != nil {
		return healthError(d.settingsErr.Error(), nil), nil
	}

	details := &healthDetails{}
	version, err := d.client.Version(ctx)
	if err != nil {
		return healthError(fmt.Sprintf("Cannot reach Dependency-Track at %s: %v", d.client.BaseURL(), err), nil), nil
	}
	details.Version = version.Version
	major, err := dtrack.MajorVersion(version.Version)
	if err != nil {
		return healthError(err.Error(), details), nil
	}
	if major < dtrack.MinSupportedMajor {
		return healthError(fmt.Sprintf("Dependency-Track %s is not supported. This plugin requires Dependency-Track %d.0 or later.", version.Version, dtrack.MinSupportedMajor), details), nil
	}

	team, err := d.client.TeamSelf(ctx)
	switch {
	case dtrack.IsStatus(err, http.StatusUnauthorized), dtrack.IsStatus(err, http.StatusForbidden):
		return healthError("Dependency-Track rejected the API key. Check that the key exists and is not expired.", details), nil
	case err != nil:
		return healthError(fmt.Sprintf("Could not verify the API key: %v", err), details), nil
	}

	details.Team = team.Name
	for _, p := range team.Permissions {
		details.Permissions = append(details.Permissions, p.Name)
	}
	if missing := team.MissingPermissions(); len(missing) > 0 {
		details.MissingPermissions = missing
		return healthError(fmt.Sprintf("The API key of team %q is missing permissions: %s", team.Name, strings.Join(missing, ", ")), details), nil
	}

	return &backend.CheckHealthResult{
		Status:      backend.HealthStatusOk,
		Message:     fmt.Sprintf("Connected to Dependency-Track %s as team %q", version.Version, team.Name),
		JSONDetails: mustJSON(details),
	}, nil
}

func healthError(msg string, details *healthDetails) *backend.CheckHealthResult {
	res := &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: msg}
	if details != nil {
		res.JSONDetails = mustJSON(details)
	}
	return res
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
