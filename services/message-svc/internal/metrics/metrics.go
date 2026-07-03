package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	messageSendTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "message_send_total",
			Help: "Total number of messages sent.",
		},
		[]string{"msg_type", "status"},
	)

	messageRecallTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "message_recall_total",
			Help: "Total number of messages recalled.",
		},
	)

	messageForwardTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "message_forward_total",
			Help: "Total number of messages forwarded.",
		},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	messageSearchTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "message_search_total",
			Help: "Total number of message searches.",
		},
	)

	readOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "message_read_ops_total",
			Help: "Total number of read receipt operations.",
		},
		[]string{"op"}, // mark_read, get_receipt, get_status
	)
)

func init() {
	prometheus.MustRegister(messageSendTotal, messageRecallTotal, messageForwardTotal,
		httpRequestDuration, httpRequestsTotal, messageSearchTotal, readOpsTotal)
}

type (
	CounterInt   prometheus.Counter
	CounterVec   *prometheus.CounterVec
	HistogramVec *prometheus.HistogramVec
)

func IncMessageSend(msgType string, status string) {
	messageSendTotal.WithLabelValues(msgType, status).Inc()
}

func IncMessageRecall() {
	messageRecallTotal.Inc()
}

func IncMessageForward() {
	messageForwardTotal.Inc()
}

func IncMessageSearch() {
	messageSearchTotal.Inc()
}

func IncReadOp(op string) {
	readOpsTotal.WithLabelValues(op).Inc()
}

func ObserveHTTP(method, path string, status int, duration float64) {
	statusStr := formatStatus(status)
	httpRequestDuration.WithLabelValues(method, path, statusStr).Observe(duration)
	httpRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
}

func formatStatus(status int) string {
	if status >= 200 && status < 300 {
		return "2xx"
	} else if status >= 300 && status < 400 {
		return "3xx"
	} else if status >= 400 && status < 500 {
		return "4xx"
	}
	return "5xx"
}

// Handler 返回 /metrics HTTP handler
func Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}
}
