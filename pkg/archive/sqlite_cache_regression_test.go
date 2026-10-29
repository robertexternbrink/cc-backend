// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package archive

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ClusterCockpit/cc-backend/internal/config"
	"github.com/ClusterCockpit/cc-lib/v2/lrucache"
	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func isolateSqlitePayloadCaches(t *testing.T) {
	t.Helper()
	oldRaw, oldStats, oldValidate := cache, statsCache, config.Keys.Validate
	cache, statsCache = lrucache.New(1024*1024), lrucache.New(1024*1024)
	config.Keys.Validate = false
	t.Cleanup(func() { cache, statsCache, config.Keys.Validate = oldRaw, oldStats, oldValidate })
}

func newSqlitePayloadArchive(t *testing.T) *SqliteArchive {
	t.Helper()
	sa := &SqliteArchive{}
	raw, err := json.Marshal(SqliteArchiveConfig{DBPath: filepath.Join(t.TempDir(), "archive.sqlite")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sa.Init(raw); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sa.db.Close() })
	return sa
}

func sqlitePayloadData(value float64, n int) schema.JobData {
	values := make([]schema.Float, n)
	for i := range values {
		values[i] = schema.Float(value)
	}
	return schema.JobData{Metrics: map[string]schema.ScopedMetrics{"cpu_user": {schema.MetricScopeNode: {Timestep: 10, Series: []schema.Series{{Hostname: "node", Data: values, Statistics: schema.MetricStatistics{Min: value, Avg: value, Max: value}}}}}}}
}

func assertSqlitePayload(t *testing.T, sa *SqliteArchive, job *schema.Job, want float64) {
	t.Helper()
	data, err := sa.LoadJobData(job)
	if err != nil {
		t.Fatal(err)
	}
	series := data.Metrics["cpu_user"][schema.MetricScopeNode].Series[0]
	if series.Data[0] != schema.Float(want) || series.Statistics.Avg != want {
		t.Errorf("raw payload still has value/statistic %v/%v, want %v", series.Data[0], series.Statistics.Avg, want)
	}
	stats, err := sa.LoadJobStats(job)
	if err != nil {
		t.Fatal(err)
	}
	if got := stats.Metrics["cpu_user"][schema.MetricScopeNode][0].Data.Avg; got != want {
		t.Errorf("statistics cache returned %v, want %v", got, want)
	}
}

func assertSqlitePayloadMissing(t *testing.T, sa *SqliteArchive, job *schema.Job) {
	t.Helper()
	if _, err := sa.LoadJobData(job); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("deleted job raw data returned error %v, want sql.ErrNoRows", err)
	}
	if _, err := sa.LoadJobStats(job); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("deleted job statistics returned error %v, want sql.ErrNoRows", err)
	}
}

func TestSqlitePayloadCacheTracksImportDeleteAndReinsert(t *testing.T) {
	for _, n := range []int{3, 1024} {
		t.Run(map[bool]string{false: "plain", true: "compressed"}[n > 3], func(t *testing.T) {
			isolateSqlitePayloadCaches(t)
			sa := newSqlitePayloadArchive(t)
			job := &schema.Job{JobID: 12345, Cluster: t.Name(), StartTime: 100, User: "original"}
			put := func(value float64) {
				t.Helper()
				data := sqlitePayloadData(value, n)
				if err := sa.ImportJob(job, &data); err != nil {
					t.Fatal(err)
				}
			}
			put(2)
			assertSqlitePayload(t, sa, job, 2)
			put(7)
			assertSqlitePayload(t, sa, job, 7)
			job.User = "updated metadata"
			if err := sa.StoreJobMeta(job); err != nil {
				t.Fatal(err)
			}
			meta, err := sa.LoadJobMeta(job)
			if err != nil || meta.User != job.User {
				t.Fatalf("metadata update failed: %v", err)
			}
			assertSqlitePayload(t, sa, job, 7)
			sa.CleanUp([]*schema.Job{nil, job})
			assertSqlitePayloadMissing(t, sa, job)
			put(9)
			assertSqlitePayload(t, sa, job, 9)
			sa.Clean(101, 0)
			assertSqlitePayloadMissing(t, sa, job)
			put(11)
			assertSqlitePayload(t, sa, job, 11)
		})
	}
}

