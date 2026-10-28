package tagger

import (
	"github.com/ClusterCockpit/cc-lib/v2/schema"
	"testing"
)

func TestJobClassMissingMetricConfigurationDoesNotPanic(t *testing.T) {
	tagger := &JobClassTagger{
		rules: make(map[string]ruleInfo),
		getStatistics: func(*schema.Job) (schema.JobStatisticsSet, error) {
			return schema.JobStatisticsSet{Metrics: map[string]schema.JobStatistics{"removed": {Avg: 1}}}, nil
		},
		getMetricConfig: func(string, string) map[string]*schema.Metric { return nil },
	}
	tagger.prepareRule([]byte(`{"tag":"removed","metrics":["removed"],"rule":"false","hint":"ok"}`), "removed.json")
	id := int64(1)
	tagger.Match(&schema.Job{ID: &id})
}
