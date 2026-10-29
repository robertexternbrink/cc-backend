// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricstore

import (
	"context"
	"testing"
	"time"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func TestMissingFirstSeriesPreservesDeviceIdentity(t *testing.T) {
	initDeviceArchive(t)
	ms := newDeviceTestStore()
	const start = int64(1800000000)
	for ts := start; ts < start+240; ts += 60 {
		if err := ms.Write([]string{deviceCluster, "a01", "filesystem/scratch"}, ts,
			[]Metric{{Name: "fs_read_bw", Value: 7}}); err != nil {
			t.Fatal(err)
		}
	}
	old := msInstance
	msInstance = ms
	t.Cleanup(func() { msInstance = old })
	job := deviceJob("a", "a01")
	job.StartTime, job.Duration = start, 240
	store := &InternalMetricStore{}
	check := func(name string, series []schema.Series) {
		t.Helper()
		if len(series) != 1 || series[0].ID == nil || *series[0].ID != "/scratch" {
			t.Errorf("%s returned %v; want only the actual /scratch device", name, series)
		}
	}
	data, err := store.LoadData(job, []string{"fs_read_bw"}, []schema.MetricScope{schema.MetricScopeFilesystem}, context.Background(), 60, "")
	if err != nil {
		t.Fatal(err)
	}
	check("LoadData", data.Metrics["fs_read_bw"][schema.MetricScopeFilesystem].Series)
	stats, err := store.LoadScopedStats(job, []string{"fs_read_bw"}, []schema.MetricScope{schema.MetricScopeFilesystem}, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	scoped := stats.Metrics["fs_read_bw"][schema.MetricScopeFilesystem]
	if len(scoped) != 1 || scoped[0].ID == nil || *scoped[0].ID != "/scratch" || scoped[0].Data == nil || scoped[0].Data.Avg != 7 {
		t.Errorf("LoadScopedStats returned %+v; want /scratch with mean 7", scoped)
	}
	nodes, err := store.LoadNodeListData(deviceCluster, "a", []string{"a01"}, []string{"fs_read_bw"},
		[]schema.MetricScope{schema.MetricScopeFilesystem}, 60, time.Unix(start, 0), time.Unix(start+240, 0), context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	check("LoadNodeListData", nodes["a01"].Metrics["fs_read_bw"][schema.MetricScopeFilesystem].Series)
}
