// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package archive

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

type scopedSelectionBackend struct {
	ArchiveBackend
	stats   schema.ScopedJobStats
	failure error
}

func (b scopedSelectionBackend) LoadJobStats(*schema.Job) (schema.ScopedJobStats, error) {
	return b.stats, b.failure
}

func TestArchivedStatisticsHonorRequestedMetricsAndScopes(t *testing.T) {
	previous := ar
	t.Cleanup(func() { ar = previous })
	node := []*schema.ScopedStats{{Hostname: "node", Data: &schema.MetricStatistics{Avg: 2}}}
	core := []*schema.ScopedStats{{Hostname: "node", Data: &schema.MetricStatistics{Avg: 3}}}
	source := schema.ScopedJobStats{Metrics: map[string]schema.ScopedMetricStats{
		"cpu_user": {schema.MetricScopeNode: node, schema.MetricScopeCore: core},
		"mem_used": {schema.MetricScopeNode: node},
	}}
	ar = scopedSelectionBackend{stats: source}
	for _, tc := range []struct {
		name    string
		metrics []string
		scopes  []schema.MetricScope
		want    map[string]schema.ScopedMetricStats
	}{
		{"node only", []string{"cpu_user"}, []schema.MetricScope{schema.MetricScopeNode}, map[string]schema.ScopedMetricStats{"cpu_user": {schema.MetricScopeNode: node}}},
		{"core only", []string{"cpu_user"}, []schema.MetricScope{schema.MetricScopeCore}, map[string]schema.ScopedMetricStats{"cpu_user": {schema.MetricScopeCore: core}}},
		{"all", nil, nil, source.Metrics},
		{"empty selections", []string{}, []schema.MetricScope{}, source.Metrics},
		{"missing scope", nil, []schema.MetricScope{schema.MetricScopeAccelerator}, map[string]schema.ScopedMetricStats{}},
		{"missing metric", []string{"absent"}, nil, map[string]schema.ScopedMetricStats{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LoadScopedStatsFromArchive(&schema.Job{}, tc.metrics, tc.scopes)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Metrics, tc.want) {
				t.Fatalf("archive selection returned %v, want %v", got.Metrics, tc.want)
			}
		})
	}
	if len(source.Metrics) != 2 || len(source.Metrics["cpu_user"]) != 2 {
		t.Fatal("selection mutated the shared archive cache")
	}
	failure := errors.New("synthetic archive failure")
	ar = scopedSelectionBackend{failure: failure}
	if _, err := LoadScopedStatsFromArchive(&schema.Job{}, []string{"cpu_user"}, nil); !errors.Is(err, failure) {
		t.Fatalf("archive error lost: %v", err)
	}
}
