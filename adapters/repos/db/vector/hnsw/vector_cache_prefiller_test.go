//                           _       _
// __      _____  __ ___   ___  __ _| |_ ___
// \ \ /\ / / _ \/ _` \ \ / / |/ _` | __/ _ \
//  \ V  V /  __/ (_| |\ V /| | (_| | ||  __/
//   \_/\_/ \___|\__,_| \_/ |_|\__,_|\__\___|
//
//  Copyright © 2016 - 2023 Weaviate B.V. All rights reserved.
//
//  CONTACT: hello@weaviate.io
//

package hnsw

import (
	"context"
	"sync"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestVectorCachePrefilling(t *testing.T) {
	cache := newFakeCache()
	index := &hnsw{
		nodes:               generateDummyVertices(100),
		currentMaximumLayer: 3,
		shardedNodeLocks:    make([]sync.RWMutex, NodeLockStripe),
	}
	for i := uint64(0); i < NodeLockStripe; i++ {
		index.shardedNodeLocks[i] = sync.RWMutex{}
	}

	logger, _ := test.NewNullLogger()

	pf := newVectorCachePrefiller[float32](cache, index, logger)

	t.Run("prefill with limit >= graph size", func(t *testing.T) {
		cache.reset()
		pf.Prefill(context.Background(), 100)
		assert.Equal(t, allNumbersUpTo(100), cache.store)
	})

	t.Run("prefill with small limit so only the upper layer fits", func(t *testing.T) {
		cache.reset()
		pf.Prefill(context.Background(), 7)
		assert.Equal(t, map[uint64]struct{}{
			0:  {},
			15: {},
			30: {},
			45: {},
			60: {},
			75: {},
			90: {},
		}, cache.store)
	})

	t.Run("limit where a layer partially fits", func(t *testing.T) {
		cache.reset()
		pf.Prefill(context.Background(), 10)
		assert.Equal(t, map[uint64]struct{}{
			// layer 3
			0:  {},
			15: {},
			30: {},
			45: {},
			60: {},
			75: {},
			90: {},

			// additional layer 2
			5:  {},
			10: {},
			20: {},
		}, cache.store)
	})
}

func newFakeCache() *fakeCache {
	return &fakeCache{
		store: map[uint64]struct{}{},
	}
}

//nolint:unused
func (f *fakeCache) all() [][]float32 {
	return nil
}

type fakeCache struct {
	store map[uint64]struct{}
}

//nolint:unused
func (f *fakeCache) get(ctx context.Context, id uint64) ([]float32, error) {
	f.store[id] = struct{}{}
	return nil, nil
}

//nolint:unused
func (f *fakeCache) delete(ctx context.Context, id uint64) {
	panic("not implemented")
}

//nolint:unused
func (f *fakeCache) preload(id uint64, vec []float32) {
	panic("not implemented")
}

//nolint:unused
func (f *fakeCache) prefetch(id uint64) {
	panic("not implemented")
}

//nolint:unused
func (f *fakeCache) grow(id uint64) {
	panic("not implemented")
}

//nolint:unused
func (f *fakeCache) updateMaxSize(size int64) {
	panic("not implemented")
}

//nolint:unused
func (f *fakeCache) drop() {
	panic("not implemented")
}

//nolint:unused
func (f *fakeCache) copyMaxSize() int64 {
	return 1e6
}

func (f *fakeCache) reset() {
	f.store = map[uint64]struct{}{}
}

//nolint:unused
func (f *fakeCache) len() int32 {
	return int32(len(f.store))
}

//nolint:unused
func (f *fakeCache) countVectors() int64 {
	panic("not implemented")
}

func generateDummyVertices(amount int) []*vertex {
	out := make([]*vertex, amount)
	for i := range out {
		out[i] = &vertex{
			id:    uint64(i),
			level: levelForDummyVertex(i),
		}
	}

	return out
}

// maximum of 3 layers
// if id % 15 == 0 -> layer 3
// if id % 5 == 0 -> layer 2
// if id % 3 == 0 -> layer 1
// remainder -> layer 0
func levelForDummyVertex(id int) int {
	if id%15 == 0 {
		return 3
	}

	if id%5 == 0 {
		return 2
	}

	if id%3 == 0 {
		return 1
	}

	return 0
}

func allNumbersUpTo(size int) map[uint64]struct{} {
	out := map[uint64]struct{}{}
	for i := 0; i < size; i++ {
		out[uint64(i)] = struct{}{}
	}

	return out
}
