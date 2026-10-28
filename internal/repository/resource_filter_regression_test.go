// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"reflect"
	"testing"

	"github.com/ClusterCockpit/cc-backend/internal/graph/model"
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

func TestResourcePrefixFilterExecutesAndMatchesAnyHost(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	noErr(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE job (id INTEGER, resources TEXT)`)
	noErr(t, err)
	_, err = db.Exec(`INSERT INTO job VALUES
		(1,'[{"hostname":"cpu001"},{"hostname":"gpu001"}]'),
		(2,'[{"hostname":"cpu002"}]'),
		(3,'[{"hostname":"othercpu001"}]'), (4,'[]')`)
	noErr(t, err)
	prefix := "cpu"
	query := buildResourceJSONCondition("hostname", &model.StringInput{StartsWith: &prefix}, sq.Select("id").From("job").OrderBy("id"))
	statement, args, err := query.ToSql()
	noErr(t, err)
	var got []int
	noErr(t, db.Select(&got, statement, args...))
	if !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("prefix filter returned %v", got)
	}
}
