// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
	"github.com/ClusterCockpit/cc-line-protocol/v2/lineprotocol"
)

func checkpointTestConfig(t *testing.T) {
	t.Helper()
	previous := Keys
	Keys.NumWorkers = 1
	Keys.Checkpoints.FileFormat = "wal"
	Keys.Checkpoints.MaxWALSize = 0
	t.Cleanup(func() { Keys = previous })
}

// Startup's wall-clock checkpoint cursor must not exclude accepted backfills,
// restored-head overwrites, or future-dated writes when rotating the WAL.
func TestCheckpointDirtyCoverageBeforeCursorAndAfterTo(t *testing.T) {
	checkpointTestConfig(t)
	const start = int64(1700000000)
	for _, format := range []string{"json", "wal"} {
		t.Run(format, func(t *testing.T) {
			checkpoint := func(m *MemoryStore, dir string, from, to int64) (int, error) {
				if format == "json" {
					return m.ToCheckpoint(dir, from, to)
				}
				n, _, err := m.checkpointWAL(dir, from, to, RotateWALFilesAfterShutdown)
				return n, err
			}
			dir := t.TempDir()
			m := newHistoryTestStore()
			path := []string{"cluster", "host"}
			for i := range 3 {
				if err := m.Write(path, start+int64(i)*10, []Metric{{Name: "metric", Value: schema.Float(10 + i)}}); err != nil {
					t.Fatal(err)
				}
			}
			if n, err := checkpoint(m, dir, start, start+30); err != nil || n != 1 {
				t.Fatalf("initial checkpoint: %d, %v", n, err)
			}
			m = newHistoryTestStore()
			if _, err := m.FromCheckpoint(dir, 0); err != nil {
				t.Fatal(err)
			}
			for _, sample := range []struct{ index, value int }{{1, 77}, {3, 13}, {4, 14}, {5, 15}, {6, 16}, {9, 19}} {
				if err := m.Write(path, start+int64(sample.index)*10, []Metric{{Name: "metric", Value: schema.Float(sample.value)}}); err != nil {
					t.Fatal(err)
				}
			}
			// The cursor is deliberately off the sample grid and later than
			// two accepted backfills. The requested end precedes a live value.
			if n, err := checkpoint(m, dir, start+57, start+70); err != nil || n != 1 {
				t.Fatalf("dirty checkpoint: %d, %v", n, err)
			}
			restored := newHistoryTestStore()
			if _, err := restored.FromCheckpoint(dir, 0); err != nil {
				t.Fatal(err)
			}
			assertCheckpointValues(t, restored, start, []schema.Float{10, 77, 12, 13, 14, 15, 16, schema.NaN, schema.NaN, 19})
			if n, err := checkpoint(m, dir, start+100, start+110); err != nil || n != 0 {
				t.Fatalf("unchanged data was copied again: %d, %v", n, err)
			}
		})
	}
}

func TestCheckpointFailureRetainsDirtyCoverage(t *testing.T) {
	checkpointTestConfig(t)
	m := newHistoryTestStore()
	if err := m.Write([]string{"cluster", "host"}, 1700000000, []Metric{{Name: "metric", Value: 42}}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cluster"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if n, hosts, err := m.ToCheckpointWAL(dir, 1700000100, 1700000200); err == nil || n != 0 || len(hosts) != 0 {
		t.Fatalf("failed snapshot became eligible for rotation: %d, %v, %v", n, hosts, err)
	}
	b := m.GetLevel([]string{"cluster", "host"}).metrics[0]
	if b.archived {
		t.Fatal("failed snapshot cleared dirty state")
	}
	if err := os.Remove(filepath.Join(dir, "cluster")); err != nil {
		t.Fatal(err)
	}
	if n, _, err := m.ToCheckpointWAL(dir, 1700000100, 1700000200); err != nil || n != 1 {
		t.Fatalf("retry: %d, %v", n, err)
	}
}

