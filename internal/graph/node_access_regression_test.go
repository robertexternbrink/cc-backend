// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package graph

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ClusterCockpit/cc-backend/internal/repository"
	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	"github.com/ClusterCockpit/cc-lib/v2/schema"
	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
	"github.com/qustavo/sqlhooks/v2"
)

type nodeAccessReadCounter struct{ reads atomic.Int64 }

func (r *nodeAccessReadCounter) Before(ctx context.Context, query string, _ ...any) (context.Context, error) {
	if strings.HasPrefix(query, "SELECT ") {
		r.reads.Add(1)
	}
	return ctx, nil
}
func (*nodeAccessReadCounter) After(ctx context.Context, _ string, _ ...any) (context.Context, error) {
	return ctx, nil
}

var nodeAccessDriverID atomic.Uint64

func TestNodeAccessCheckedBeforeLookup(t *testing.T) {
	cclog.Init("warn", true)
	if err := repository.ResetConnection(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repository.ResetConnection() })
	path := filepath.Join(t.TempDir(), "connection.db")
	if err := repository.MigrateDB(path); err != nil {
		t.Fatal(err)
	}
	repository.Connect(path)
	repo := repository.GetNodeRepository()
	counter := &nodeAccessReadCounter{}
	driver := fmt.Sprintf("node-access-regression-%d", nodeAccessDriverID.Add(1))
	sql.Register(driver, sqlhooks.Wrap(&sqlite3.SQLiteDriver{}, counter))
	db, err := sqlx.Open(driver, filepath.Join(t.TempDir(), "nodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	original := repo.DB
	repo.DB = db
	t.Cleanup(func() { repo.DB = original; db.Close() })
	_, err = db.Exec(`CREATE TABLE node(id INTEGER PRIMARY KEY,hostname TEXT,cluster TEXT,subcluster TEXT);
 CREATE TABLE node_state(id INTEGER PRIMARY KEY,node_id INTEGER,time_stamp INTEGER,node_state TEXT,health_state TEXT);
 INSERT INTO node VALUES(1,'n1','cluster','subcluster');
 INSERT INTO node_state VALUES(1,1,100,'idle','full');`)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &queryResolver{Resolver: &Resolver{}}
	for _, tc := range []struct {
		name  string
		user  *schema.User
		allow bool
	}{
		{"missing user", nil, false},
		{"OIDC user", &schema.User{Username: "caller", Roles: []string{"user"}, AuthSource: schema.AuthViaOIDC}, false},
		{"LDAP manager", &schema.User{Username: "caller", Roles: []string{"manager"}, AuthSource: schema.AuthViaLDAP}, false},
		{"API only", &schema.User{Username: "service", Roles: []string{"api"}, AuthSource: schema.AuthViaToken}, false},
		{"mixed API", &schema.User{Username: "caller", Roles: []string{"api", "user"}}, false},
		{"no roles", &schema.User{Username: "caller"}, false},
		{"OIDC admin", &schema.User{Username: "admin", Roles: []string{"admin"}, AuthSource: schema.AuthViaOIDC}, true},
		{"LDAP support", &schema.User{Username: "support", Roles: []string{"support"}, AuthSource: schema.AuthViaLDAP}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.user != nil {
				ctx = context.WithValue(ctx, repository.ContextUserKey, tc.user)
			}
			counter.reads.Store(0)
			node, err := resolver.Node(ctx, "1")
			if tc.allow {
				if err != nil || node == nil || node.Hostname != "n1" || counter.reads.Load() != 1 {
					t.Fatalf("allowed lookup changed: node=%+v reads=%d err=%v", node, counter.reads.Load(), err)
				}
			} else if err == nil || node != nil || counter.reads.Load() != 0 {
				t.Fatalf("unauthorized lookup reached database: node=%+v reads=%d err=%v", node, counter.reads.Load(), err)
			}
		})
	}
}
