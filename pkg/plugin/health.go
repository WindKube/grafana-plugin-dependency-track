package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

const healthTimeout = 15 * time.Second

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
	// Save & test should answer quickly, so no retries and a short deadline.
	ctx, cancel := context.WithTimeout(dtrack.WithoutRetries(ctx), healthTimeout)
	defer cancel()
	logger := log.DefaultLogger.FromContext(ctx)

	details := &healthDetails{}
	version, err := d.client.Version(ctx)
	if err != nil {
		logger.Warn("Health check: version request failed", "url", d.client.BaseURL(), "error", err)
		var apiErr *dtrack.APIError
		if errors.As(err, &apiErr) || errors.Is(err, dtrack.ErrUnexpectedResponse) {
			return healthError(fmt.Sprintf("%s does not look like a Dependency-Track API server (%s from /api/version). Check that the URL points to the API server, not the frontend.", d.client.BaseURL(), describe(err)), nil), nil
		}
		return healthError(fmt.Sprintf("Cannot reach Dependency-Track at %s: %s", d.client.BaseURL(), describe(err)), nil), nil
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
		logger.Warn("Health check: team request failed", "error", err)
		return healthError(fmt.Sprintf("Could not verify the API key: %s", describe(err)), details), nil
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

// describe summarises an error for the health alert without the response body, which can be a
// large HTML page from a proxy; the full error is logged.
func describe(err error) string {
	var apiErr *dtrack.APIError
	switch {
	case errors.As(err, &apiErr):
		return fmt.Sprintf("HTTP %d %s", apiErr.StatusCode, http.StatusText(apiErr.StatusCode))
	case errors.Is(err, dtrack.ErrUnexpectedResponse):
		return "a response that is not JSON"
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Sprintf("no response within %s", healthTimeout)
	default:
		return err.Error()
	}
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
