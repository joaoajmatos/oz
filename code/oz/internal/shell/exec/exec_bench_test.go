package exec_test

import (
	"os/exec"
	"sort"
	"testing"
	"time"
)

// BenchmarkExecRoundTrip measures subprocess fork+capture baseline.
func BenchmarkExecRoundTrip(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("echo", "hello world")
		if _, err := cmd.CombinedOutput(); err != nil {
			b.Fatalf("combined output: %v", err)
		}
	}
}

// BenchmarkExecRoundTrip_P95 reports p50/p95 subprocess latency in ms.
func BenchmarkExecRoundTrip_P95(b *testing.B) {
	latencies := make([]float64, 0, b.N)
	for i := 0; i < b.N; i++ {
		start := time.Now()
		cmd := exec.Command("echo", "hello world")
		if _, err := cmd.CombinedOutput(); err != nil {
			b.Fatalf("combined output: %v", err)
		}
		latencies = append(latencies, float64(time.Since(start).Microseconds())/1000.0)
	}

	if len(latencies) == 0 {
		return
	}
	sort.Float64s(latencies)
	p50 := percentile(latencies, 0.50)
	p95 := percentile(latencies, 0.95)
	b.ReportMetric(p50, "p50_ms")
	b.ReportMetric(p95, "p95_ms")
}

func percentile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	idx := int(float64(len(values)-1) * q)
	return values[idx]
}
