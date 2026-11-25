package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"
	"github.com/WindKube/grafana-plugin-dependency-track/pkg/models"
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
	httpClient, err := httpclient.New(opts)
	if err != nil {
		return nil, fmt.Errorf("http client: %w", err)
	}
	ds.client, err = dtrack.New(dtrack.Options{
		BaseURL:    settings.URL,
		APIKey:     settings.APIKey,
		HTTPClient: httpClient,
	})
	if err != nil {
		ds.settingsErr = err
	}
	return ds, nil
}

func (d *Datasource) Dispose() {}

func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()
	for _, q := range req.Queries {
		response.Responses[q.RefID] = d.query(ctx, req.PluginContext, q)
	}
	return response, nil
}

type queryModel struct{}

func (d *Datasource) query(_ context.Context, _ backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	var qm queryModel
	if len(query.JSON) > 0 {
		if err := json.Unmarshal(query.JSON, &qm); err != nil {
			return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err))
		}
	}

	frame := data.NewFrame("response",
		data.NewField("time", nil, []time.Time{query.TimeRange.From, query.TimeRange.To}),
		data.NewField("values", nil, []int64{10, 20}),
	)
	return backend.DataResponse{Frames: data.Frames{frame}}
}
