// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricstore

import (
	"slices"
	"strconv"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
	"github.com/ClusterCockpit/cc-lib/v2/util"
)

// TestCheckpointRoundTripPreservesSampleTimes verifies that restoring successive
// checkpoints preserves both data and timestamps, including odd frequencies.
func TestCheckpointRoundTripPreservesSampleTimes(t *testing.T) {
	for _, frequency := range []int64{5, 10, 60} {
		t.Run(strconv.FormatInt(frequency, 10), func(t *testing.T) {
			const from = int64(1700000000)
			selector := []string{"local", "node"}
			query := util.Selector{{String: "local"}, {String: "node"}}
			newStore := func() *MemoryStore {
				return &MemoryStore{Metrics: map[string]MetricConfig{
					"audit": {Frequency: frequency, Aggregation: AvgAggregation},
				}}
			}
			store := newStore()
			var checkpoints []*CheckpointFile
			for i := range 12 {
				if err := store.Write(selector, from+int64(i)*frequency,
					[]Metric{{Name: "audit", Value: schema.Float(i + 1)}}); err != nil {
					t.Fatal(err)
				}
			}
			for cycle := range 4 {
				to := from + int64(12+cycle)*frequency
				before, startBefore, endBefore, _, err := store.Read(query, "audit", from-frequency/3, to, 0, "")
				if err != nil {
					t.Fatal(err)
				}
				checkpoint, err := store.GetLevel(selector).toCheckpointFile(from, to, store)
				if err != nil || checkpoint == nil {
					t.Fatalf("creating checkpoint: %v", err)
				}
				checkpoints = append(checkpoints, checkpoint)
				restored := newStore()
				// Checkpoints contain newly dirty ranges; restore the retained
				// sequence just as FromCheckpoint does on an actual restart.
				for _, previous := range checkpoints {
					if err := restored.GetLevel(selector).loadFile(previous, restored); err != nil {
						t.Fatal(err)
					}
				}
				after, startAfter, endAfter, _, err := restored.Read(query, "audit", from-frequency/3, to, 0, "")
				if err != nil {
					t.Fatal(err)
				}
				if startBefore != startAfter || endBefore != endAfter || !slices.EqualFunc(before, after, func(a, b schema.Float) bool { return a == b || (a.IsNaN() && b.IsNaN()) }) {
					t.Fatalf("cycle %d changed samples: before [%d,%d) %v; after [%d,%d) %v",
						cycle, startBefore, endBefore, before, startAfter, endAfter, after)
				}
				if err := restored.Write(selector, to, []Metric{{Name: "audit", Value: schema.Float(13 + cycle)}}); err != nil {
					t.Fatal(err)
				}
				store = restored
			}
		})
	}
}
