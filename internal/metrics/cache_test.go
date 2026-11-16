package metrics

import (
	"testing"

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
