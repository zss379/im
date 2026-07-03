package metrics

import "github.com/prometheus/client_golang/prometheus"

const ns = "session_svc"

var (
	SessionCreateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "session_create_total", Help: "Total sessions created"},
	)
	SessionDeleteTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "session_delete_total", Help: "Total sessions deleted"},
	)
	PinOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: ns, Name: "pin_ops_total", Help: "Total pin/unpin operations"},
		[]string{"action"}, // pin, unpin
	)
	MuteOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: ns, Name: "mute_ops_total", Help: "Total mute/unmute operations"},
		[]string{"action"}, // mute, unmute
	)
	UnreadOpsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "unread_ops_total", Help: "Total unread count updates"},
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
		SessionCreateTotal,
		SessionDeleteTotal,
		PinOpsTotal,
		MuteOpsTotal,
		UnreadOpsTotal,
		HttpRequestDuration,
		HttpRequestsTotal,
	)
}
