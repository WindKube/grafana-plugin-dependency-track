package dtrack

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "dependencytrack",
		Name:      "api_request_duration_seconds",
		Help:      "Duration of Dependency-Track API requests, including retries.",
		Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
	}, []string{"endpoint", "status"})

	cacheRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "dependencytrack",
		Name:      "cache_requests_total",
		Help:      "Response cache lookups by result: hit, miss or shared (joined an in-flight request).",
	}, []string{"result"})

	inFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "dependencytrack",
		Name:      "api_requests_in_flight",
		Help:      "Dependency-Track API requests currently in flight.",
	})
)

var staticSegments = map[string]bool{
	"v1": true, "version": true, "team": true, "self": true, "project": true, "tag": true,
	"classifier": true, "metrics": true, "portfolio": true, "current": true, "since": true,
	"vulnerability": true, "finding": true, "violation": true, "policy": true,
}

// endpoint turns a request path into a bounded metric label by replacing UUIDs, tags and
// dates with ":param".
func endpoint(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		if !staticSegments[p] {
			parts[i] = ":param"
		}
	}
	return "/" + strings.Join(parts, "/")
}
