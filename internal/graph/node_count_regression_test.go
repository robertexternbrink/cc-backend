// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package graph

import (
	"context"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func TestNodeCountResolverMatchesExpandedDimensions(t *testing.T) {
	resolver := &subClusterResolver{}
	for _, test := range []struct {
		nodes string
		count int
	}{
		{"rack[01-02]node[01-02]gpu,login01", 5},
		{"cn-00[13-18,22-24],login01", 10},
		{"node01", 1},
	} {
		got, err := resolver.NumberOfNodes(context.Background(), &schema.SubCluster{Nodes: test.nodes})
		if err != nil || got != test.count {
			t.Fatalf("node expression %s: got %d, error %v; want %d", test.nodes, got, err, test.count)
		}
	}
}
