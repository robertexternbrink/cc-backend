// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"testing"

	"github.com/ClusterCockpit/cc-backend/internal/graph/model"
)

func TestStatisticsRoundsHoursAfterConvertingSeconds(t *testing.T) {
	r := statisticsParityRepository(t)
	_, err := r.DB.Exec(`INSERT INTO job(id,hpc_user,job_state,start_time,duration,num_nodes,num_hwthreads,num_acc)
 VALUES(1,'owner','completed',1000,5400,3,7,1)`)
	noErr(t, err)
	fields := map[string]bool{"totalWalltime": true, "totalNodeHours": true, "totalCoreHours": true, "totalAccHours": true}
	check := func(stats []*model.JobsStatistics) {
		t.Helper()
		if len(stats) != 1 {
			t.Fatalf("got %d summaries", len(stats))
		}
		got := stats[0]
		if got.TotalWalltime != 2 || got.TotalNodeHours != 5 || got.TotalCoreHours != 11 || got.TotalAccHours != 2 {
			t.Fatalf("half-hours truncated before rounding: got%d/%d/%d/%d, want2/5/11/2", got.TotalWalltime, got.TotalNodeHours, got.TotalCoreHours, got.TotalAccHours)
		}
	}
	stats, err := r.JobsStats(getContext(t), nil, fields)
	noErr(t, err)
	check(stats)
	group := model.AggregateUser
	grouped, err := r.JobsStatsGrouped(getContext(t), nil, nil, nil, &group, fields)
	noErr(t, err)
	check(grouped)
}
