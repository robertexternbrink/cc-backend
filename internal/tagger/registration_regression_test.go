// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package tagger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
	"github.com/ClusterCockpit/cc-lib/v2/util"
)

func TestFailedTaggerRegistrationCannotReachJobHooks(t *testing.T) {
	for _, validApp := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing-all", true: "valid-app-missing-class"}[validApp], func(t *testing.T) {
			t.Chdir(t.TempDir())
			previous := jobTagger
			t.Cleanup(func() { jobTagger = previous })
			if validApp {
				if err := os.MkdirAll(defaultConfigPath, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(defaultConfigPath, "audit.txt"), []byte("audit"), 0o600); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(util.FsWatcherShutdown)
			}
			newTagger()
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("failed registration reached a job hook and panicked: %v", recovered)
				}
			}()
			id := int64(1)
			jobTagger.JobStopCallback(&schema.Job{ID: &id})
			wantStart := 0
			if validApp {
				wantStart = 1
			}
			if len(jobTagger.startTaggers) != wantStart || len(jobTagger.stopTaggers) != 0 {
				t.Fatal("registration outcome was not reflected in active taggers")
			}
		})
	}
}
