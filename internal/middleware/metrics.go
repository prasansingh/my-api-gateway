package middleware

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "route", "status_code"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.015, 0.02, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 5, 10, 15},
	}, []string{"method", "route"})

	requestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gateway_requests_in_flight",
		Help: "Number of HTTP requests currently being processed.",
	})

	responseSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_response_size_bytes",
		Help:    "Size of HTTP responses in bytes.",
		Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000, 5000000},
	}, []string{"method", "route"})
)

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

// Metrics returns a middleware that records Prometheus metrics for each request.
// It must be placed inside Logging() in the chain so RouteInfo is available via context.
func Metrics() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestsInFlight.Inc()
			defer requestsInFlight.Dec()

			mw := &metricsResponseWriter{ResponseWriter: w, statusCode: 200}

			timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
				route := "unmatched"
				if ri := GetRouteInfo(r.Context()); ri != nil && ri.MatchedRoute != "" {
					route = ri.MatchedRoute
				}
				requestDuration.WithLabelValues(r.Method, route).Observe(v)
			}))

			next.ServeHTTP(mw, r)

			timer.ObserveDuration()

			route := "unmatched"
			if ri := GetRouteInfo(r.Context()); ri != nil && ri.MatchedRoute != "" {
				route = ri.MatchedRoute
			}

			requestsTotal.WithLabelValues(r.Method, route, strconv.Itoa(mw.statusCode)).Inc()
			responseSize.WithLabelValues(r.Method, route).Observe(float64(mw.bytesWritten))
		})
	}
}
