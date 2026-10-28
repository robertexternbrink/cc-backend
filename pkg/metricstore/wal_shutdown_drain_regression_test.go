// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricstore

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

// Gate the worker's first select until the fixture has queued accepted records.
// This models cancellation with a backlog without sleeps or an external writer.
type walDrainGateContext struct {
	context.Context
	ready, release chan struct{}
	once           sync.Once
}

func (c *walDrainGateContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.ready) })
	<-c.release
	return c.Context.Done()
}

func TestWALCancellationDrainsAcceptedRecords(t *testing.T) {
	checkpointTestConfig(t)
	previousChannels, previousShards := walShardChs, walNumShards
	previousStopping := walShuttingDown.Load()
	t.Cleanup(func() {
		walShardChs, walNumShards = previousChannels, previousShards
		walShuttingDown.Store(previousStopping)
	})
	// More than one normal drain batch ensures cancellation is exercised while
	// queued records remain. Repeat select scheduling; every trial must persist
	// the entire accepted queue even when the final snapshot is unavailable.
	for trial := range 8 {
		t.Run(fmt.Sprint(trial), func(t *testing.T) {
			Keys.Checkpoints.RootDir = t.TempDir()
			walShuttingDown.Store(false)
			ctx, cancel := context.WithCancel(context.Background())
			gate := &walDrainGateContext{Context: ctx, ready: make(chan struct{}), release: make(chan struct{})}
			var workers sync.WaitGroup
			WALStaging(&workers, gate)
			<-gate.ready
			const records = 8192
			for i := range records {
				if !SendWALMessage(&WALMessage{MetricName: "metric", Cluster: "cluster", Node: "host", Timestamp: 1700000000 + int64(i)*10, Value: schema.Float(i + 1)}) {
					t.Fatal("fixture did not enqueue every record")
				}
			}
			cancel()
			close(walShardChs[0])
			close(gate.release)
			workers.Wait()
			// No final checkpoint is created: the WAL must retain every accepted
			// record even if writing a snapshot after shutdown fails.
			path := filepath.Join(Keys.Checkpoints.RootDir, "cluster", "host", "current.wal")
			f, err := os.Open(path)
			if err != nil {
				t.Errorf("accepted WAL backlog was not persisted: %v", err)
				return
			}
			defer f.Close()
			reader := bufio.NewReader(f)
			var magic uint32
			if err := binary.Read(reader, binary.LittleEndian, &magic); err != nil || magic != walFileMagic {
				t.Fatalf("WAL header: %x %v", magic, err)
			}
			count := 0
			for {
				msg, err := readWALRecord(reader)
				if err != nil {
					t.Fatal(err)
				}
				if msg == nil {
					break
				}
				if msg.Timestamp != 1700000000+int64(count)*10 || msg.Value != schema.Float(count+1) {
					t.Fatalf("record %d changed", count)
				}
				count++
			}
			if count != records {
				t.Errorf("cancellation discarded accepted WAL records: got%d want%d", count, records)
			}
		})
	}
}
