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

func TestPlotCacheChargesTransformedResult(t *testing.T) {
	for _, tt := range []struct {
		name, metric string
		series       int
		scopes       []schema.MetricScope
	}{
		{"statistics", "cpu_user", 8, []schema.MetricScope{schema.MetricScopeCore}},
		{"added_node_scope", "flops_any", 2, []schema.MetricScope{schema.MetricScopeCore, schema.MetricScopeNode}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := installLiveCacheTestRepo(t)
			source := schema.JobData{Metrics: map[string]schema.ScopedMetrics{tt.metric: {schema.MetricScopeCore: {Timestep: 10}}}}
			metric := source.Metrics[tt.metric][schema.MetricScopeCore]
			for i := range tt.series {
				values := make([]schema.Float, 64)
				for j := range values {
					values[j] = schema.Float(i + 1)
				}
				metric.Series = append(metric.Series, schema.Series{Hostname: "node", Data: values})
			}
			// Raw data fits, but either generated statistics or the added node series
			// exceeds this budget. Verify admission through the real LoadData path.
			budget := plotCacheSize(source) + 256
			cache = lrucache.New(budget)
			repo.data = func(*schema.Job) schema.JobData { return deepCopy(source) }
			id := int64(908072)
			job := schema.Job{ID: &id, Cluster: "live-cache-regression", State: schema.JobStateRunning, StartTime: 1000, Duration: 640}
			result, err := LoadData(&job, []string{tt.metric}, tt.scopes, context.Background(), 240, "lttb")
			if err != nil {
				t.Fatal(err)
			}
			if tt.name == "statistics" && result.Metrics[tt.metric][schema.MetricScopeCore].StatisticsSeries == nil {
				t.Fatal("fixture did not generate statistics")
			}
			if tt.name == "added_node_scope" && result.Metrics[tt.metric][schema.MetricScopeNode] == nil {
				t.Fatal("fixture did not generate node scope")
			}
			if size := plotCacheSize(result); size <= budget {
				t.Fatalf("fixture transformed size %d does not exceed cache budget %d", size, budget)
			}
			key := cacheKey(&job, []string{tt.metric}, tt.scopes, 240, "lttb")
			if cache.Get(key, nil) != nil {
				t.Error("cache admitted the transformed result using its smaller pre-transformation size")
			}
		})
	}
}
