// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package archive

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ClusterCockpit/cc-lib/v2/lrucache"
	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func isolatedArchiveDataCache(t *testing.T) {
	t.Helper()
	previous := cache
	cache = lrucache.New(1024 * 1024)
	t.Cleanup(func() { cache = previous })
}

func statisticsCacheFixture() schema.JobData {
	id := "0"
	return schema.JobData{Metrics: map[string]schema.ScopedMetrics{
		"cpu_user": {schema.MetricScopeNode: {Timestep: 10, Series: []schema.Series{{Hostname: "node", ID: &id, Data: []schema.Float{1, 2, 3}, Statistics: schema.MetricStatistics{Min: 1, Avg: 2, Max: 3}}}}},
	}}
}

func TestArchiveStatisticsSurviveRawCacheEviction(t *testing.T) {
	isolatedArchiveDataCache(t)
	raw, err := json.Marshal(statisticsCacheFixture())
	if err != nil {
		t.Fatal(err)
	}
	first, err := DecodeJobStats(strings.NewReader(string(raw)), t.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !cache.Del(t.Name()) {
		t.Fatal("expected an existing raw cache entry")
	}
	// The second reader cannot supply an archive: this must use the independent
	// statistics summary after evicting the expensive raw-data cache entry.
	second, err := DecodeJobStats(strings.NewReader("raw archive must not be read again"), t.Name())
	if err != nil {
		t.Fatalf("statistics reread an evicted raw archive: %v", err)
	}
	if first.Metrics["cpu_user"][schema.MetricScopeNode][0].Data.Avg != second.Metrics["cpu_user"][schema.MetricScopeNode][0].Data.Avg {
		t.Fatal("cached summary changed")
	}
}

func TestArchiveStatisticsOwnScalarValues(t *testing.T) {
	isolatedArchiveDataCache(t)
	source := statisticsCacheFixture()
	cache.Get(t.Name(), func() (any, time.Duration, int) { return source, time.Hour, source.Size() })
	got, err := DecodeJobStats(strings.NewReader("must use cached raw data"), t.Name())
	if err != nil {
		t.Fatal(err)
	}
	series := &source.Metrics["cpu_user"][schema.MetricScopeNode].Series[0]
	series.Statistics.Avg = 99
	*series.ID = "changed"
	summary := got.Metrics["cpu_user"][schema.MetricScopeNode][0]
	if summary.Data.Avg != 2 || *summary.ID != "0" {
		t.Fatalf("summary aliases raw series: average=%v id=%q", summary.Data.Avg, *summary.ID)
	}
}
