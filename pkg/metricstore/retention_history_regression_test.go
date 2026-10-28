// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricstore

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestRetentionSweepsWithinBoundedInterval(t *testing.T) {
	for _, tt := range []struct {
		name, retention string
		wait            time.Duration
	}{
		{"long_window", "168h", 5*time.Minute + time.Second},
		{"short_window", "1m", 31 * time.Second},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				oldStore, oldState, oldKeys := msInstance, state, Keys
				newTestPool(t)
				ms := newTestStore()
				msInstance, state = ms, &GlobalState{}
				Keys.RetentionInMemory = tt.retention
				d, err := time.ParseDuration(tt.retention)
				if err != nil {
					t.Fatal(err)
				}
				stale := time.Now().Add(-d - time.Hour).Unix()
				writeNode(ms, "cluster", "expired", nil, 60, stale, stale+120)
				writeNode(ms, "cluster", "protected", nil, 60, stale, stale+120)
				fresh := time.Now().Unix()
				writeNode(ms, "cluster", "fresh", nil, 60, fresh, fresh+120)
				ms.SetNodeProvider(&fakeNodeProvider{nodes: map[string][]string{"cluster": {"protected"}}})
				ctx, cancel := context.WithCancel(context.Background())
				var wg sync.WaitGroup
				defer func() { cancel(); wg.Wait(); msInstance, state, Keys = oldStore, oldState, oldKeys }()
				Retention(&wg, ctx)
				synctest.Wait()
				time.Sleep(tt.wait)
				synctest.Wait()
				if hasNode(ms, "cluster", "expired") {
					t.Errorf("expired history still retained after %v for retention %s", tt.wait, tt.retention)
				}
				if !hasNode(ms, "cluster", "protected") {
					t.Error("retention removed history needed by a running job")
				}
				if !hasNode(ms, "cluster", "fresh") {
					t.Error("retention removed fresh data")
				}
			})
		})
	}
}

func TestMemoryTrackerPreservesProtectedHistoryBelowCap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		oldStore, oldState, oldKeys := msInstance, state, Keys
		newTestPool(t)
		ms := newTestStore()
		msInstance, state = ms, &GlobalState{}
		Keys.MemoryCap = 100
		Keys.RetentionInMemory = "24h"
		stale := time.Now().Add(-7 * 24 * time.Hour).Unix()
		writeNode(ms, "cluster", "running", nil, 60, stale, stale+120)
		ms.SetNodeProvider(&fakeNodeProvider{nodes: map[string][]string{"cluster": {"running"}}})
		cutoff := time.Now().Add(-24 * time.Hour)
		state.lastRetentionTime = cutoff.Unix()
		if _, err := Free(ms, cutoff); err != nil {
			t.Fatal(err)
		}
		if !hasNode(ms, "cluster", "running") {
			t.Fatal("normal retention removed protected data")
		}
		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		defer func() { cancel(); wg.Wait(); msInstance, state, Keys = oldStore, oldState, oldKeys }()
		MemoryUsageTracker(&wg, ctx)
		synctest.Wait()
		time.Sleep(DefaultMemoryUsageTrackerInterval + time.Second)
		synctest.Wait()
		if !hasNode(ms, "cluster", "running") {
			t.Error("memory tracker removed protected history below its heap cap")
		}
	})
}
