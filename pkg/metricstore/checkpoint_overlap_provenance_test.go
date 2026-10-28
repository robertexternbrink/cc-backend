// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricstore

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
	"github.com/ClusterCockpit/cc-lib/v2/util"
)

func newHistoryTestStore() *MemoryStore {
	return &MemoryStore{Metrics: map[string]MetricConfig{"metric": {Frequency: 10, Aggregation: AvgAggregation}}}
}

func TestCheckpointOverlayPreservesSurvivingRanges(t *testing.T) {
	const start = int64(1700000000)
	m := newHistoryTestStore()
	level := m.GetLevel([]string{"cluster", "host"})
	for _, metric := range []*CheckpointMetrics{
		{Frequency: 10, Start: start, Data: []schema.Float{10, 11, 12, 13, 14, 15, 16}},
		{Frequency: 10, Start: start + 20, Data: []schema.Float{72, 73}},
		{Frequency: 10, Start: start - 10, Data: []schema.Float{9, 90}},
	} {
		if err := level.loadFile(&CheckpointFile{Metrics: map[string]*CheckpointMetrics{"metric": metric}}, m); err != nil {
			t.Fatal(err)
		}
	}
	assertCheckpointValues(t, m, start-10, []schema.Float{9, 90, 11, 72, 73, 14, 15, 16})
}

func assertCheckpointValues(t *testing.T, m *MemoryStore, start int64, want []schema.Float) {
	t.Helper()
	data, from, to, _, err := m.Read(util.Selector{{String: "cluster"}, {String: "host"}}, "metric", start, start+int64(len(want))*10, 0, "")
	if err != nil || from != start || to != start+int64(len(want))*10 || len(data) != len(want) {
		t.Fatalf("restored range: %d..%d, %d values, %v", from, to, len(data), err)
	}
	for i, expected := range want {
		if data[i] != expected && !(math.IsNaN(float64(data[i])) && math.IsNaN(float64(expected))) {
			t.Fatalf("sample %d: got %v, want %v", i, data[i], expected)
		}
	}
}

// This mirrors recovery from the preserved pre-restart checkpoint/WAL backup:
// replay may fill a gap or overwrite a buffer older than the latest snapshot.
func TestWALRecoveryOlderLinksAndGaps(t *testing.T) {
	const start = int64(1700000000)
	{
		{
			m := newHistoryTestStore()
			level := m.GetLevel([]string{"cluster", "host"})
			for _, metric := range []*CheckpointMetrics{
				{Frequency: 10, Start: start, Data: []schema.Float{10, 11, 12}},
				{Frequency: 10, Start: start + 50, Data: []schema.Float{15, 16}},
			} {
				if err := level.loadFile(&CheckpointFile{Metrics: map[string]*CheckpointMetrics{"metric": metric}}, m); err != nil {
					t.Fatal(err)
				}
			}
			var wal bytes.Buffer
			binary.Write(&wal, binary.LittleEndian, walFileMagic)
			w := bufio.NewWriter(&wal)
			for _, sample := range []struct{ index, value int }{{1, 77}, {3, 13}, {4, 14}, {6, 66}} {
				if _, err := writeWALRecordDirect(w, &WALMessage{MetricName: "metric", Timestamp: start + int64(sample.index)*10, Value: schema.Float(sample.value)}); err != nil {
					t.Fatal(err)
				}
			}
			w.Flush()
			filename := filepath.Join(t.TempDir(), "current.wal")
			if err := os.WriteFile(filename, wal.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			f, err := os.Open(filename)
			if err != nil {
				t.Fatal(err)
			}
			err = level.loadWALFile(m, f, 0)
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
			assertCheckpointValues(t, m, start, []schema.Float{10, 77, 12, 13, 14, 15, 66})
		}
	}
}
