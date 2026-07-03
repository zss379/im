package metrics

import "github.com/prometheus/client_golang/prometheus"

const ns = "audit_svc"

var (
	AdminLogWriteTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "admin_log_write_total", Help: "Total admin operation logs written"},
	)
	MsgLogWriteTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "msg_log_write_total", Help: "Total message audit logs written"},
	)
	MsgLogBatchWriteTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "msg_log_batch_write_total", Help: "Total batch message audit log writes"},
	)
	LogQueryTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "log_query_total", Help: "Total log query operations"},
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
		AdminLogWriteTotal,
		MsgLogWriteTotal,
		MsgLogBatchWriteTotal,
		LogQueryTotal,
		HttpRequestDuration,
		HttpRequestsTotal,
	)
}
