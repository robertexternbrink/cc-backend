// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"testing"
)

func TestStatisticsIncludesRequestedResourceTotals(t *testing.T) {
	r := statisticsParityRepository(t)
	_, err := r.DB.Exec(`INSERT INTO job(id,cluster,hpc_user,job_state,start_time,duration,num_nodes,num_hwthreads,num_acc)
 VALUES(1,'cluster','owner','completed',1000,3600,2,96,4)`)
	noErr(t, err)
	for _, fields := range []map[string]bool{nil, {"totalNodes": true, "totalCores": true, "totalAccs": true}, {"totalNodes": true}} {
		stats, err := r.JobsStats(getContext(t), nil, fields)
		noErr(t, err)
		if len(stats) != 1 {
			t.Fatalf("got %d summaries", len(stats))
		}
		wantCores, wantAccs := 96, 4
		if fields != nil && !fields["totalCores"] {
			wantCores = 0
		}
		if fields != nil && !fields["totalAccs"] {
			wantAccs = 0
		}
		got := stats[0]
		if got.TotalNodes != 2 || got.TotalCores != wantCores || got.TotalAccs != wantAccs {
			t.Fatalf("fields %v: resource totals=%d/%d/%d, want2/%d/%d", fields, got.TotalNodes, got.TotalCores, got.TotalAccs, wantCores, wantAccs)
		}
		if fields != nil && (got.TotalNodeHours != 0 || got.TotalCoreHours != 0 || got.TotalAccHours != 0) {
			t.Fatalf("unrequested hours were populated: %+v", got)
		}
	}
}
