package metric

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var logger = log.WithFields(log.Fields{"component": "metrics"})

var (
	namespace = ""
	subsystem = "service"
)

var (
	reqLabels   = []string{"service", "operation", "code"}
	reqDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "idler_request_duration_seconds",
		Help:      "Bucketed histogram of processing time (s) of requests.",
		Buckets:   prometheus.ExponentialBuckets(0.05, 2, 8),
	}, reqLabels)
)

func registerMetrics() {
	reqDuration = register(reqDuration, "idler_request_duration_seconds").(*prometheus.HistogramVec)
}

func register(c prometheus.Collector, name string) prometheus.Collector {
	err := prometheus.Register(c)
	if err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return are.ExistingCollector
		}
		logger.
			WithField("err", err).
			WithField("metric_name", prometheus.BuildFQName(namespace, subsystem, name)).
			Panic("Failed to register the prometheus metric")
	}
	logger.
		WithField("metric_name", prometheus.BuildFQName(namespace, subsystem, name)).
		Debug("metric registered successfully")
	return c
}

func reportRequestDuration(jenkinsService, operation string, code int, elapsedTime float64) {
	if jenkinsService != "" && operation != "" && elapsedTime != 0 && code != 0 {
		reqDuration.WithLabelValues(jenkinsService, operation, codeVal(code)).Observe(elapsedTime)
	}
}

func codeVal(status int) string {
	code := (status - (status % 100)) / 100
	return strconv.Itoa(code) + "xx"
}
