// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricstore

import (
	"errors"
	"math"

	"github.com/ClusterCockpit/cc-lib/v2/resampler"
	"github.com/ClusterCockpit/cc-lib/v2/schema"
	"github.com/ClusterCockpit/cc-lib/v2/util"
)

var errMultipleStatisticsBuffers = errors.New("multiple buffers require temporal aggregation")

// queryStatistics avoids allocating the requested sample array for a native
// series. Spatial aggregation retains Read/AddStats: combining scalar extrema
// would change per-timestep aggregation and propagation of missing samples.
func (m *MemoryStore) queryStatistics(selector util.Selector, metric string, from, to int64, resampleAlgo string) (APIMetricData, error) {
	minfo, ok := m.Metrics[metric]
	if !ok {
		return APIMetricData{}, errors.New("[METRICSTORE]> unknown metric: " + metric)
	}
	// Read validates this option even when no downsampling is requested.
	if _, err := resampler.GetResampler(resampleAlgo); err != nil {
		return APIMetricData{}, err
	}
	count := 0
	var result APIMetricData
	err := m.root.findBuffers(selector, minfo.offset, func(b *buffer, _ []string) error {
		count++
		if count > 1 {
			return errMultipleStatisticsBuffers
		}
		result = b.statisticsForRead(from, to)
		return nil
	}, nil)
	if errors.Is(err, errMultipleStatisticsBuffers) {
		values, actualFrom, actualTo, resolution, err := m.Read(selector, metric, from, to, 0, resampleAlgo)
		if err != nil {
			return APIMetricData{}, err
		}
		result = APIMetricData{Data: values, From: actualFrom, To: actualTo, Resolution: resolution}
		result.AddStats()
		result.Data = nil
		return result, nil
	}
	if err != nil {
		return APIMetricData{}, err
	}
	if count == 0 {
		return APIMetricData{}, ErrNoHostOrMetric
	}
	return result, nil
}

// statisticsForRead reduces the same sample grid as buffer.read without
// materializing its NaN padding or values. Bounds remain the requested window,
// as in MemoryStore.Read; gaps and unwritten slots do not contribute statistics.
func (b *buffer) statisticsForRead(from, to int64) APIMetricData {
	for from < b.firstWrite() && b.prev != nil {
		b = b.prev
	}
	result := APIMetricData{From: from, To: to, Resolution: b.frequency, Avg: schema.NaN, Min: schema.NaN, Max: schema.NaN}
	n, sum, minimum, maximum := 0, 0.0, math.MaxFloat64, -math.MaxFloat64
	for t := from; t < to; t += b.frequency {
		idx := int((t - b.start) / b.frequency)
		if idx >= cap(b.data) {
			if b.next == nil {
				continue
			}
			b = b.next
			idx = int((t - b.start) / b.frequency)
		}
		if idx >= len(b.data) || t < b.start {
			continue
		}
		// Read adds into a zero-initialized array, which normalizes negative zero.
		value := float64(schema.Float(0) + b.data[idx])
		if math.IsNaN(value) {
			continue
		}
		n++
		sum += value
		minimum = math.Min(minimum, value)
		maximum = math.Max(maximum, value)
	}
	if n > 0 {
		result.Avg, result.Min, result.Max = schema.Float(sum/float64(n)), schema.Float(minimum), schema.Float(maximum)
	}
	return result
}
