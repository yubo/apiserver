package metrics

import (
	"math"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// requestLatency is a Prometheus Summary metric type partitioned by
	// "verb" and "url" labels. It is used for the rest client latency metrics.
	requestLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rest_client_request_duration_seconds",
			Help:    "Request latency in seconds. Broken down by verb and URL.",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		},
		[]string{"verb", "url"},
	)

	rateLimiterLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rest_client_rate_limiter_duration_seconds",
			Help:    "Client side rate limiter latency in seconds. Broken down by verb and URL.",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		},
		[]string{"verb", "url"},
	)

	requestResult = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rest_client_requests_total",
			Help: "Number of HTTP requests, partitioned by status code, method, and host.",
		},
		[]string{"code", "method", "host"},
	)

	execPluginCertTTLAdapter = &expiryToTTLAdapter{}

	execPluginCertTTL = promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "rest_client_exec_plugin_ttl_seconds",
			Help: "Gauge of the shortest TTL (time-to-live) of the client " +
				"certificate(s) managed by the auth exec plugin. The value " +
				"is in seconds until certificate expiry (negative if " +
				"already expired). If auth exec plugins are unused or manage no " +
				"TLS certificates, the value will be +INF.",
			//StabilityLevel: promauto.ALPHA,
		},
		func() float64 {
			if execPluginCertTTLAdapter.e == nil {
				return math.Inf(1)
			}
			return execPluginCertTTLAdapter.e.Sub(time.Now()).Seconds()
		},
	)

	execPluginCertRotation = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name: "rest_client_exec_plugin_certificate_rotation_age",
			Help: "Histogram of the number of seconds the last auth exec " +
				"plugin client certificate lived before being rotated. " +
				"If auth exec plugin client certificates are unused, " +
				"histogram will contain no data.",
			// There are three sets of ranges these buckets intend to capture:
			//   - 10-60 minutes: captures a rotation cadence which is
			//     happening too quickly.
			//   - 4 hours - 1 month: captures an ideal rotation cadence.
			//   - 3 months - 4 years: captures a rotation cadence which is
			//     is probably too slow or much too slow.
			Buckets: []float64{
				600,       // 10 minutes
				1800,      // 30 minutes
				3600,      // 1  hour
				14400,     // 4  hours
				86400,     // 1  day
				604800,    // 1  week
				2592000,   // 1  month
				7776000,   // 3  months
				15552000,  // 6  months
				31104000,  // 1  year
				124416000, // 4  years
			},
		},
	)
	// ClientCertExpiry is the expiry time of a client certificate
	ClientCertExpiry ExpiryMetric = noopExpiry{}
	// ClientCertRotationAge is the age of a certificate that has just been rotated.
	ClientCertRotationAge DurationMetric = noopDuration{}
	// RequestLatency is the latency metric that rest clients will update.
	RequestLatency LatencyMetric = noopLatency{}
	// RateLimiterLatency is the client side rate limiter latency metric.
	RateLimiterLatency LatencyMetric = noopLatency{}
	// RequestResult is the result metric that rest clients will update.
	RequestResult ResultMetric = noopResult{}
)

func RestRegister() {
	RequestResult = &resultAdapter{m: requestResult}
	ClientCertExpiry = execPluginCertTTLAdapter
	ClientCertRotationAge = &rotationAdapter{m: execPluginCertRotation}
	RequestLatency = &latencyAdapter{m: requestLatency}
	RateLimiterLatency = &latencyAdapter{m: rateLimiterLatency}
}

// DurationMetric is a measurement of some amount of time.
type DurationMetric interface {
	Observe(duration time.Duration)
}

// ExpiryMetric sets some time of expiry. If nil, assume not relevant.
type ExpiryMetric interface {
	Set(expiry *time.Time)
}

// LatencyMetric observes client latency partitioned by verb and url.
type LatencyMetric interface {
	Observe(verb string, u url.URL, latency time.Duration)
}

// ResultMetric counts response codes partitioned by method and host.
type ResultMetric interface {
	Increment(code string, method string, host string)
}

type latencyAdapter struct {
	m *prometheus.HistogramVec
}

func (l *latencyAdapter) Observe(verb string, u url.URL, latency time.Duration) {
	l.m.WithLabelValues(verb, u.String()).Observe(latency.Seconds())
}

type resultAdapter struct {
	m *prometheus.CounterVec
}

func (r *resultAdapter) Increment(code, method, host string) {
	r.m.WithLabelValues(code, method, host).Inc()
}

type expiryToTTLAdapter struct {
	e *time.Time
}

func (e *expiryToTTLAdapter) Set(expiry *time.Time) {
	e.e = expiry
}

type rotationAdapter struct {
	m prometheus.Histogram
}

func (r *rotationAdapter) Observe(d time.Duration) {
	r.m.Observe(d.Seconds())
}

type noopDuration struct{}

func (noopDuration) Observe(time.Duration) {}

type noopExpiry struct{}

func (noopExpiry) Set(*time.Time) {}

type noopLatency struct{}

func (noopLatency) Observe(string, url.URL, time.Duration) {}

type noopResult struct{}

func (noopResult) Increment(string, string, string) {}
