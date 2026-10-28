package tagger

import ("testing";"github.com/ClusterCockpit/cc-lib/v2/schema")

func TestInvalidHintIsNotPublished(t *testing.T) {
	tagger := &JobClassTagger{rules: make(map[string]ruleInfo)}
	tagger.prepareRule([]byte(`{"tag":"bad","rule":"true","hint":"{{"}`), "broken.json")
	if _, exists := tagger.rules["bad"]; exists {
		t.Fatal("invalid hint was published and could panic during classification")
	}
}

func TestInvalidHintNeverReachesMatching(t *testing.T) {
	repository := new(MockJobRepository)
	repository.On("HasTag", int64(1), "jobClass", "invalid").Return(true)
	tagger := &JobClassTagger{
		rules: make(map[string]ruleInfo), repo: repository, tagType: "jobClass",
		getStatistics:   func(*schema.Job) (schema.JobStatisticsSet, error) { return schema.JobStatisticsSet{}, nil },
		getMetricConfig: func(string, string) map[string]*schema.Metric { return nil },
	}
	tagger.prepareRule([]byte(`{"tag":"invalid","rule":"true","hint":"{{"}`), "invalid.rule")
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("invalid hint reached the actual matching path and panicked: %v", recovered)
		}
	}()
	id := int64(1)
	tagger.Match(&schema.Job{ID: &id})
}
