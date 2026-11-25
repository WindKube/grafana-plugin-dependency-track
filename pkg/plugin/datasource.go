package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"
	"github.com/WindKube/grafana-plugin-dependency-track/pkg/models"
	"github.com/WindKube/grafana-plugin-dependency-track/pkg/query"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

type Datasource struct {
	settings    *models.Settings
	settingsErr error
	client      *dtrack.Client
	httpClient  *http.Client
}

func NewDatasource(ctx context.Context, s backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	settings, err := models.LoadSettings(s)
	if err != nil {
		return nil, err
	}
	ds := &Datasource{settings: settings}
	if ds.settingsErr = settings.Validate(); ds.settingsErr != nil {
		return ds, nil
	}

	opts, err := s.HTTPClientOptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("http client options: %w", err)
	}
	if !settings.CustomTimeout && opts.Timeouts != nil {
		opts.Timeouts.Timeout = dtrack.DefaultTimeout
	}
	if ds.httpClient, err = httpclient.New(opts); err != nil {
		return nil, fmt.Errorf("http client: %w", err)
	}
	ds.client, err = dtrack.New(dtrack.Options{
		BaseURL:    settings.URL,
		APIKey:     settings.APIKey,
		HTTPClient: ds.httpClient,
	})
	if err != nil {
		ds.settingsErr = err
	}
	return ds, nil
}

// Dispose runs when the settings change and a new instance replaces this one.
func (d *Datasource) Dispose() {
	if d.httpClient != nil {
		d.httpClient.CloseIdleConnections()
	}
}

func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()
	for _, q := range req.Queries {
		response.Responses[q.RefID] = d.query(ctx, q)
	}
	return response, nil
}

func (d *Datasource) query(ctx context.Context, dq backend.DataQuery) backend.DataResponse {
	if d.settingsErr != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, d.settingsErr.Error())
	}
	q, err := query.Parse(dq.JSON)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, err.Error())
	}

	var frames data.Frames
	switch q.QueryType {
	case query.TypePortfolioMetrics:
		frames, err = d.portfolioMetrics(ctx, q, dq.TimeRange)
	case query.TypeProjectMetrics:
		frames, err = d.projectMetrics(ctx, q, dq.TimeRange)
	}
	if err != nil {
		return errorResponse(err)
	}
	for _, f := range frames {
		f.RefID = dq.RefID
	}
	return backend.DataResponse{Frames: frames}
}

func errorResponse(err error) backend.DataResponse {
	status := backend.StatusInternal
	var apiErr *dtrack.APIError
	switch {
	case errors.As(err, &apiErr):
		status = backend.Status(apiErr.StatusCode)
	case errors.Is(err, context.DeadlineExceeded):
		status = backend.StatusTimeout
	}
	return backend.ErrDataResponseWithSource(status, backend.ErrorSourceDownstream, err.Error())
}
