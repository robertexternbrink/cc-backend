// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"testing"

	"github.com/ClusterCockpit/cc-backend/internal/graph/model"
)

func TestStatisticsDurationHistogramUsesWholeDayBins(t *testing.T) {
	r := statisticsParityRepository(t)
	_, err := r.DB.Exec(`INSERT INTO job(id,hpc_user,job_state,start_time,duration,num_nodes,num_hwthreads,num_acc) VALUES
 (1,'owner','completed',1000,3600,1,4,0),
 (2,'owner','completed',1000,86400,1,4,0),
 (3,'owner','completed',1000,129600,1,4,0)`)
	noErr(t, err)
	bins := "24h"
	stats, err := r.AddHistograms(getContext(t), nil, &model.JobsStatistics{}, &bins)
	noErr(t, err)
	if len(stats.HistDuration) != 24 {
		t.Fatalf("got %d daily bins", len(stats.HistDuration))
	}
	if first, second := stats.HistDuration[0], stats.HistDuration[1]; first.Value != 86400 || first.Count != 1 || second.Value != 172800 || second.Count != 2 {
		t.Fatalf("daily bins are not whole days: first=%+v second=%+v", first, second)
	}
	for _, unit := range []struct {
		name         string
		width, count int
	}{{"1m", 60, 60}, {"10m", 600, 72}, {"1h", 3600, 48}, {"6h", 21600, 12}, {"12h", 43200, 14}} {
		stats, err := r.AddHistograms(getContext(t), nil, &model.JobsStatistics{}, &unit.name)
		noErr(t, err)
		if len(stats.HistDuration) != unit.count || stats.HistDuration[0].Value != unit.width {
			t.Fatalf("existing %s bins changed", unit.name)
		}
	}
}
