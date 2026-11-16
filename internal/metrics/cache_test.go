package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacheMetrics(t *testing.T) {
	var m CacheMetrics

	m.RecordHit()
	m.RecordHit()
	m.RecordMiss()
	m.RecordError()

	stats := m.Snapshot()

	assert.Equal(t, uint64(2), stats.Hits)
	assert.Equal(t, uint64(1), stats.Misses)
	assert.Equal(t, uint64(1), stats.Errors)
}

func TestCacheStatsHitRate(t *testing.T) {
	tests := []struct {
		name  string
		stats CacheStats
		want  float64
	}{
		{
			name:  "no requests",
			stats: CacheStats{},
			want:  0,
		},
		{
			name:  "all hits",
			stats: CacheStats{Hits: 10},
			want:  100,
		},
		{
			name:  "all misses",
			stats: CacheStats{Misses: 10},
			want:  0,
		},
		{
			name: "mixed results",
			stats: CacheStats{
				Hits:   80,
				Misses: 15,
				Errors: 5,
			},
			want: 80,
		},
		{
			name: "fractional percentage",
			stats: CacheStats{
				Hits:   1,
				Misses: 2,
			},
			want: 100.0 / 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.stats.HitRate()

			assert.InDelta(t, tt.want, got, 0.000001)
		})
	}
}

func TestAverageRedisGETLatency(t *testing.T) {
	var m CacheMetrics

	m.RecordRedisGETDuration(1 * time.Millisecond)
	m.RecordRedisGETDuration(2 * time.Millisecond)
	m.RecordRedisGETDuration(3 * time.Millisecond)

	stats := m.Snapshot()

	assert.Equal(t, uint64(3), stats.RedisGETCount)

	assert.InDelta(
		t,
		6_000_000,
		float64(stats.RedisGETTotalNanos),
		0.001,
	)

	assert.InDelta(
		t,
		2.0,
		stats.AverageRedisGETLatencyMS(),
		0.000001,
	)
}

func TestAverageRedisGETLatencyInitiallyZero(t *testing.T) {
	stats := CacheStats{}

	assert.Zero(t, stats.AverageRedisGETLatencyMS())
}
