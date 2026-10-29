// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricstore

import (
	"sync"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func TestPersistentPoolBoundsConcurrentReturns(t *testing.T) {
	pool := NewPersistentBufferPool()
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 2560 {
				pool.Put(&buffer{data: make([]schema.Float, 0, BufferCap)})
			}
		})
	}
	wg.Wait()
	// The deployed bound allows roughly 64 MiB of idle sample arrays.
	if n := pool.GetSize(); n > 16*1024 || n == 0 {
		t.Fatalf("idle pool retained %d buffers, want at most 16384 reusable arrays", n)
	}
	for pool.GetSize() > 0 {
		if b := pool.Get(); cap(b.data) != BufferCap || len(b.data) != 0 {
			t.Fatal("invalid reusable buffer")
		}
	}
}

func TestPersistentPoolDiscardsRestoredArraysAndLinks(t *testing.T) {
	pool := NewPersistentBufferPool()
	b := &buffer{data: make([]schema.Float, 1, BufferCap), prev: &buffer{}, next: &buffer{}}
	pool.Put(b)
	if b.prev != nil || b.next != nil {
		t.Error("idle buffer retains another buffer chain")
	}
	if got := pool.Get(); got != b || len(got.data) != 0 {
		t.Fatal("standard buffer not reset and reused")
	}
	restored := &buffer{data: make([]schema.Float, 720)}
	allocations := testing.AllocsPerRun(5, func() { pool.Put(restored) })
	if allocations != 0 || pool.GetSize() != 0 {
		t.Fatalf("discarding restored array allocated %v objects and retained %d buffers", allocations, pool.GetSize())
	}
}

func TestFreeDiscardsNonstandardArraysWithoutReplacement(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(map[bool]string{false: "retention", true: "emergency"}[force], func(t *testing.T) {
			pool := newTestPool(t)
			ms := &MemoryStore{Metrics: map[string]MetricConfig{"metric": {Frequency: 60}}}
			l := ms.GetLevel([]string{"cluster", "node"})
			l.metrics[0] = &buffer{data: make([]schema.Float, 720), frequency: 60, start: 1700000000}
			var n int
			var err error
			if force {
				n, err = l.forceFree()
			} else {
				n, err = ms.Free(nil, 1700100000)
			}
			if err != nil || n != 1 {
				t.Fatalf("free returned %d, %v", n, err)
			}
			if pool.GetSize() != 0 {
				t.Fatal("free retained a newly allocated replacement for discarded checkpoint data")
			}
		})
	}
}
