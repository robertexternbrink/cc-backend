// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricdispatch

import (
	"context"
	"testing"
	"time"
)

func TestNodeQueriesRejectUnknownClusterWithoutPanicking(t *testing.T) {
	from, to := time.Unix(1700000000, 0), time.Unix(1700000100, 0)
	t.Run("node", func(t *testing.T) {
		if _, err := LoadNodeData("does-not-exist", nil, []string{"host"}, nil, from, to, context.Background()); err == nil {
			t.Fatal("unknown cluster was accepted")
		}
	})
	t.Run("node-list", func(t *testing.T) {
		if _, err := LoadNodeListData("does-not-exist", "", []string{"host"}, nil, nil, 60, from, to, context.Background(), ""); err == nil {
			t.Fatal("unknown cluster was accepted")
		}
	})
}
