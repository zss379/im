package metrics

import "github.com/prometheus/client_golang/prometheus"

const ns = "auth_svc"

var (
	LoginTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: ns, Name: "login_total", Help: "Total login attempts"},
		[]string{"result"}, // success, failure
	)
	UserCreateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "user_create_total", Help: "Total users created"},
	)
	TokenRefreshTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "token_refresh_total", Help: "Total token refreshes"},
	)
	StatusChangeTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "status_change_total", Help: "Total status changes"},
	)
	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Namespace: ns, Name: "http_request_duration_seconds", Help: "HTTP request latencies", Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1}},
		[]string{"method", "path"},
	)
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: ns, Name: "http_requests_total", Help: "Total HTTP requests"},
		[]string{"method", "path", "status"},
	)
)

func init() {
	prometheus.MustRegister(
		LoginTotal,
		UserCreateTotal,
		TokenRefreshTotal,
		StatusChangeTotal,
		HttpRequestDuration,
		HttpRequestsTotal,
	)
}
