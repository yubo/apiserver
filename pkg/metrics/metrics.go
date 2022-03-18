package metrics

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/golib/util/sets"
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
			Help: "Gauge of deprecated APIs that have been requested, broken out by API group, version, resource, subresource.",
		},
		[]string{"path"},
	)

	// TODO(a-robinson): Add unit tests for the handling of these metrics once
	// the upstream library supports it.
	requestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apiserver_request_total",
			//Help: "Counter of apiserver requests broken out for each verb, dry run value, group, version, resource, scope, component, and HTTP response contentType and code.",
			Help: "Counter of apiserver requests broken out for each verb, dry run value, path, component, and HTTP response contentType and code.",
		},
		// The label_name contentType doesn't follow the label_name convention defined here:
		// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/instrumentation.md
		// But changing it would break backwards compatibility. Future label_names
		// should be all lowercase and separated by underscores.
		//[]string{"verb", "dry_run", "group", "version", "resource", "subresource", "scope", "component", "contentType", "code"},
		[]string{"verb", "dry_run", "path", "component", "contentType", "code"},
	)
	longRunningRequestGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "apiserver_longrunning_gauge",
			Help: "Gauge of all active long-running apiserver requests broken out by verb, group, version, resource, scope and component. Not all requests are tracked this way.",
		},
		//[]string{"verb", "group", "version", "resource", "subresource", "scope", "component"},
		[]string{"verb", "path", "component"},
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
		//[]string{"verb", "dry_run", "group", "version", "resource", "subresource", "scope", "component"},
		[]string{"verb", "dry_run", "path", "component"},
	)
	responseSizes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "apiserver_response_sizes",
			Help: "Response size distribution in bytes for each group, version, verb, resource, subresource, scope and component.",
			// Use buckets ranging from 1000 bytes (1KB) to 10^9 bytes (1GB).
			Buckets: prometheus.ExponentialBuckets(1000, 10.0, 7),
		},
		//[]string{"verb", "group", "version", "resource", "subresource", "scope", "component"},
		[]string{"verb", "path", "component"},
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
		//[]string{"group", "version", "kind"},
		[]string{"path"},
	)
	WatchEvents = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apiserver_watch_events_total",
			Help: "Number of events sent in watch clients",
		},
		//[]string{"group", "version", "kind"},
		[]string{"path"},
	)
	WatchEventsSizes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "apiserver_watch_events_sizes",
			Help:    "Watch event size distribution in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 2.0, 8), // 1K, 2K, 4K, 8K, ..., 128K.
		},
		//[]string{"group", "version", "kind"},
		[]string{"path"},
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
		//[]string{"verb", "group", "version", "resource", "subresource", "scope", "component", "code"},
		[]string{"verb", "path", "component", "code"},
	)

	apiSelfRequestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "apiserver_selfrequest_total",
			Help: "Counter of apiserver self-requests broken out for each verb, API resource and subresource.",
		},
		//[]string{"verb", "resource", "subresource"},
		[]string{"verb", "path"},
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

	// these are the known (e.g. whitelisted/known) content types which we will report for
	// request metrics. Any other RFC compliant content types will be aggregated under 'unknown'
	knownMetricContentTypes = sets.NewString(
		"application/apply-patch+yaml",
		"application/json",
		"application/json-patch+json",
		"application/merge-patch+json",
		"application/strategic-merge-patch+json",
		"application/vnd.kubernetes.protobuf",
		"application/vnd.kubernetes.protobuf;stream=watch",
		"application/yaml",
		"text/plain",
		"text/plain;charset=utf-8")
	// these are the valid request methods which we report in our metrics. Any other request methods
	// will be aggregated under 'unknown'
	validRequestMethods = sets.NewString(
		"APPLY",
		"CONNECT",
		"CREATE",
		"DELETE",
		"DELETECOLLECTION",
		"GET",
		"LIST",
		"PATCH",
		"POST",
		"PROXY",
		"PUT",
		"UPDATE",
		"WATCH",
		"WATCHLIST")
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

