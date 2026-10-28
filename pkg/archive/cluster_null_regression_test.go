// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package archive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ClusterCockpit/cc-backend/internal/config"
)

func TestClusterInitializationRejectsNullDefinitions(t *testing.T) {
	originalArchive := ar
	originalClusters, originalMetrics, originalUserMetrics, originalLists := Clusters, GlobalMetricList, GlobalUserMetricList, NodeLists
	originalValidation := config.Keys.Validate
	config.Keys.Validate = false
	t.Cleanup(func() {
		ar = originalArchive
		Clusters, GlobalMetricList, GlobalUserMetricList, NodeLists = originalClusters, originalMetrics, originalUserMetrics, originalLists
		config.Keys.Validate = originalValidation
	})
	fixture, err := os.ReadFile("testdata/archive/fritz/cluster.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"valid", "metric", "subcluster", "override"} {
		t.Run(kind, func(t *testing.T) {
			var document map[string]any
			if err := json.Unmarshal(fixture, &document); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "metric":
				document["metricConfig"].([]any)[0] = nil
			case "subcluster":
				document["subClusters"].([]any)[0] = nil
			case "override":
				document["metricConfig"].([]any)[0].(map[string]any)["subClusters"] = []any{nil}
			}
			data, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			path := filepath.Join(root, "fixture", "cluster.json")
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			ar = &FsArchive{path: root, clusters: []string{"fixture"}}
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("invalid configuration must return an error, not panic: %v", recovered)
				}
			}()
			err = initClusterConfig()
			if (err != nil) != (kind != "valid") {
				t.Fatalf("configuration kind %s returned error=%v", kind, err)
			}
		})
	}
}
