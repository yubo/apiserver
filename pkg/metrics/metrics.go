package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/yubo/apiserver/pkg/request"
)

// resettableCollector is the interface implemented by prometheus.MetricVec
// that can be used by Prometheus to collect metrics and reset their values.
type resettableCollector interface {
	Reset()
}

const (
	APIServerComponent string = "apiserver"
	OtherContentType   string = "other"
	OtherRequestMethod string = "other"
)

var (
	deprecatedRequestGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "apiserver_requested_deprecated_apis",
			Help: "Gauge of deprecated APIs that have been requested, broken out by API group, version, resource, subresource, and removed_release.",
		},
		[]string{"group", "version", "resource", "subresource", "removed_release"},
	)

	// TODO(a-robinson): Add unit tests for the handling of these metrics once
	// the upstream library supports it.
	requestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apiserver_request_total",
			Help: "Counter of apiserver requests broken out for each verb, dry run value, group, version, resource, scope, component, and HTTP response contentType and code.",
		},
		// The label_name contentType doesn't follow the label_name convention defined here:
		// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/instrumentation.md
		// But changing it would break backwards compatibility. Future label_names
		// should be all lowercase and separated by underscores.
		[]string{"verb", "dry_run", "group", "version", "resource", "subresource", "scope", "component", "contentType", "code"},
	)
	longRunningRequestGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "apiserver_longrunning_gauge",
			Help: "Gauge of all active long-running apiserver requests broken out by verb, group, version, resource, scope and component. Not all requests are tracked this way.",
		},
		[]string{"verb", "group", "version", "resource", "subresource", "scope", "component"},
	)
	requestLatencies = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "apiserver_request_duration_seconds",
			Help: "Response latency distribution in seconds for each verb, dry run value, group, version, resource, subresource, scope and component.",
			// This metric is used for verifying api call latencies SLO,
			// as well as tracking regressions in this aspects.
			// Thus we customize buckets significantly, to empower both usecases.
			Buckets: []float64{0.05, 0.1, 0.15, 0.2, 0.25, 0.3, 0.35, 0.4, 0.45, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0,
				1.25, 1.5, 1.75, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5, 6, 7, 8, 9, 10, 15, 20, 25, 30, 40, 50, 60},
		},
		[]string{"verb", "dry_run", "group", "version", "resource", "subresource", "scope", "component"},
	)
	responseSizes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "apiserver_response_sizes",
			Help: "Response size distribution in bytes for each group, version, verb, resource, subresource, scope and component.",
			// Use buckets ranging from 1000 bytes (1KB) to 10^9 bytes (1GB).
			Buckets: prometheus.ExponentialBuckets(1000, 10.0, 7),
		},
		[]string{"verb", "group", "version", "resource", "subresource", "scope", "component"},
	)
	// DroppedRequests is a number of requests dropped with 'Try again later' response"
	DroppedRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apiserver_dropped_requests_total",
			Help: "Number of requests dropped with 'Try again later' response",
		},
		[]string{"request_kind"},
	)
	// TLSHandshakeErrors is a number of requests dropped with 'TLS handshake error from' error
	TLSHandshakeErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "apiserver_tls_handshake_errors_total",
			Help: "Number of requests dropped with 'TLS handshake error from' error",
		},
	)
	// RegisteredWatchers is a number of currently registered watchers splitted by resource.
	RegisteredWatchers = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "apiserver_registered_watchers",
			Help: "Number of currently registered watchers for a given resources",
		},
		[]string{"group", "version", "kind"},
	)
	WatchEvents = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apiserver_watch_events_total",
			Help: "Number of events sent in watch clients",
		},
		[]string{"group", "version", "kind"},
	)
	WatchEventsSizes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "apiserver_watch_events_sizes",
			Help:    "Watch event size distribution in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 2.0, 8), // 1K, 2K, 4K, 8K, ..., 128K.
		},
		[]string{"group", "version", "kind"},
	)
	// Because of volatility of the base metric this is pre-aggregated one. Instead of reporting current usage all the time
	// it reports maximal usage during the last second.
	currentInflightRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "apiserver_current_inflight_requests",
			Help: "Maximal number of currently used inflight request limit of this apiserver per request kind in last second.",
		},
		[]string{"request_kind"},
	)
	currentInqueueRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "apiserver_current_inqueue_requests",
			Help: "Maximal number of queued requests in this apiserver per request kind in last second.",
		},
		[]string{"request_kind"},
	)

	requestTerminationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apiserver_request_terminations_total",
			Help: "Number of requests which apiserver terminated in self-defense.",
		},
		[]string{"verb", "group", "version", "resource", "subresource", "scope", "component", "code"},
	)

	apiSelfRequestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apiserver_selfrequest_total",
			Help: "Counter of apiserver self-requests broken out for each verb, API resource and subresource.",
		},
		[]string{"verb", "resource", "subresource"},
	)

	requestFilterDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "apiserver_request_filter_duration_seconds",
			Help:    "Request filter latency distribution in seconds, for each filter type",
			Buckets: []float64{0.0001, 0.0003, 0.001, 0.003, 0.01, 0.03, 0.1, 0.3, 1.0, 5.0},
		},
		[]string{"filter"},
	)

	// requestAbortsTotal is a number of aborted requests with http.ErrAbortHandler
	requestAbortsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apiserver_request_aborts_total",
			Help: "Number of requests which apiserver aborted possibly due to a timeout, for each group, version, verb, resource, subresource and scope",
		},
		[]string{"verb", "path"},
	)

	metrics = []resettableCollector{
		deprecatedRequestGauge,
		requestCounter,
		longRunningRequestGauge,
		requestLatencies,
		responseSizes,
		DroppedRequests,
		RegisteredWatchers,
		WatchEvents,
		WatchEventsSizes,
		currentInflightRequests,
		currentInqueueRequests,
		requestTerminationsTotal,
		apiSelfRequestCounter,
		requestFilterDuration,
		requestAbortsTotal,
	}
)

const (
	// ReadOnlyKind is a string identifying read only request kind
	ReadOnlyKind = "readOnly"
	// MutatingKind is a string identifying mutating request kind
	MutatingKind = "mutating"

	// WaitingPhase is the phase value for a request waiting in a queue
	WaitingPhase = "waiting"
	// ExecutingPhase is the phase value for an executing request
	ExecutingPhase = "executing"
)

// Reset all metrics.
func Reset() {
	for _, metric := range metrics {
		metric.Reset()
	}
}

// UpdateInflightRequestMetrics reports concurrency metrics classified by
// mutating vs Readonly.
func UpdateInflightRequestMetrics(phase string, nonmutating, mutating int) {
	for _, kc := range []struct {
		kind  string
		count int
	}{{ReadOnlyKind, nonmutating}, {MutatingKind, mutating}} {
		if phase == ExecutingPhase {
			currentInflightRequests.WithLabelValues(kc.kind).Set(float64(kc.count))
		} else {
			currentInqueueRequests.WithLabelValues(kc.kind).Set(float64(kc.count))
		}
	}
}

func RecordFilterLatency(ctx context.Context, name string, elapsed time.Duration) {
	requestFilterDuration.WithLabelValues(name).Observe(elapsed.Seconds())
}

// RecordRequestAbort records that the request was aborted possibly due to a timeout.
func RecordRequestAbort(req *http.Request, requestInfo *request.RequestInfo) {
	requestInfo = &request.RequestInfo{Verb: req.Method, Path: req.URL.Path}
	requestAbortsTotal.WithLabelValues(requestInfo.Verb, requestInfo.Path).Inc()
}
