package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	serviceUp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cel_service_up",
		Help: "1 when the service process registered metrics",
	}, []string{"service"})

	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cel_http_requests_total",
		Help: "HTTP requests by service, method, route pattern, and status",
	}, []string{"service", "method", "route", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "cel_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "method", "route"})
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Register exposes /metrics and marks the service as up.
func Register(mux *http.ServeMux, service string) {
	serviceUp.WithLabelValues(service).Set(1)
	mux.Handle("GET /metrics", promhttp.Handler())
}

// Wrap records request counts and latency for all routes except /metrics.
func Wrap(service string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		route := r.URL.Path
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		status := strconv.Itoa(sw.status)
		httpRequests.WithLabelValues(service, r.Method, route, status).Inc()
		httpDuration.WithLabelValues(service, r.Method, route).Observe(time.Since(start).Seconds())
	})
}
