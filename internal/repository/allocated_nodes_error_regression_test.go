// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var allocatedRowsFailure = errors.New("synthetic allocated-node row failure")

type (
	allocatedRowsDriver     struct{}
	allocatedRowsConnection struct{ mode string }
	allocatedRowsStatement  struct{ mode string }
	allocatedRowsResult     struct {
		mode     string
		returned bool
	}
)

func (allocatedRowsDriver) Open(mode string) (driver.Conn, error) {
	return &allocatedRowsConnection{mode: mode}, nil
}

func (c *allocatedRowsConnection) Prepare(query string) (driver.Stmt, error) {
	if !strings.HasPrefix(query, "SELECT resources, subcluster FROM job WHERE") {
		return nil, errors.New("unexpected synthetic query")
	}
	return &allocatedRowsStatement{mode: c.mode}, nil
}

func (*allocatedRowsConnection) Close() error              { return nil }
func (*allocatedRowsConnection) Begin() (driver.Tx, error) { return nil, driver.ErrSkip }
func (*allocatedRowsStatement) Close() error               { return nil }
func (*allocatedRowsStatement) NumInput() int              { return 1 }
func (*allocatedRowsStatement) Exec([]driver.Value) (driver.Result, error) {
	return nil, driver.ErrSkip
}

func (s *allocatedRowsStatement) Query(arguments []driver.Value) (driver.Rows, error) {
	if len(arguments) != 1 || arguments[0] != "audit-cluster" {
		return nil, errors.New("unexpected synthetic cluster argument")
	}
	return &allocatedRowsResult{mode: s.mode}, nil
}

func (*allocatedRowsResult) Columns() []string { return []string{"resources", "subcluster"} }
func (*allocatedRowsResult) Close() error      { return nil }
func (r *allocatedRowsResult) Next(destination []driver.Value) error {
	if !r.returned && strings.HasPrefix(r.mode, "prefix") {
		r.returned = true
		destination[0] = []byte(`[{"hostname":"node01"},{"hostname":"node02"}]`)
		destination[1] = "compute"
		return nil
	}
	if strings.HasSuffix(r.mode, "fault") {
		return allocatedRowsFailure
	}
	return io.EOF
}

var allocatedRowsRegister sync.Once

func TestAllocatedNodesReportsIterationErrors(t *testing.T) {
	allocatedRowsRegister.Do(func() { sql.Register("audit-allocated-rows", allocatedRowsDriver{}) })
	for _, mode := range []string{"empty-eof", "prefix-eof", "empty-fault", "prefix-fault"} {
		t.Run(mode, func(t *testing.T) {
			db, err := sqlx.Open("audit-allocated-rows", mode)
			noErr(t, err)
			defer db.Close()
			repository := &JobRepository{DB: db, stmtCache: sq.NewStmtCache(db)}
			got, err := repository.AllocatedNodes("audit-cluster")
			if strings.HasSuffix(mode, "fault") {
				if !errors.Is(err, allocatedRowsFailure) || got != nil {
					t.Fatalf("row failure must return error and no partial result; got result=%v error=%v", got, err)
				}
				return
			}
			noErr(t, err)
			want := map[string]map[string]int{}
			if mode == "prefix-eof" {
				want["compute"] = map[string]int{"node01": 1, "node02": 1}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("normal EOF changed result: got %v want %v", got, want)
			}
		})
	}
}
