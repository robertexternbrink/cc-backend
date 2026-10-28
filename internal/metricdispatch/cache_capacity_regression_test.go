// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricdispatch

import (
	"context"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/lrucache"
	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func TestPlotCacheBudgetIncludesBackingCapacityAndPercentiles(t *testing.T) {
	const budget = 64 * 1024
	large := func() []schema.Float { return make([]schema.Float, 1, 128*1024) }
	for _, tt := range []struct {
		name    string
		enlarge func(*schema.JobMetric)
	}{
		{"raw_data_capacity", func(m *schema.JobMetric) { m.Series[0].Data = large() }},
		{"series_capacity", func(m *schema.JobMetric) { m.Series = make([]schema.Series, 1, 4096) }},
		{"statistics_min_capacity", func(m *schema.JobMetric) { m.StatisticsSeries = &schema.StatsSeries{Min: large()} }},
		{"statistics_mean_capacity", func(m *schema.JobMetric) { m.StatisticsSeries = &schema.StatsSeries{Mean: large()} }},
		{"statistics_median_capacity", func(m *schema.JobMetric) { m.StatisticsSeries = &schema.StatsSeries{Median: large()} }},
		{"statistics_max_capacity", func(m *schema.JobMetric) { m.StatisticsSeries = &schema.StatsSeries{Max: large()} }},
		{"statistics_percentile_capacity", func(m *schema.JobMetric) {
			m.StatisticsSeries = &schema.StatsSeries{Percentiles: map[int][]schema.Float{90: large()}}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := installLiveCacheTestRepo(t)
			cache = lrucache.New(budget)
			metric := &schema.JobMetric{Timestep: 10, Series: []schema.Series{{Hostname: "node", Data: []schema.Float{1}}}}
			tt.enlarge(metric)
			data := schema.JobData{Metrics: map[string]schema.ScopedMetrics{"cpu_user": {schema.MetricScopeNode: metric}}}
			repo.data = func(*schema.Job) schema.JobData { return data }
			id := int64(908073)
			job := schema.Job{ID: &id, Cluster: "live-cache-regression", State: schema.JobStateRunning, StartTime: 1000, Duration: 10}
			metrics, scopes := []string{"cpu_user"}, []schema.MetricScope{schema.MetricScopeNode}
			if _, err := LoadData(&job, metrics, scopes, context.Background(), 240, "lttb"); err != nil {
				t.Fatal(err)
			}
			key := cacheKey(&job, metrics, scopes, 240, "lttb")
			if cache.Get(key, nil) != nil {
				t.Errorf("cache admitted %s despite its backing allocation exceeding the 64 KiB budget", tt.name)
			}
			// A one-point control must still be admitted under the same budget.
			compact := schema.JobData{Metrics: map[string]schema.ScopedMetrics{"cpu_user": {schema.MetricScopeNode: {Timestep: 10, Series: []schema.Series{{Hostname: "node", Data: []schema.Float{1}}}}}}}
			repo.data = func(*schema.Job) schema.JobData { return compact }
			id++
			if _, err := LoadData(&job, metrics, scopes, context.Background(), 240, "lttb"); err != nil {
				t.Fatal(err)
			}
			if cache.Get(cacheKey(&job, metrics, scopes, 240, "lttb"), nil) == nil {
				t.Fatal("one-point compact control was incorrectly rejected")
			}
		})
	}
}
