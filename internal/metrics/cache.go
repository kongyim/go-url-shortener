package metrics

import (
	"sync/atomic"
	"time"
)

type CacheMetrics struct {
	hits   atomic.Uint64
	misses atomic.Uint64
	errors atomic.Uint64

	redisGETCount      atomic.Uint64
	redisGETTotalNanos atomic.Uint64

	postgresLookupCount      atomic.Uint64
	postgresLookupTotalNanos atomic.Uint64
}

func (m *CacheMetrics) RecordHit() {
	m.hits.Add(1)
}

func (m *CacheMetrics) RecordMiss() {
	m.misses.Add(1)
}

func (m *CacheMetrics) RecordError() {
	m.errors.Add(1)
}

type CacheStats struct {
	Hits   uint64
	Misses uint64
	Errors uint64

	RedisGETCount      uint64
	RedisGETTotalNanos uint64

	PostgresLookupCount      uint64
	PostgresLookupTotalNanos uint64
}

func (m *CacheMetrics) Snapshot() CacheStats {
	return CacheStats{
		Hits:   m.hits.Load(),
		Misses: m.misses.Load(),
		Errors: m.errors.Load(),

		RedisGETCount:            m.redisGETCount.Load(),
		RedisGETTotalNanos:       m.redisGETTotalNanos.Load(),
		PostgresLookupCount:      m.postgresLookupCount.Load(),
		PostgresLookupTotalNanos: m.postgresLookupTotalNanos.Load(),
	}
}

func (s CacheStats) HitRate() float64 {
	total := s.Hits + s.Misses + s.Errors

	if total == 0 {
		return 0
	}

	return float64(s.Hits) / float64(total) * 100
}

func (s CacheStats) AverageRedisGETLatencyMS() float64 {
	if s.RedisGETCount == 0 {
		return 0
	}

	totalMilliseconds := float64(s.RedisGETTotalNanos) / 1_000_000

	return totalMilliseconds / float64(s.RedisGETCount)
}

func (m *CacheMetrics) RecordRedisGETDuration(d time.Duration) {
	if d < 0 {
		return
	}

	m.redisGETCount.Add(1)
	m.redisGETTotalNanos.Add(uint64(d.Nanoseconds()))
}

func (s CacheStats) AveragePostgresLookupLatencyMS() float64 {
	if s.PostgresLookupCount == 0 {
		return 0
	}
	return float64(s.PostgresLookupTotalNanos) / 1_000_000 / float64(s.PostgresLookupCount)
}

func (m *CacheMetrics) RecordPostgresLookupDuration(d time.Duration) {
	if d < 0 {
		return
	}
	m.postgresLookupCount.Add(1)
	m.postgresLookupTotalNanos.Add(uint64(d.Nanoseconds()))
}