const (
	// deprecatedAnnotationKey is a key for an audit annotation set to
	// "true" on requests made to deprecated API versions
	deprecatedAnnotationKey = "k8s.io/deprecated"
	// removedReleaseAnnotationKey is a key for an audit annotation set to
	// the target removal release, in "<major>.<minor>" format,
	// on requests made to deprecated API versions with a target removal release
	removedReleaseAnnotationKey = "k8s.io/removed-release"
)

// nothing
func Register() {}

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

// RecordRequestTermination records that the request was terminated early as part of a resource
// preservation or apiserver self-defense mechanism (e.g. timeouts, maxinflight throttling,
// proxyHandler errors). RecordRequestTermination should only be called zero or one times
// per request.
func RecordRequestTermination(req *http.Request, requestInfo *request.RequestInfo, component string, code int) {
	if requestInfo == nil {
		requestInfo = &request.RequestInfo{Verb: req.Method, Path: req.URL.Path}
	}
	scope := CleanScope(requestInfo)

	// We don't use verb from <requestInfo>, as this may be propagated from
	// InstrumentRouteFunc which is registered in installer.go with predefined
	// list of verbs (different than those translated to RequestInfo).
	// However, we need to tweak it e.g. to differentiate GET from LIST.
	reportedVerb := cleanVerb(canonicalVerb(strings.ToUpper(req.Method), scope), req)

	if requestInfo.IsResourceRequest {
		requestTerminationsTotal.WithLabelValues(reportedVerb, requestInfo.APIGroup, requestInfo.APIVersion, requestInfo.Resource, requestInfo.Subresource, scope, component, codeToString(code)).Inc()
	} else {
		requestTerminationsTotal.WithLabelValues(reportedVerb, "", "", "", requestInfo.Path, scope, component, codeToString(code)).Inc()
	}
}

// RecordLongRunning tracks the execution of a long running request against the API server. It provides an accurate count
// of the total number of open long running requests. requestInfo may be nil if the caller is not in the normal request flow.
func RecordLongRunning(req *http.Request, requestInfo *request.RequestInfo, component string, fn func()) {
	if requestInfo == nil {
		requestInfo = &request.RequestInfo{Verb: req.Method, Path: req.URL.Path}
	}
	var g prometheus.Gauge
	scope := CleanScope(requestInfo)

	// We don't use verb from <requestInfo>, as this may be propagated from
	// InstrumentRouteFunc which is registered in installer.go with predefined
	// list of verbs (different than those translated to RequestInfo).
	// However, we need to tweak it e.g. to differentiate GET from LIST.
	reportedVerb := cleanVerb(canonicalVerb(strings.ToUpper(req.Method), scope), req)

	if requestInfo.IsResourceRequest {
		g = longRunningRequestGauge.WithLabelValues(reportedVerb, requestInfo.APIGroup, requestInfo.APIVersion, requestInfo.Resource, requestInfo.Subresource, scope, component)
	} else {
		g = longRunningRequestGauge.WithLabelValues(reportedVerb, "", "", "", requestInfo.Path, scope, component)
	}
	g.Inc()
	defer g.Dec()
	fn()
}

// MonitorRequest handles standard transformations for client and the reported verb and then invokes Monitor to record
// a request. verb must be uppercase to be backwards compatible with existing monitoring tooling.
func MonitorRequest(req *http.Request, verb, path, component string, deprecated bool, contentType string, httpCode, respSize int, elapsed time.Duration) {
	// We don't use verb from <requestInfo>, as this may be propagated from
	// InstrumentRouteFunc which is registered in installer.go with predefined
	// list of verbs (different than those translated to RequestInfo).
	// However, we need to tweak it e.g. to differentiate GET from LIST.
	reportedVerb := cleanVerb(strings.ToUpper(req.Method), req)

	dryRun := cleanDryRun(req.URL)
	elapsedSeconds := elapsed.Seconds()
	cleanContentType := cleanContentType(contentType)
	requestCounter.WithLabelValues(reportedVerb, dryRun, path, component, cleanContentType, codeToString(httpCode)).Inc()
	// MonitorRequest happens after authentication, so we can trust the username given by the request
	info, ok := request.UserFrom(req.Context())
	if ok && info.GetName() == user.APIServerUser {
		apiSelfRequestCounter.WithLabelValues(reportedVerb, path).Inc()
	}
	if deprecated {
		deprecatedRequestGauge.WithLabelValues(path).Set(1)
		audit.AddAuditAnnotation(req.Context(), deprecatedAnnotationKey, "true")
	}
	requestLatencies.WithLabelValues(reportedVerb, dryRun, path, component).Observe(elapsedSeconds)
	// We are only interested in response sizes of read requests.
	if verb == "GET" || verb == "LIST" {
		responseSizes.WithLabelValues(reportedVerb, path, component).Observe(float64(respSize))
	}
}

