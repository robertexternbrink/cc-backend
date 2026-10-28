// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/ClusterCockpit/cc-lib/v2/lrucache"
	"github.com/ClusterCockpit/cc-lib/v2/schema"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

func TestEnergyFootprintCacheUsesSerializedSize(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	noErr(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE job (id INTEGER, energy_footprint TEXT)`)
	noErr(t, err)
	_, err = db.Exec(`INSERT INTO job VALUES (1,?)`, `{"`+strings.Repeat("large_metric_name", 8)+`":1}`)
	noErr(t, err)
	cache := lrucache.New(48)
	repo := &JobRepository{DB: db, stmtCache: sq.NewStmtCache(db), cache: cache}
	id := int64(1)
	data, err := repo.FetchEnergyFootprint(&schema.Job{ID: &id})
	noErr(t, err)
	if len(data) != 1 {
		t.Fatal("energy footprint was not returned")
	}
	cache.Put("competing-entry", struct{}{}, 32, time.Minute)
	if cache.Get("energyFootprint:1", nil) != nil {
		t.Fatal("oversized footprint was charged as one map entry instead of serialized bytes")
	}
}
