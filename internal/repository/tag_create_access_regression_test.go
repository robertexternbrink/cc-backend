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

func TestTagCreationRequiresJobAccessBeforeMutation(t *testing.T) {
	r := tagAccessRepository(t)
	for i, tc := range []struct {
		name  string
		user  *schema.User
		job   int64
		scope string
		allow bool
	}{
		{"OIDC owner private", &schema.User{Username: "owner", Roles: []string{"user"}, AuthSource: schema.AuthViaOIDC}, 9000000, "owner", true},
		{"LDAP owner foreign job", &schema.User{Username: "owner", Roles: []string{"user"}, AuthSource: schema.AuthViaLDAP}, 9000001, "owner", false},
		{"manager foreign project", &schema.User{Username: "manager", Roles: []string{"manager"}, Projects: []string{"unassigned"}}, 9000001, "manager", false},
		{"manager assigned project", &schema.User{Username: "manager", Roles: []string{"manager"}, Projects: []string{"project"}}, 9000001, "manager", true},
		{"API only", &schema.User{Username: "service", Roles: []string{"api"}, AuthSource: schema.AuthViaToken}, 9000001, "global", true},
		{"mixed API", &schema.User{Username: "service", Roles: []string{"api", "user"}}, 9000001, "service", false},
		{"support global", &schema.User{Username: "support", Roles: []string{"support"}}, 9000001, "global", true},
		{"support admin denied", &schema.User{Username: "support", Roles: []string{"support"}}, 9000001, "admin", false},
		{"OIDC admin", &schema.User{Username: "admin", Roles: []string{"admin"}, AuthSource: schema.AuthViaOIDC}, 9000001, "admin", true},
		{"missing job", &schema.User{Username: "owner", Roles: []string{"user"}}, 9000099, "owner", false},
		{"missing user", nil, 9000000, "global", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := fmt.Sprintf("creation-%d", i)
			_, err := r.AddTagOrCreate(tc.user, tc.job, "access-guard", name, tc.scope)
			_, exists := r.TagID("access-guard", name, tc.scope)
			associated := r.HasTag(tc.job, "access-guard", name)
			if tc.allow {
				if err != nil || !exists || !associated {
					t.Fatalf("allowed tag mutation changed: exists=%v linked=%v err=%v", exists, associated, err)
				}
			} else if err == nil || exists || associated {
				t.Fatalf("denied job access still changed tag definitions: exists=%v linked=%v err=%v", exists, associated, err)
			}
		})
	}
}
