// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricstore

import (
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func TestPaddingUsesReturnedResolution(t *testing.T) {
	ms := &MemoryStore{Metrics: map[string]MetricConfig{"metric": {Frequency: 10}}}
	for _, tc := range []struct {
		resolution int64
		padding    int
	}{{20, 2}, {10, 4}, {0, 4}} {
		data := APIMetricData{Data: []schema.Float{7, 9}, From: 140, To: 180, Resolution: tc.resolution}
		data.PadDataWithNull(ms, 100, 180, "metric")
		if len(data.Data) != tc.padding+2 {
			t.Errorf("resolution %d: got %d points, want %d", tc.resolution, len(data.Data), tc.padding+2)
			continue
		}
		for _, x := range data.Data[:tc.padding] {
			if !x.IsNaN() {
				t.Errorf("padding contains nonmissing value %v", x)
			}
		}
		if data.Data[tc.padding] != 7 || data.Data[tc.padding+1] != 9 || data.From != 140 || data.To != 180 {
			t.Fatalf("padding changed original values or time metadata: %+v", data)
		}
	}
}