func TestCheckpointFencesWritesAndWALRotationDrainsFIFO(t *testing.T) {
	checkpointTestConfig(t)
	previousChs, previousShards := walShardChs, walNumShards
	previousStopping := walShuttingDown.Load()
	t.Cleanup(func() {
		walShardChs, walNumShards = previousChs, previousShards
		walShuttingDown.Store(previousStopping)
	})
	m := newHistoryTestStore()
	Keys.Checkpoints.RootDir = t.TempDir()
	walShuttingDown.Store(false)
	ctx, cancel := context.WithCancel(context.Background())
	var workers sync.WaitGroup
	WALStaging(&workers, ctx)
	const start = int64(1700000000)
	for i := range 2000 {
		line := fmt.Sprintf("metric,cluster=cluster,hostname=host value=%d %d", i, start+int64(i)*10)
		if err := DecodeLine(lineprotocol.NewDecoderWithBytes([]byte(line)), m, ""); err != nil {
			t.Fatal(err)
		}
	}
	writerDone := make(chan error, 1)
	n, _, err := m.checkpointWAL(Keys.Checkpoints.RootDir, start+17000, start+20000, func(hosts []string) {
		go func() {
			line := fmt.Sprintf("metric,cluster=cluster,hostname=host value=2000 %d", start+20000)
			writerDone <- DecodeLine(lineprotocol.NewDecoderWithBytes([]byte(line)), m, "")
		}()
		select {
		case err := <-writerDone:
			writerDone <- err
			t.Error("write crossed the snapshot/rotation fence")
		case <-time.After(20 * time.Millisecond):
		}
		RotateWALFiles(hosts)
		for _, host := range hosts {
			if _, err := os.Stat(filepath.Join(host, "current.wal")); !os.IsNotExist(err) {
				t.Errorf("FIFO rotation left old WAL records: %v", err)
			}
		}
	})
	if err != nil || n != 1 {
		t.Fatalf("checkpoint: %d, %v", n, err)
	}
	if err := <-writerDone; err != nil {
		t.Fatal(err)
	}
	cancel()
	workers.Wait()
	restored := newHistoryTestStore()
	if _, err := restored.FromCheckpoint(Keys.Checkpoints.RootDir, 0); err != nil {
		t.Fatal(err)
	}
	want := make([]schema.Float, 2001)
	for i := range want {
		want[i] = schema.Float(i)
	}
	assertCheckpointValues(t, restored, start, want)
}

// Shutdown must not close the FIFO while an earlier snapshot can still enqueue
// its rotation barrier. It must wait for that checkpoint to finish first.
func TestShutdownWaitsForCheckpointBeforeClosingWAL(t *testing.T) {
	checkpointTestConfig(t)
	previousStore, previousChannels := msInstance, walShardChs
	previousStopping := walShuttingDown.Load()
	shutdownFuncMu.Lock()
	previousShutdown := shutdownFunc
	cancelled := make(chan struct{})
	shutdownFunc = func() { close(cancelled) }
	shutdownFuncMu.Unlock()
	t.Cleanup(func() {
		msInstance, walShardChs = previousStore, previousChannels
		walShuttingDown.Store(previousStopping)
		shutdownFuncMu.Lock()
		shutdownFunc = previousShutdown
		shutdownFuncMu.Unlock()
	})
	m := newHistoryTestStore()
	msInstance = m
	Keys.Checkpoints.RootDir = t.TempDir()
	walShardChs = []chan *WALMessage{make(chan *WALMessage, 1)}
	walShuttingDown.Store(false)
	m.checkpointMu.Lock()
	done := make(chan struct{})
	go func() { Shutdown(); close(done) }()
	<-cancelled
	deadline := time.Now().Add(20 * time.Millisecond)
	for time.Now().Before(deadline) && !walShuttingDown.Load() {
		time.Sleep(time.Millisecond)
	}
	closedTooSoon := walShuttingDown.Load()
	m.checkpointMu.Unlock()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not finish after the checkpoint released its fence")
	}
	if closedTooSoon {
		t.Fatal("WAL shutdown crossed an active checkpoint/rotation fence")
	}
}
