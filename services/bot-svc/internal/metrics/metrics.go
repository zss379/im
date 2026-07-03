package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Bot service metrics
var (
	// Webhook invocation count
	WebhookInvoked = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "im_bot_webhook_invoked_total",
		Help: "Total number of bot webhook invocations",
	}, []string{"bot_id", "mode"})

	// Webhook failure count
	WebhookFailed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "im_bot_webhook_failed_total",
		Help: "Total number of failed bot webhook invocations",
	}, []string{"bot_id", "mode", "reason"})

	// Webhook latency
	WebhookLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "im_bot_webhook_duration_ms",
		Help:    "Webhook invocation latency in ms",
		Buckets: []float64{50, 100, 200, 500, 1000, 2000, 3000, 5000},
	}, []string{"bot_id", "mode"})

	// SSE connections
	SSEActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "im_bot_sse_active_connections",
		Help: "Current number of active SSE connections",
	})

	SSEMaxConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "im_bot_sse_max_connections",
		Help: "Maximum number of SSE connections allowed",
	})

	// Bot trigger events consumed
	BotTriggerEvents = promauto.NewCounter(prometheus.CounterOpts{
		Name: "im_bot_trigger_events_total",
		Help: "Total number of bot_trigger events consumed",
	})

	// @trigger detection count
	AtMentionDetected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "im_bot_at_mention_detected_total",
		Help: "Total number of @mention detections",
	})
)
