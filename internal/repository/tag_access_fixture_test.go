// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import "testing"

func tagAccessRepository(t *testing.T) *JobRepository {
	t.Helper()
	r := setup(t)
	for i, user := range []string{"owner", "other"} {
		_, err := r.DB.Exec(`INSERT INTO job(id,job_id,cluster,subcluster,start_time,hpc_user,project,duration,walltime,job_state,meta_data,resources,num_nodes,num_hwthreads,num_acc,shared,monitoring_status,footprint,cluster_partition,array_job_id,smt,energy)
  VALUES(?,?,'tag-access','cpu',1000,?,'project',1000,2000,'running','{}','[]',1,4,0,'none',1,'{}','',0,1,0)`, 9000000+i, 9000000+i, user)
		noErr(t, err)
	}
	return r
}
