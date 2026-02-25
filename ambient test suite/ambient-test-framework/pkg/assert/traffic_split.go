// pkg/assert/traffic_split.go
package assert

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/yourorg/ambient-test-framework/pkg/traffic"
)

// SplitResult holds the result of a traffic split measurement.
type SplitResult struct {
	TotalRequests int64
	Hits          map[string]int64 // key -> count
}

// Percentage returns the percentage of requests that hit key.
func (r *SplitResult) Percentage(key string) float64 {
	if r.TotalRequests == 0 {
		return 0
	}
	return float64(r.Hits[key]) / float64(r.TotalRequests) * 100
}

// MeasureTrafficSplit sends totalRequests to endpoint and counts responses
// matching each key in the JSON body field "message".
func MeasureTrafficSplit(ctx context.Context,
	client *http.Client,
	endpoint traffic.Endpoint,
	totalRequests, concurrency int) (*SplitResult, error) {

	if client == nil {
		client = &http.Client{}
	}

	hits := sync.Map{}
	var total atomic.Int64
	var wg sync.WaitGroup

	perWorker := totalRequests / concurrency
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				resp, err := traffic.HTTPGetJSON(ctx, client, endpoint)
				if err != nil {
					continue
				}
				total.Add(1)
				if msg, ok := resp["message"].(string); ok {
					cur, _ := hits.LoadOrStore(msg, new(atomic.Int64))
					cur.(*atomic.Int64).Add(1)
				}
			}
		}()
	}
	wg.Wait()

	result := &SplitResult{
		TotalRequests: total.Load(),
		Hits:          make(map[string]int64),
	}
	hits.Range(func(k, v interface{}) bool {
		result.Hits[k.(string)] = v.(*atomic.Int64).Load()
		return true
	})
	return result, nil
}

// TrafficSplitWithinTolerance asserts that actual traffic percentages
// are within tolerance of the expected weights.
func TrafficSplitWithinTolerance(t *testing.T,
	result *SplitResult,
	expected map[string]float64,
	tolerance float64) {

	t.Helper()
	if result.TotalRequests == 0 {
		t.Fatal("no successful responses received — cannot measure traffic split")
	}
	for key, expectedPct := range expected {
		actualPct := result.Percentage(key)
		diff := expectedPct - actualPct
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			t.Errorf("traffic split mismatch for %q: expected=%.1f%%, got=%.1f%% (diff=%.1f%%, tolerance=%.1f%%)",
				key, expectedPct, actualPct, diff, tolerance)
		} else {
			t.Logf("traffic split OK for %q: expected=%.1f%%, got=%.1f%%",
				key, expectedPct, actualPct)
		}
	}
	t.Logf("total requests: %d, distribution: %s",
		result.TotalRequests, formatHits(result))
}

func formatHits(r *SplitResult) string {
	s := "{"
	first := true
	for k, v := range r.Hits {
		if !first {
			s += ", "
		}
		s += fmt.Sprintf("%s: %d (%.1f%%)", k, v, r.Percentage(k))
		first = false
	}
	return s + "}"
}
