// Package prom provides a Prometheus-backed monitor.MetricsRecorder.
package prom

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Recorder holds registered Prometheus metrics and implements MetricsRecorder.
type Recorder struct {
	dbConnectDuration   prometheus.Histogram
	dbQueryErrors       *prometheus.CounterVec
	cacheOpErrors       *prometheus.CounterVec
	lockAcquireFailures prometheus.Counter
	httpRequestDuration *prometheus.HistogramVec
}

// NewRecorder creates and registers all gocore Prometheus metrics.
// Pass prometheus.DefaultRegisterer for production, or a custom registry for tests.
func NewRecorder(reg prometheus.Registerer) *Recorder {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	r := &Recorder{
		dbConnectDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "gocore",
			Subsystem: "database",
			Name:      "connect_duration_seconds",
			Help:      "Duration of database connection establishment in seconds",
			Buckets:   prometheus.DefBuckets,
		}),
		dbQueryErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gocore",
			Subsystem: "database",
			Name:      "query_errors_total",
			Help:      "Total number of database errors by error class",
		}, []string{"error_class"}),
		cacheOpErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "gocore",
			Subsystem: "cache",
			Name:      "op_errors_total",
			Help:      "Total number of cache operation failures by operation",
		}, []string{"op"}),
		lockAcquireFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "gocore",
			Subsystem: "lock",
			Name:      "acquire_failures_total",
			Help:      "Total number of distributed lock acquisition contention failures",
		}),
		httpRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "gocore",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request latency histogram in seconds",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "route", "status"}),
	}

	reg.MustRegister(
		r.dbConnectDuration,
		r.dbQueryErrors,
		r.cacheOpErrors,
		r.lockAcquireFailures,
		r.httpRequestDuration,
	)

	return r
}

// DBConnectDuration returns the histogram for database connection latency.
func (r *Recorder) DBConnectDuration() prometheus.Histogram {
	return r.dbConnectDuration
}

// DBQueryErrors returns the counter vec for database query errors.
func (r *Recorder) DBQueryErrors() *prometheus.CounterVec {
	return r.dbQueryErrors
}

// CacheOpErrors returns the counter vec for cache operation failures.
func (r *Recorder) CacheOpErrors() *prometheus.CounterVec {
	return r.cacheOpErrors
}

// LockAcquireFailures returns the counter for lock acquisition failures.
func (r *Recorder) LockAcquireFailures() prometheus.Counter {
	return r.lockAcquireFailures
}

// RecordDuration records an HTTP request duration metric.
func (r *Recorder) RecordDuration(method, path string, statusCode int, duration time.Duration) {
	r.httpRequestDuration.WithLabelValues(method, path, strconv.Itoa(statusCode)).Observe(duration.Seconds())
}

// IncrementInFlight is a no-op placeholder for in-flight request tracking.
func (r *Recorder) IncrementInFlight() {}

// DecrementInFlight is a no-op placeholder for in-flight request tracking.
func (r *Recorder) DecrementInFlight() {}

// Handler returns the http.Handler serving the Prometheus /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}
