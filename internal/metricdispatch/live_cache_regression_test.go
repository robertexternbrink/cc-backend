// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricdispatch

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ClusterCockpit/cc-lib/v2/lrucache"
	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

type liveCacheTestRepo struct {
	fakeRepo
	dataCalls int
	data      func(*schema.Job) schema.JobData
}

func (r *liveCacheTestRepo) LoadData(job *schema.Job, _ []string, _ []schema.MetricScope, _ context.Context, _ int, _ string) (schema.JobData, error) {
	r.dataCalls++
	return r.data(job), nil
}

func installLiveCacheTestRepo(t *testing.T) *liveCacheTestRepo {
	t.Helper()
	oldCache := cache
	cache = lrucache.New(128 * 1024 * 1024)
	t.Cleanup(func() { cache = oldCache })
	r := &liveCacheTestRepo{data: func(job *schema.Job) schema.JobData {
		values := make([]schema.Float, job.Duration/10)
		for i := range values {
			values[i] = schema.Float(job.StartTime + int64(i)*10)
		}
		return schema.JobData{Metrics: map[string]schema.ScopedMetrics{"cpu_user": {
			schema.MetricScopeNode: {Timestep: 10, Series: []schema.Series{{Hostname: "node", Data: values}}},
		}}}
	}}
	useRepo(t, "live-cache-regression", r)
	return r
}

func TestLiveCacheTracksTimeRangeAndMonitoringStatus(t *testing.T) {
	repo := installLiveCacheTestRepo(t)
	id := int64(908070)
	job := schema.Job{ID: &id, Cluster: "live-cache-regression", State: schema.JobStateRunning, StartTime: 1000, Duration: 60}
	load := func() schema.Series {
		t.Helper()
		data, err := LoadData(&job, []string{"cpu_user"}, []schema.MetricScope{schema.MetricScopeNode}, context.Background(), 240, "lttb")
		if err != nil {
			t.Fatal(err)
		}
		return data.Metrics["cpu_user"][schema.MetricScopeNode].Series[0]
	}
	if got := load(); len(got.Data) != 6 {
		t.Fatalf("initial result has %d points", len(got.Data))
	}
	load()
	if repo.dataCalls != 1 {
		t.Fatal("unchanged request did not reuse its cached result")
	}
	job.Duration = 180
	if got := load(); len(got.Data) != 18 {
		t.Errorf("refreshed duration returns %d points, want18", len(got.Data))
	}
	job.StartTime = 1500
	if got := load(); got.Data[0] != 1500 {
		t.Errorf("corrected start time returns stale first value %v", got.Data[0])
	}
	before := repo.dataCalls
	job.MonitoringStatus = schema.MonitoringStatusRunningOrArchiving
	load()
	if repo.dataCalls != before+1 {
		t.Error("changed monitoring status reused a result from the previous monitoring state")
	}
}

func TestArchivingJobUsesLiveCacheTTL(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		repo := installLiveCacheTestRepo(t)
		id := int64(908071)
		job := schema.Job{ID: &id, Cluster: "live-cache-regression", State: schema.JobStateCompleted, MonitoringStatus: schema.MonitoringStatusRunningOrArchiving, StartTime: 1000, Duration: 60}
		load := func() {
			t.Helper()
			if _, err := LoadData(&job, []string{"cpu_user"}, []schema.MetricScope{schema.MetricScopeNode}, context.Background(), 240, "lttb"); err != nil {
				t.Fatal(err)
			}
		}
		load()
		load()
		if repo.dataCalls != 1 {
			t.Fatal("initial archiving request was not cached")
		}
		time.Sleep(2*time.Minute + time.Second)
		load()
		if repo.dataCalls != 2 {
			t.Errorf("archiving result remained cached beyond live TTL: %d backend calls, want2", repo.dataCalls)
		}
	})
}
