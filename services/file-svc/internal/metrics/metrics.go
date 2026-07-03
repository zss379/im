package metrics

import "github.com/prometheus/client_golang/prometheus"

const ns = "file_svc"

var (
	FileUploadTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: ns, Name: "file_upload_total", Help: "Total file uploads"},
		[]string{"file_type"}, // image, video, audio, file
	)
	FileDownloadTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "file_download_total", Help: "Total file downloads"},
	)
	MultipartUploadTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "multipart_upload_total", Help: "Total multipart uploads"},
	)
	UploadBytesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Namespace: ns, Name: "upload_bytes_total", Help: "Total bytes uploaded"},
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
		FileUploadTotal,
		FileDownloadTotal,
		MultipartUploadTotal,
		UploadBytesTotal,
		HttpRequestDuration,
		HttpRequestsTotal,
	)
}
