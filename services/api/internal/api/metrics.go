package api

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var finalizeDurationBuckets = []int64{50, 100, 250, 500, 1000, 2500, 5000}

type Metrics struct {
	mu sync.Mutex

	apiRequestsTotal int64
	apiErrorsTotal   int64

	finalizeRequestsTotal int64
	finalizeErrorsTotal   int64
	finalizeFallbackTotal int64
	finalizeDurationCount int64
	finalizeDurationSumMS int64
	finalizeBuckets       map[int64]int64

	copyRequestsTotal int64
	copyFailuresTotal int64
	copyTimeoutsTotal int64
}

func NewMetrics() *Metrics {
	buckets := make(map[int64]int64, len(finalizeDurationBuckets))
	for _, bucket := range finalizeDurationBuckets {
		buckets[bucket] = 0
	}
	return &Metrics{finalizeBuckets: buckets}
}

func (m *Metrics) ObserveAPIRequest(statusCode int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.apiRequestsTotal++
	if statusCode >= http.StatusBadRequest {
		m.apiErrorsTotal++
	}
}

func (m *Metrics) ObserveFinalize(duration time.Duration, fallbackUsed bool, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.finalizeRequestsTotal++
	if !success {
		m.finalizeErrorsTotal++
	}
	if fallbackUsed {
		m.finalizeFallbackTotal++
	}
	ms := duration.Milliseconds()
	if ms < 0 {
		ms = 0
	}
	m.finalizeDurationCount++
	m.finalizeDurationSumMS += ms
	for _, bucket := range finalizeDurationBuckets {
		if ms <= bucket {
			m.finalizeBuckets[bucket]++
		}
	}
}

func (m *Metrics) ObserveCopyRequest() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.copyRequestsTotal++
}

func (m *Metrics) ObserveCopyFailure(timeout bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.copyFailuresTotal++
	if timeout {
		m.copyTimeoutsTotal++
	}
}

func (m *Metrics) Render() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var b strings.Builder
	writeMetric(&b, "api_requests_total", m.apiRequestsTotal)
	writeMetric(&b, "api_errors_total", m.apiErrorsTotal)
	writeMetric(&b, "finalize_requests_total", m.finalizeRequestsTotal)
	writeMetric(&b, "finalize_errors_total", m.finalizeErrorsTotal)
	writeMetric(&b, "finalize_fallback_total", m.finalizeFallbackTotal)
	writeMetric(&b, "finalize_duration_ms_count", m.finalizeDurationCount)
	writeMetric(&b, "finalize_duration_ms_sum", m.finalizeDurationSumMS)
	for _, bucket := range finalizeDurationBuckets {
		fmt.Fprintf(&b, "finalize_duration_ms_bucket{le=\"%d\"} %d\n", bucket, m.finalizeBuckets[bucket])
	}
	writeMetric(&b, "copy_requests_total", m.copyRequestsTotal)
	writeMetric(&b, "copy_failures_total", m.copyFailuresTotal)
	writeMetric(&b, "copy_timeouts_total", m.copyTimeoutsTotal)
	return b.String()
}

func writeMetric(b *strings.Builder, name string, value int64) {
	fmt.Fprintf(b, "%s %d\n", name, value)
}