// InstrumentRouteFunc works like Prometheus' InstrumentHandlerFunc but wraps
// the go-restful RouteFunction instead of a HandlerFunc plus some Kubernetes endpoint specific information.
func InstrumentRouteFunc(verb, path, component string, deprecated bool, routeFunc restful.RouteFunction) restful.RouteFunction {
	return restful.RouteFunction(func(req *restful.Request, response *restful.Response) {
		requestReceivedTimestamp, ok := request.ReceivedTimestampFrom(req.Request.Context())
		if !ok {
			requestReceivedTimestamp = time.Now()
		}

		delegate := &ResponseWriterDelegator{ResponseWriter: response.ResponseWriter}

		_, cn := response.ResponseWriter.(http.CloseNotifier)
		_, fl := response.ResponseWriter.(http.Flusher)
		_, hj := response.ResponseWriter.(http.Hijacker)
		var rw http.ResponseWriter
		if cn && fl && hj {
			rw = &fancyResponseWriterDelegator{delegate}
		} else {
			rw = delegate
		}
		response.ResponseWriter = rw

		routeFunc(req, response)

		MonitorRequest(req.Request, verb, path, component, deprecated, delegate.Header().Get("Content-Type"), delegate.Status(), delegate.ContentLength(), time.Since(requestReceivedTimestamp))
	})
}

// InstrumentHandlerFunc works like Prometheus' InstrumentHandlerFunc but adds some Kubernetes endpoint specific information.
func InstrumentHandlerFunc(verb, path, component string, deprecated bool, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		requestReceivedTimestamp, ok := request.ReceivedTimestampFrom(req.Context())
		if !ok {
			requestReceivedTimestamp = time.Now()
		}

		delegate := &ResponseWriterDelegator{ResponseWriter: w}

		_, cn := w.(http.CloseNotifier)
		_, fl := w.(http.Flusher)
		_, hj := w.(http.Hijacker)
		if cn && fl && hj {
			w = &fancyResponseWriterDelegator{delegate}
		} else {
			w = delegate
		}

		handler(w, req)

		MonitorRequest(req, verb, path, component, deprecated, delegate.Header().Get("Content-Type"), delegate.Status(), delegate.ContentLength(), time.Since(requestReceivedTimestamp))
	}
}

// cleanContentType binds the contentType (for metrics related purposes) to a
// bounded set of known/expected content-types.
func cleanContentType(contentType string) string {
	normalizedContentType := strings.ToLower(contentType)
	if strings.HasSuffix(contentType, " stream=watch") || strings.HasSuffix(contentType, " charset=utf-8") {
		normalizedContentType = strings.ReplaceAll(contentType, " ", "")
	}
	if knownMetricContentTypes.Has(normalizedContentType) {
		return normalizedContentType
	}
	return OtherContentType
}

// CleanScope returns the scope of the request.
func CleanScope(requestInfo *request.RequestInfo) string {
	if requestInfo.Namespace != "" {
		return "namespace"
	}
	if requestInfo.Name != "" {
		return "resource"
	}
	if requestInfo.IsResourceRequest {
		return "cluster"
	}
	// this is the empty scope
	return ""
}

