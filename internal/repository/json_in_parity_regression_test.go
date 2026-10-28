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

func TestExistingJSONNameAndNodeInFilters(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	noErr(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE job(id INTEGER,meta_data TEXT,resources TEXT);
        INSERT INTO job VALUES
        (1,'{"jobName":"first"}','[{"hostname":"cpu001"},{"hostname":"cpu002"}]'),
        (2,'{"jobName":"second"}','[{"hostname":"cpu003"}]'),
        (3,'{"jobName":""}','[]'),
        (4,'{"jobName":"quote''; DROP TABLE job; --"}','[{"hostname":"quote''; DROP TABLE job; --"}]');`)
	noErr(t, err)
	first := "first"
	cpu001 := "cpu001"
	for _, tc := range []struct {
		name, kind string
		input      model.StringInput
		want       []int
	}{
		{"name two", "name", model.StringInput{In: []string{"first", "second", "first"}}, []int{1, 2}},
		{"name empty string", "name", model.StringInput{In: []string{""}}, []int{3}},
		{"name none", "name", model.StringInput{In: []string{}}, []int{}},
		{"name missing", "name", model.StringInput{In: []string{"missing"}}, []int{}},
		{"name precedence", "name", model.StringInput{Eq: &first, In: []string{"second"}}, []int{1}},
		{"name quotes", "name", model.StringInput{In: []string{"quote'; DROP TABLE job; --"}}, []int{4}},
		{"node two", "node", model.StringInput{In: []string{"cpu001", "cpu002", "cpu003"}}, []int{1, 2}},
		{"node duplicates", "node", model.StringInput{In: []string{"cpu001", "cpu001"}}, []int{1}},
		{"node none", "node", model.StringInput{In: []string{}}, []int{}},
		{"node missing", "node", model.StringInput{In: []string{"missing"}}, []int{}},
		{"node precedence", "node", model.StringInput{Eq: &cpu001, In: []string{"cpu003"}}, []int{1}},
		{"node quotes", "node", model.StringInput{In: []string{"quote'; DROP TABLE job; --"}}, []int{4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := sq.Select("id").From("job").OrderBy("id")
			if tc.kind == "name" {
				q = buildMetaJSONCondition("jobName", &tc.input, q)
			} else {
				q = buildResourceJSONCondition("hostname", &tc.input, q)
			}
			statement, args, err := q.ToSql()
			noErr(t, err)
			got := []int{}
			noErr(t, db.Select(&got, statement, args...))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("%s IN selection returned %v, want %v", tc.kind, got, tc.want)
			}
		})
	}
}
