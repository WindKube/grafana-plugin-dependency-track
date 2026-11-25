package plugin

import (
	"context"
	"time"

	"github.com/WindKube/grafana-plugin-dependency-track/pkg/query"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var queryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "dependencytrack",
	Name:      "query_duration_seconds",
	Help:      "Duration of data source queries by query type.",
	Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
}, []string{"query_type", "status"})

// observe wraps a query with a span, a duration metric and a debug log line.
func observe(ctx context.Context, q *query.Query, run func(context.Context) (data.Frames, error)) (data.Frames, error) {
	ctx, span := tracing.DefaultTracer().Start(ctx, "dependencytrack.query", trace.WithAttributes(
		attribute.String("query_type", string(q.QueryType)),
		attribute.String("format", string(q.Format)),
		attribute.String("projects.mode", string(q.Projects.Mode)),
	))
	defer span.End()

	start := time.Now()
	frames, err := run(ctx)
	elapsed := time.Since(start)

	status := "ok"
	logger := log.DefaultLogger.FromContext(ctx)
	if err != nil {
		status = "error"
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Warn("Query failed", "queryType", q.QueryType, "duration", elapsed, "error", err)
	} else {
		rows := 0
		for _, f := range frames {
			rows += f.Rows()
		}
		span.SetAttributes(attribute.Int("frames", len(frames)), attribute.Int("rows", rows))
		logger.Debug("Query finished", "queryType", q.QueryType, "duration", elapsed, "frames", len(frames), "rows", rows)
	}
	queryDuration.WithLabelValues(string(q.QueryType), status).Observe(elapsed.Seconds())
	return frames, err
}
