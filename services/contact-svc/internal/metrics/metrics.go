package metrics

import "github.com/prometheus/client_golang/prometheus"

const ns = "contact_svc"

var (
	DeptCreateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "dept_create_total", Help: "Total departments created"},
	)
	DeptDeleteTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "dept_delete_total", Help: "Total departments deleted"},
	)
	MemberUpdateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "member_update_total", Help: "Total member profile updates"},
	)
	SyncTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "sync_total", Help: "Total HR sync operations"},
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
		DeptCreateTotal,
		DeptDeleteTotal,
		MemberUpdateTotal,
		SyncTotal,
		HttpRequestDuration,
		HttpRequestsTotal,
	)
}
