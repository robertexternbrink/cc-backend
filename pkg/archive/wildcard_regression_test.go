// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package archive

import (
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func TestWildcardNodeLookupAgreesWithJobAssignment(t *testing.T) {
	previousClusters, previousLists := Clusters, NodeLists
	t.Cleanup(func() { Clusters, NodeLists = previousClusters, previousLists })
	Clusters = []*schema.Cluster{{Name: "dynamic", SubClusters: []*schema.SubCluster{{Name: "all", Nodes: "*"}}}}
	NodeLists = map[string]map[string]NodeList{"dynamic": {}}
	job := &schema.Job{Cluster: "dynamic", Resources: []*schema.Resource{{Hostname: "new-node"}}}
	if err := AssignSubCluster(job); err != nil {
		t.Fatal(err)
	}
	got, err := GetSubClusterByNode("dynamic", "new-node")
	if err != nil || got != job.SubCluster {
		t.Fatalf("node lookup %q/%v disagrees with job assignment %q", got, err, job.SubCluster)
	}
}
