// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package repository

import (
	"fmt"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func TestTagRemovalRequiresExistingWriteScope(t *testing.T) {
	r := tagAccessRepository(t)
	for i, tc := range []struct {
		name  string
		user  *schema.User
		job   int64
		scope string
		allow bool
	}{
		{"OIDC owner global denied", &schema.User{Username: "owner", Roles: []string{"user"}, AuthSource: schema.AuthViaOIDC}, 9000000, "global", false},
		{"LDAP owner private", &schema.User{Username: "owner", Roles: []string{"user"}, AuthSource: schema.AuthViaLDAP}, 9000000, "owner", true},
		{"owner foreign private denied", &schema.User{Username: "owner", Roles: []string{"user"}}, 9000000, "other", false},
		{"owner foreign job denied", &schema.User{Username: "owner", Roles: []string{"user"}}, 9000001, "owner", false},
		{"support global", &schema.User{Username: "support", Roles: []string{"support"}}, 9000000, "global", true},
		{"support admin denied", &schema.User{Username: "support", Roles: []string{"support"}}, 9000000, "admin", false},
		{"API only admin", &schema.User{Username: "service", Roles: []string{"api"}, AuthSource: schema.AuthViaToken}, 9000000, "admin", true},
		{"API only global", &schema.User{Username: "service", Roles: []string{"api"}}, 9000000, "global", true},
		{"mixed API admin denied", &schema.User{Username: "owner", Roles: []string{"api", "user"}}, 9000000, "admin", false},
		{"admin foreign private denied", &schema.User{Username: "admin", Roles: []string{"admin"}}, 9000000, "owner", false},
		{"admin admin", &schema.User{Username: "admin", Roles: []string{"admin"}}, 9000000, "admin", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := fmt.Sprintf("removal-%d", i)
			id, err := r.CreateTag("access-guard", name, tc.scope)
			noErr(t, err)
			_, err = r.DB.Exec("INSERT INTO jobtag(job_id,tag_id) VALUES(?,?)", tc.job, id)
			noErr(t, err)
			_, err = r.RemoveJobTagByRequest(tc.user, tc.job, "access-guard", name, tc.scope)
			remains := r.HasTag(tc.job, "access-guard", name)
			if tc.allow {
				if err != nil || remains {
					t.Fatalf("allowed scope removal changed: linked=%v err=%v", remains, err)
				}
			} else if err == nil || !remains {
				t.Fatalf("forbidden tag scope removed: linked=%v err=%v", remains, err)
			}
		})
	}
}