func TestSqlitePayloadCacheIsolatesArchiveInstances(t *testing.T) {
	isolateSqlitePayloadCaches(t)
	first, second := newSqlitePayloadArchive(t), newSqlitePayloadArchive(t)
	job := &schema.Job{JobID: 12345, Cluster: t.Name(), StartTime: 100}
	for i, sa := range []*SqliteArchive{first, second} {
		data := sqlitePayloadData(float64(i+1), 3)
		if err := sa.ImportJob(job, &data); err != nil {
			t.Fatal(err)
		}
	}
	assertSqlitePayload(t, first, job, 1)
	assertSqlitePayload(t, second, job, 2)
	for item := range second.Iter(true) {
		if item.Data == nil || item.Data.Metrics["cpu_user"][schema.MetricScopeNode].Series[0].Data[0] != 2 {
			t.Fatal("iterator substituted a cached payload for its own database row")
		}
	}
	first.CleanUp([]*schema.Job{job})
	assertSqlitePayload(t, second, job, 2)
}

func TestSqliteFailedPayloadMutationsPreserveDataAndMetadata(t *testing.T) {
	isolateSqlitePayloadCaches(t)
	sa := newSqlitePayloadArchive(t)
	job := &schema.Job{JobID: 12345, Cluster: t.Name(), StartTime: 100, User: "original"}
	data := sqlitePayloadData(2, 3)
	if err := sa.ImportJob(job, &data); err != nil {
		t.Fatal(err)
	}
	assertSqlitePayload(t, sa, job, 2)
	bad := sqlitePayloadData(math.Inf(1), 3)
	if err := sa.ImportJob(job, &bad); err == nil {
		t.Fatal("invalid JSON payload was accepted")
	}
	if _, err := sa.db.Exec("CREATE TRIGGER reject_payload_update BEFORE UPDATE ON jobs BEGIN SELECT RAISE(ABORT,'reject update'); END"); err != nil {
		t.Fatal(err)
	}
	newer := sqlitePayloadData(7, 3)
	if err := sa.ImportJob(job, &newer); err == nil {
		t.Fatal("trigger-rejected import unexpectedly succeeded")
	}
	if _, err := sa.db.Exec("CREATE TRIGGER reject_payload_delete BEFORE DELETE ON jobs BEGIN SELECT RAISE(ABORT,'reject delete'); END"); err != nil {
		t.Fatal(err)
	}
	sa.CleanUp([]*schema.Job{job})
	assertSqlitePayload(t, sa, job, 2)
	meta, err := sa.LoadJobMeta(job)
	if err != nil || meta.User != "original" {
		t.Fatalf("failed mutation changed metadata: %v", err)
	}
}

func TestSqlitePayloadConcurrentReadsAndImports(t *testing.T) {
	isolateSqlitePayloadCaches(t)
	sa := newSqlitePayloadArchive(t)
	job := &schema.Job{JobID: 12345, Cluster: t.Name(), StartTime: 100}
	data := sqlitePayloadData(0, 3)
	if err := sa.ImportJob(job, &data); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	defer func() { cancel(); wg.Wait() }()
	for range 4 {
		wg.Go(func() {
			for ctx.Err() == nil {
				if _, err := sa.LoadJobData(job); err != nil {
					t.Error(err)
					return
				}
				if _, err := sa.LoadJobStats(job); err != nil {
					t.Error(err)
					return
				}
			}
		})
	}
	for i := 1; i <= 20; i++ {
		data := sqlitePayloadData(float64(i), 3)
		if err := sa.ImportJob(job, &data); err != nil {
			t.Error(err)
			break
		}
		assertSqlitePayload(t, sa, job, float64(i))
	}
	cancel()
	wg.Wait()
	assertSqlitePayload(t, sa, job, 20)
}
