package metricdispatch

import (
	"github.com/ClusterCockpit/cc-lib/v2/lrucache"
	"github.com/ClusterCockpit/cc-lib/v2/schema"
	"strings"
	"testing"
	"time"
)

func TestDeviceIdentifiersRespectPlotCacheBudget(t *testing.T) {
	const budget = 256 * 1024
	id := strings.Repeat("mount/", budget)
	data := schema.JobData{Metrics: map[string]schema.ScopedMetrics{"fs_read_bw": {
		schema.MetricScopeFilesystem: {Timestep: 10, Series: []schema.Series{{Hostname: "fixture", ID: &id}}},
	}}}
	cache := lrucache.New(budget)
	cache.Put("oversized-device-id", data, plotCacheSize(data), time.Minute)
	if cache.Get("oversized-device-id", nil) != nil {
		t.Fatal("device identifier larger than entire cache budget was admitted")
	}
	data.Metrics["fs_read_bw"][schema.MetricScopeFilesystem].Series[0].ID = nil
	cache.Put("small", data, plotCacheSize(data), time.Minute)
	if cache.Get("small", nil) == nil {
		t.Fatal("small series unexpectedly rejected")
	}
}
