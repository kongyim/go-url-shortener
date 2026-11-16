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

func TestPostgresLookupLatency(t *testing.T) {
	for _, tt := range []struct {
		name        string
		durations   []time.Duration
		wantCount   uint64
		wantNanos   uint64
		wantAverage float64
	}{
		{name: "initially zero"},
		{name: "fractional average", durations: []time.Duration{1500 * time.Microsecond, 3 * time.Millisecond}, wantCount: 2, wantNanos: 4_500_000, wantAverage: 2.25},
		{name: "zero duration counts", durations: []time.Duration{0}, wantCount: 1},
		{name: "negative duration ignored", durations: []time.Duration{-time.Millisecond, 2 * time.Millisecond}, wantCount: 1, wantNanos: 2_000_000, wantAverage: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var m CacheMetrics
			for _, d := range tt.durations {
				m.RecordPostgresLookupDuration(d)
			}
			stats := m.Snapshot()
			assert.Equal(t, tt.wantCount, stats.PostgresLookupCount)
			assert.Equal(t, tt.wantNanos, stats.PostgresLookupTotalNanos)
			assert.InDelta(t, tt.wantAverage, stats.AveragePostgresLookupLatencyMS(), 0.000001)
			assert.Zero(t, stats.RedisGETCount)
			assert.Zero(t, stats.RedisGETTotalNanos)
		})
	}
}
