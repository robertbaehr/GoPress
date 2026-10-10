package wp

import "sync"

// MetadataLazyloader queues object IDs for batch meta cache priming (P2.17).
// Mirrors WP_Metadata_Lazyloader: callers QueueObjects during a loop, then the
// first GetMetadata for that type flushes via UpdateMetaCache.
type MetadataLazyloader struct {
	mu      sync.Mutex
	meta    *Meta
	pending map[MetaType]map[int64]struct{}
}

// NewMetadataLazyloader binds a lazyloader to meta. meta may be nil (no-op flush).
func NewMetadataLazyloader(meta *Meta) *MetadataLazyloader {
	return &MetadataLazyloader{
		meta:    meta,
		pending: make(map[MetaType]map[int64]struct{}),
	}
}

// QueueObjects adds object IDs for later batch UpdateMetaCache.
func (l *MetadataLazyloader) QueueObjects(metaType MetaType, objectIDs []int64) {
	if l == nil || !validMetaType(metaType) || len(objectIDs) == 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket := l.pending[metaType]
	if bucket == nil {
		bucket = make(map[int64]struct{})
		l.pending[metaType] = bucket
	}
	for _, id := range objectIDs {
		if id > 0 {
			bucket[id] = struct{}{}
		}
	}
}

// ResetQueue clears the pending queue for metaType.
func (l *MetadataLazyloader) ResetQueue(metaType MetaType) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.pending, metaType)
}

// PendingCount returns how many IDs are queued for metaType (tests / diagnostics).
func (l *MetadataLazyloader) PendingCount(metaType MetaType) int {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.pending[metaType])
}

// Flush runs UpdateMetaCache for queued IDs (plus optional extraID) and resets the queue.
func (l *MetadataLazyloader) Flush(metaType MetaType, extraID int64) {
	if l == nil || !validMetaType(metaType) {
		return
	}
	l.mu.Lock()
	bucket := l.pending[metaType]
	ids := make([]int64, 0, len(bucket)+1)
	for id := range bucket {
		ids = append(ids, id)
	}
	delete(l.pending, metaType)
	l.mu.Unlock()

	if extraID > 0 {
		found := false
		for _, id := range ids {
			if id == extraID {
				found = true
				break
			}
		}
		if !found {
			ids = append(ids, extraID)
		}
	}
	if len(ids) == 0 || l.meta == nil {
		return
	}
	_, _ = l.meta.UpdateMetaCache(metaType, ids)
}
