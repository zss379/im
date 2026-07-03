package metrics

import "github.com/prometheus/client_golang/prometheus"

const ns = "group_svc"

var (
	GroupCreateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "group_create_total", Help: "Total groups created"},
	)
	GroupDismissTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "group_dismiss_total", Help: "Total groups dismissed"},
	)
	MemberAddTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "member_add_total", Help: "Total members added"},
	)
	MemberRemoveTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "member_remove_total", Help: "Total members removed"},
	)
	MuteOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: ns, Name: "mute_ops_total", Help: "Total mute operations"},
		[]string{"type"}, // member_mute, member_unmute, global_mute, global_unmute
	)
	JoinRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: ns, Name: "join_request_total", Help: "Total join requests"},
		[]string{"status"}, // created, approved, rejected
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
		GroupCreateTotal,
		GroupDismissTotal,
		MemberAddTotal,
		MemberRemoveTotal,
		MuteOpsTotal,
		JoinRequestTotal,
		HttpRequestDuration,
		HttpRequestsTotal,
	)
}
