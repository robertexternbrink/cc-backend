// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"
	"testing"

	"github.com/ClusterCockpit/cc-backend/internal/graph/model"
	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func TestStatisticsTagFilterCountsEachJobOnce(t *testing.T) {
	r := statisticsParityRepository(t)
	_, err := r.DB.Exec(`INSERT INTO job(id,cluster,hpc_user,project,job_state,start_time,duration,num_nodes,num_hwthreads,num_acc) VALUES
 (1,'cluster','owner','project','completed',1000,3600,2,96,4),
 (2,'cluster','other','project','completed',1000,7200,6,384,8);
 INSERT INTO jobtag VALUES(1,1),(1,2),(2,1);`)
	noErr(t, err)
	ctx := context.WithValue(getContext(t), ContextUserKey, &schema.User{Username: "owner", Roles: []string{"user"}})
	filters := []*model.JobFilter{{Tags: []string{"1", "2"}}}
	stats, err := r.JobsStats(ctx, filters, nil)
	noErr(t, err)
	if len(stats) != 1 || stats[0].TotalJobs != 1 || stats[0].TotalNodeHours != 2 || stats[0].TotalCoreHours != 96 || stats[0].TotalAccHours != 4 {
		t.Fatalf("tag matches multiplied aggregate or escaped user restriction: %+v", stats[0])
	}
	group := model.AggregateUser
	grouped, err := r.JobsStatsGrouped(ctx, filters, nil, nil, &group, nil)
	noErr(t, err)
	if len(grouped) != 1 || grouped[0].ID != "owner" || grouped[0].TotalJobs != 1 || grouped[0].TotalNodes != 2 {
		t.Fatalf("tag matches multiplied grouped statistics: %+v", grouped[0])
	}
	bins := "1h"
	histogram, err := r.AddHistograms(ctx, filters, &model.JobsStatistics{}, &bins)
	noErr(t, err)
	if histogram.HistDuration[1].Count != 1 || len(histogram.HistNumNodes) != 1 || histogram.HistNumNodes[0].Count != 1 {
		t.Fatalf("tag matches multiplied histogram: duration=%+v nodes=%+v", histogram.HistDuration[1], histogram.HistNumNodes)
	}
	count, err := r.CountJobs(ctx, filters)
	noErr(t, err)
	if count != 1 {
		t.Fatalf("list count changed: %d", count)
	}
	empty, err := r.CountJobs(ctx, []*model.JobFilter{{Tags: []string{}}})
	noErr(t, err)
	if empty != 0 {
		t.Fatalf("empty tag selection matched %d jobs", empty)
	}
}
