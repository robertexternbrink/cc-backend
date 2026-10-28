// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"path/filepath"
	"testing"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

// statisticsParityRepository uses a new SQLite file and only the existing
// statistics schema, without introducing search tables, indices or projections.
func statisticsParityRepository(t *testing.T) *JobRepository {
	t.Helper()
	cclog.Init("warn", true)
	db, err := sqlx.Open("sqlite3", filepath.Join(t.TempDir(), "statistics.db"))
	noErr(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE job (
  id INTEGER PRIMARY KEY, cluster TEXT, subcluster TEXT, hpc_user TEXT,
  project TEXT, job_state TEXT, start_time INTEGER, duration INTEGER,
  num_nodes INTEGER, num_hwthreads INTEGER, num_acc INTEGER, footprint TEXT);
  CREATE TABLE hpc_user(username TEXT PRIMARY KEY,name TEXT);
  CREATE TABLE jobtag(job_id INTEGER,tag_id INTEGER,PRIMARY KEY(job_id,tag_id));`)
	noErr(t, err)
	return &JobRepository{DB: db, stmtCache: sq.NewStmtCache(db.DB)}
}
