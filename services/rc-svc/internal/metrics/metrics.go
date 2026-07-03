package metrics

import "github.com/prometheus/client_golang/prometheus"

const ns = "rc_svc"

var (
	SensitiveCheckTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: ns, Name: "sensitive_check_total", Help: "Total sensitive word checks"},
		[]string{"passed"},
	)
	SensitiveWordHitTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "sensitive_word_hit_total", Help: "Total sensitive word hits"},
	)
	RateLimitCheckTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: ns, Name: "rate_limit_check_total", Help: "Total rate limit checks"},
		[]string{"passed"},
	)
	FileLimitCheckTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: ns, Name: "file_limit_check_total", Help: "Total file limit checks"},
		[]string{"passed"},
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
		SensitiveCheckTotal,
		SensitiveWordHitTotal,
		RateLimitCheckTotal,
		FileLimitCheckTotal,
		HttpRequestDuration,
		HttpRequestsTotal,
	)
}
