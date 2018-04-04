package metric

// Recorder interface that encapsulates all logic of metrics
type Recorder interface {
	Initialize()
	RecordReqDuration(jenkinsService, operation string, code int, elapsedTime float64)
}

// PrometheusRecorder struct used to record metrics to be consumed by Prometheus
type PrometheusRecorder struct {
}

// Initialize all metrics
func (pr PrometheusRecorder) Initialize() {
	registerMetrics()
}

// RecordReqDuration records the duration of given operation in metrics system
func (pr PrometheusRecorder) RecordReqDuration(jenkinsService, operation string, code int, elapsedTime float64) {
	reportRequestDuration(jenkinsService, operation, code, elapsedTime)
}
