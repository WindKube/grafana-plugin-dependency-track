package plugin

import (
	"context"
	"fmt"
	"time"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/dtrack"
	"github.com/WindKube/grafana-plugin-dependency-track/pkg/query"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"golang.org/x/sync/errgroup"
)

func (d *Datasource) portfolioMetrics(ctx context.Context, q *query.Query, tr backend.TimeRange) (data.Frames, error) {
	if q.Mode == query.ModeCurrent {
		m, err := d.client.PortfolioMetricsCurrent(ctx)
		if err != nil {
			return nil, err
		}
		frames := data.Frames{}
		for _, name := range q.Metrics {
			metric := query.Metrics[name]
			frames = append(frames, numericFrame(metric, nil, metric.Display, metric.Value(*m)))
		}
		return frames, nil
	}

	rows, err := d.client.PortfolioMetricsSince(ctx, tr.From)
	if err != nil {
		return nil, err
	}
	frames := data.Frames{}
	for _, name := range q.Metrics {
		metric := query.Metrics[name]
		frames = append(frames, timeSeriesFrame(metric, nil, metric.Display, rows, tr))
	}
	return frames, nil
}

func (d *Datasource) projectMetrics(ctx context.Context, q *query.Query, tr backend.TimeRange) (data.Frames, error) {
	projects, err := d.resolveProjects(ctx, q.Projects)
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return emptyFrames(q.Mode), nil
	}

	series := make([][]dtrack.PortfolioMetrics, len(projects))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)
	for i, p := range projects {
		g.Go(func() error {
			if q.Mode == query.ModeCurrent {
				if p.Metrics != nil {
					series[i] = []dtrack.PortfolioMetrics{{Metrics: p.Metrics.Metrics}}
					return nil
				}
				m, err := d.client.ProjectMetricsCurrent(gctx, p.UUID)
				if err != nil {
					return fmt.Errorf("metrics for %s %s: %w", p.Name, p.Version, err)
				}
				series[i] = []dtrack.PortfolioMetrics{{Metrics: m.Metrics}}
				return nil
			}
			rows, err := d.client.ProjectMetricsSince(gctx, p.UUID, tr.From)
			if err != nil {
				return fmt.Errorf("metrics for %s %s: %w", p.Name, p.Version, err)
			}
			for _, r := range rows {
				series[i] = append(series[i], dtrack.PortfolioMetrics{Metrics: r.Metrics})
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	frames := data.Frames{}
	for _, name := range q.Metrics {
		metric := query.Metrics[name]
		for i, p := range projects {
			labels := projectLabels(p)
			display := fmt.Sprintf("%s %s %s", p.Name, p.Version, metric.Display)
			if q.Mode == query.ModeCurrent {
				frames = append(frames, numericFrame(metric, labels, display, metric.Value(series[i][0])))
			} else {
				frames = append(frames, timeSeriesFrame(metric, labels, display, series[i], tr))
			}
		}
	}
	return frames, nil
}

func seriesLabels(metric query.Metric, extra map[string]string) data.Labels {
	labels := data.Labels{"metric": metric.Name}
	if metric.Severity != "" {
		labels["severity"] = metric.Severity
	}
	for k, v := range extra {
		labels[k] = v
	}
	return labels
}

func numericFrame(metric query.Metric, extra map[string]string, display string, value float64) *data.Frame {
	field := data.NewField(metric.Name, seriesLabels(metric, extra), []float64{value})
	field.Config = &data.FieldConfig{DisplayNameFromDS: display}
	frame := data.NewFrame(metric.Name, field)
	frame.SetMeta(&data.FrameMeta{Type: data.FrameTypeNumericMulti, TypeVersion: data.FrameTypeVersion{0, 1}})
	return frame
}

func timeSeriesFrame(metric query.Metric, extra map[string]string, display string, rows []dtrack.PortfolioMetrics, tr backend.TimeRange) *data.Frame {
	times := make([]time.Time, 0, len(rows))
	values := make([]float64, 0, len(rows))
	for _, r := range rows {
		ts := r.LastOccurrence.Time
		if ts.Before(tr.From) || ts.After(tr.To) {
			continue
		}
		times = append(times, ts)
		values = append(values, metric.Value(r))
	}
	valueField := data.NewField(metric.Name, seriesLabels(metric, extra), values)
	valueField.Config = &data.FieldConfig{DisplayNameFromDS: display}
	frame := data.NewFrame(metric.Name, data.NewField("time", nil, times), valueField)
	frame.SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesMulti, TypeVersion: data.FrameTypeVersion{0, 1}})
	return frame
}

func emptyFrames(mode query.Mode) data.Frames {
	frameType := data.FrameTypeNumericMulti
	if mode == query.ModeTimeSeries {
		frameType = data.FrameTypeTimeSeriesMulti
	}
	frame := data.NewFrame("")
	frame.SetMeta(&data.FrameMeta{Type: frameType, TypeVersion: data.FrameTypeVersion{0, 1}})
	return data.Frames{frame}
}
