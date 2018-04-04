package metric

import (
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
)

func TestCodeVal(t *testing.T) {
	tables := []struct {
		in  int
		out string
	}{
		{100, "1xx"},
		{200, "2xx"},
		{201, "2xx"},
		{404, "4xx"},
	}

	for _, table := range tables {
		actual := codeVal(table.in)
		if table.out != actual {
			t.Errorf("output was incorrect, want:%s, got:%s", table.out, actual)
		}
	}
}
func TestReqDurationMetric(t *testing.T) {
	recorder := PrometheusRecorder{}
	reqTimes := []time.Duration{51, 101, 201, 401, 801, 1601, 3201, 6401}
	expectedBound := []float64{0.05, 0.1, 0.2, 0.4, 0.8, 1.6, 3.2, 6.4}
	expectedCnt := []uint64{0, 1, 2, 3, 4, 5, 6, 7}

	// add post method
	for _, reqTime := range reqTimes {
		startTime := time.Now().Add(time.Millisecond * -reqTime)
		recorder.RecordReqDuration("jenkins", "idle", 200, time.Since(startTime).Seconds())
	}

	// validate
	reqMetric, _ := reqDuration.GetMetricWithLabelValues("jenkins", "idle", "2xx")
	m := &dto.Metric{}
	reqMetric.Write(m)
	checkHistogram(t, m, uint64(len(reqTimes)), expectedBound, expectedCnt)
}

func checkHistogram(t *testing.T, m *dto.Metric, expectedCount uint64, expectedBound []float64, expectedCnt []uint64) {
	if expectedCount != m.Histogram.GetSampleCount() {
		t.Errorf("Histogram count was incorrect, want: %d, got: %d",
			expectedCount, m.Histogram.GetSampleCount())
	}
	for ind, bucket := range m.Histogram.GetBucket() {
		if expectedBound[ind] != *bucket.UpperBound {
			t.Errorf("Bucket upper bound was incorrect, want: %f, got: %f\n",
				expectedBound[ind], *bucket.UpperBound)
		}
		if expectedCnt[ind] != *bucket.CumulativeCount {
			t.Errorf("Bucket cumulative count was incorrect, want: %d, got: %d\n",
				expectedCnt[ind], *bucket.CumulativeCount)
		}
	}
}
