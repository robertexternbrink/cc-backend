// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricstore

import (
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func TestScaleByPreservesStatisticalExtrema(t *testing.T) {
	for _, tc := range []struct {
		factor, min, max, avg schema.Float
	}{
		{-2, -6, -2, -4}, {2, 2, 6, 4}, {1, 1, 3, 2}, {0, 0, 0, 0},
	} {
		data := APIMetricData{Data: []schema.Float{1, 3}, Min: 1, Max: 3, Avg: 2}
		data.ScaleBy(tc.factor)
		if data.Min != tc.min || data.Max != tc.max || data.Avg != tc.avg || data.Data[0] != tc.factor || data.Data[1] != 3*tc.factor {
			t.Errorf("factor %v: got %+v, want min/max/avg %v/%v/%v", tc.factor, data, tc.min, tc.max, tc.avg)
		}
	}
}