func canonicalVerb(verb string, scope string) string {
	switch verb {
	case "GET", "HEAD":
		if scope != "resource" && scope != "" {
			return "LIST"
		}
		return "GET"
	default:
		return verb
	}
}

func cleanVerb(verb string, request *http.Request) string {
	reportedVerb := verb
	if verb == "LIST" {
		// see apimachinery/pkg/runtime/conversion.go Convert_Slice_string_To_bool
		if values := request.URL.Query()["watch"]; len(values) > 0 {
			if value := strings.ToLower(values[0]); value != "0" && value != "false" {
				reportedVerb = "WATCH"
			}
		}
	}
	// normalize the legacy WATCHLIST to WATCH to ensure users aren't surprised by metrics
	if verb == "WATCHLIST" {
		reportedVerb = "WATCH"
	}
	//if verb == "PATCH" && request.Header.Get("Content-Type") == string(types.ApplyPatchType) && utilfeature.DefaultFeatureGate.Enabled(features.ServerSideApply) {
	//	reportedVerb = "APPLY"
	//}
	if validRequestMethods.Has(reportedVerb) {
		return reportedVerb
	}
	return OtherRequestMethod
}

func cleanDryRun(u *url.URL) string {
	// avoid allocating when we don't see dryRun in the query
	if !strings.Contains(u.RawQuery, "dryRun") {
		return ""
	}
	dryRun := u.Query()["dryRun"]
	// Since dryRun could be valid with any arbitrarily long length
	// we have to dedup and sort the elements before joining them together
	// TODO: this is a fairly large allocation for what it does, consider
	//   a sort and dedup in a single pass
	return strings.Join(sets.NewString(dryRun...).List(), ",")
}

// ResponseWriterDelegator interface wraps http.ResponseWriter to additionally record content-length, status-code, etc.
type ResponseWriterDelegator struct {
	http.ResponseWriter

	status      int
	written     int64
	wroteHeader bool
}

func (r *ResponseWriterDelegator) WriteHeader(code int) {
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *ResponseWriterDelegator) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.written += int64(n)
	return n, err
}

func (r *ResponseWriterDelegator) Status() int {
	return r.status
}

func (r *ResponseWriterDelegator) ContentLength() int {
	return int(r.written)
}

type fancyResponseWriterDelegator struct {
	*ResponseWriterDelegator
}

func (f *fancyResponseWriterDelegator) CloseNotify() <-chan bool {
	return f.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

func (f *fancyResponseWriterDelegator) Flush() {
	f.ResponseWriter.(http.Flusher).Flush()
}

func (f *fancyResponseWriterDelegator) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return f.ResponseWriter.(http.Hijacker).Hijack()
}

// Small optimization over Itoa
func codeToString(s int) string {
	switch s {
	case 100:
		return "100"
	case 101:
		return "101"

	case 200:
		return "200"
	case 201:
		return "201"
	case 202:
		return "202"
	case 203:
		return "203"
	case 204:
		return "204"
	case 205:
		return "205"
	case 206:
		return "206"

	case 300:
		return "300"
	case 301:
		return "301"
	case 302:
		return "302"
	case 304:
		return "304"
	case 305:
		return "305"
	case 307:
		return "307"

	case 400:
		return "400"
	case 401:
		return "401"
	case 402:
		return "402"
	case 403:
		return "403"
	case 404:
		return "404"
	case 405:
		return "405"
	case 406:
		return "406"
	case 407:
		return "407"
	case 408:
		return "408"
	case 409:
		return "409"
	case 410:
		return "410"
	case 411:
		return "411"
	case 412:
		return "412"
	case 413:
		return "413"
	case 414:
		return "414"
	case 415:
		return "415"
	case 416:
		return "416"
	case 417:
		return "417"
	case 418:
		return "418"

	case 500:
		return "500"
	case 501:
		return "501"
	case 502:
		return "502"
	case 503:
		return "503"
	case 504:
		return "504"
	case 505:
		return "505"

	case 428:
		return "428"
	case 429:
		return "429"
	case 431:
		return "431"
	case 511:
		return "511"

	default:
		return strconv.Itoa(s)
	}
}
