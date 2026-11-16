package metrics

import "sync/atomic"

type CacheMetrics struct {
	hits   atomic.Uint64
	misses atomic.Uint64
	errors atomic.Uint64
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
}

func (m *CacheMetrics) Snapshot() CacheStats {
	return CacheStats{
		Hits:   m.hits.Load(),
		Misses: m.misses.Load(),
		Errors: m.errors.Load(),
	}
}
