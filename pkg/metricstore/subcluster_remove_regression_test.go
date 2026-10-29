// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricstore

import (
	"testing"

	"github.com/ClusterCockpit/cc-backend/pkg/archive"
	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func TestInferredSubClusterHonorsRemovedMetric(t *testing.T) {
	initDeviceArchive(t)
	mc := archive.GetMetricConfig(deviceCluster, "fs_read_bw")
	previous := mc.SubClusters
	mc.SubClusters = []*schema.SubClusterConfig{{Metric: schema.Metric{Name: "a"}, Remove: true}}
	t.Cleanup(func() { mc.SubClusters = previous })
	for _, subCluster := range []string{"a", ""} {
		queries, _, err := buildNodeQueries(deviceCluster, subCluster, []string{"a01"},
			[]string{"fs_read_bw"}, []schema.MetricScope{schema.MetricScopeFilesystem}, 60)
		if err != nil || len(queries) != 0 {
			t.Errorf("subcluster %q: queries=%v err=%v; want no removed metric", subCluster, queries, err)
		}
	}
	queries, _, err := buildNodeQueries(deviceCluster, "", []string{"b01"},
		[]string{"fs_read_bw"}, []schema.MetricScope{schema.MetricScopeFilesystem}, 60)
	if err != nil || len(queries) != 1 {
		t.Fatalf("unremoved subcluster b: queries=%v err=%v", queries, err)
	}
}
