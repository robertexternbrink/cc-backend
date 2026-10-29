// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package metricstore

import (
	"math"
	"runtime"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

func sameStatisticsFloat(a, b schema.Float) bool {
	return math.Float64bits(float64(a)) == math.Float64bits(float64(b)) || a.IsNaN() && b.IsNaN()
}

func checkStatisticsOnly(t *testing.T, req APIQueryRequest) {
	t.Helper()
	req.WithData, req.WithStats = true, true
	full, err := FetchData(req)
	if err != nil {
		t.Fatal(err)
	}
	req.WithData = false
	stats, err := FetchData(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(full.Results) != len(stats.Results) {
		t.Fatal("query count changed")
	}
	for i, row := range full.Results {
		if len(row) != len(stats.Results[i]) {
			t.Fatalf("query %d selector count changed", i)
		}
		for j, want := range row {
			got := stats.Results[i][j]
			if len(got.Data) != 0 {
				t.Fatalf("query %d selector %d returned %d unrequested samples", i, j, len(got.Data))
			}
			if (got.Error == nil) != (want.Error == nil) {
				t.Fatalf("query %d selector %d error changed", i, j)
			}
			if want.Error != nil {
				if *got.Error != *want.Error {
					t.Fatal("error message changed")
				}
				continue
			}
			if got.From != want.From || got.To != want.To || got.Resolution != want.Resolution || !sameStatisticsFloat(got.Avg, want.Avg) || !sameStatisticsFloat(got.Min, want.Min) || !sameStatisticsFloat(got.Max, want.Max) {
				t.Fatalf("query %d selector %d stats-only [%d,%d)/%d %v/%v/%v, full [%d,%d)/%d %v/%v/%v", i, j, got.From, got.To, got.Resolution, got.Min, got.Avg, got.Max, want.From, want.To, want.Resolution, want.Min, want.Avg, want.Max)
			}
		}
	}
}

func TestStatisticsOnlyHonorsDataFlagAndPreservesReadSemantics(t *testing.T) {
	ms := newFetchStatsStore(t, 3)
	// Cover native gaps, overwrites, linked buffers, unaligned windows and NaNs.
	for i := fsCount; i < 2*BufferCap+20; i++ {
		if i%97 == 0 {
			continue
		}
		value := schema.Float(i%17) * 0.125
		if i%43 == 0 {
			value = schema.NaN
		}
		if err := ms.Write([]string{"cl", "n1"}, fsFrom+int64(i)*fsFreq, []Metric{{Name: "load", Value: value}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, window := range [][2]int64{{-20, 31}, {0, 512}, {1, 800}, {511, 1044}, {1045, 1080}, {1080, 1090}, {7, 7}} {
		req := APIQueryRequest{Cluster: "cl", From: fsFrom + window[0]*fsFreq + 1, To: fsFrom + window[1]*fsFreq + 1, Queries: []APIQuery{{Metric: "load", Hostname: "n1"}}}
		checkStatisticsOnly(t, req)
	}
	for _, q := range []APIQuery{coreQuery("flops", false), coreQuery("temp", false), {Metric: "load", Hostname: "absent"}, {Metric: "absent", Hostname: "n1"}, {Metric: "load", Hostname: "n1", Resolution: 20}} {
		checkStatisticsOnly(t, APIQueryRequest{Cluster: "cl", From: fsFrom, To: fsFrom + fsCount*fsFreq, Queries: []APIQuery{q}, ResampleAlgo: "average"})
	}
	checkStatisticsOnly(t, APIQueryRequest{Cluster: "cl", From: fsFrom, To: fsFrom + fsCount*fsFreq, Queries: []APIQuery{{Metric: "load", Hostname: "n1"}}, ResampleAlgo: "unknown"})
}

func TestStatisticsOnlyNativeDoesNotAllocateSampleArray(t *testing.T) {
	newFetchStatsStore(t, 0)
	req := APIQueryRequest{Cluster: "cl", From: fsFrom, To: fsFrom + 32768*fsFreq, WithStats: true, Queries: []APIQuery{{Metric: "load", Hostname: "n1"}}}
	// A result array for this window alone costs 256 KiB. Read the process's
	// allocation counter around several calls to distinguish that cost from
	// constant query/selector allocations, with a wide margin for the harness.
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range 8 {
		result, err := FetchData(req)
		if err != nil {
			t.Fatal(err)
		}
		runtime.KeepAlive(result)
	}
	runtime.ReadMemStats(&after)
	if perCall := (after.TotalAlloc - before.TotalAlloc) / 8; perCall > 64*1024 {
		t.Fatalf("native statistics allocated %d bytes per call; sample arrays must not be materialized", perCall)
	}
}

func TestStatisticsOnlySpecialValuesAndExplicitData(t *testing.T) {
	for _, values := range [][]schema.Float{{schema.NaN, schema.NaN}, {schema.Float(math.Copysign(0, -1))}, {schema.Float(math.Inf(1)), schema.Float(math.Inf(-1))}, {math.MaxFloat64, -math.MaxFloat64, math.SmallestNonzeroFloat64}} {
		ms := newFetchStatsStore(t, 0)
		b := ms.GetLevel([]string{"cl", "n1"}).metrics[0]
		b.data = append(b.data[:0], values...)
		b.statsValid = false
		req := APIQueryRequest{Cluster: "cl", From: fsFrom, To: fsFrom + int64(len(values))*fsFreq, Queries: []APIQuery{{Metric: "load", Hostname: "n1"}}}
		checkStatisticsOnly(t, req)
		req.WithData, req.WithStats = true, true
		got, err := FetchData(req)
		if err != nil || len(got.Results[0][0].Data) != len(values) {
			t.Fatalf("explicit data request changed: %v", err)
		}
	}
}
