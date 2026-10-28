// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package tagger

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ClusterCockpit/cc-backend/internal/repository"
	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

const reloadRule = `{"tag":"keep","parameters":["threshold"],"rule":"false","hint":"ok"}`

func writeReloadFile(t *testing.T, root, name, contents string) {
	t.Helper()
	noErr(t, os.WriteFile(filepath.Join(root, name), []byte(contents), 0o600))
}

func classReloadFixture(t *testing.T) *JobClassTagger {
	t.Helper()
	cclog.Init("warn", true)
	root := t.TempDir()
	writeReloadFile(t, root, "parameters.json", `{"threshold":1,"removed":2}`)
	writeReloadFile(t, root, "rule.json", reloadRule)
	tagger := &JobClassTagger{
		cfgPath: root, parameters: make(map[string]any),
		getStatistics:   func(*schema.Job) (schema.JobStatisticsSet, error) { return schema.JobStatisticsSet{}, nil },
		getMetricConfig: func(string, string) map[string]*schema.Metric { return nil },
	}
	tagger.EventCallback()
	if len(tagger.rules) != 1 {
		t.Fatal("initial rules did not load")
	}
	return tagger
}

func TestJobClassReloadRejectsPartialConfiguration(t *testing.T) {
	tagger := classReloadFixture(t)
	writeReloadFile(t, tagger.cfgPath, "rule.json", `{"tag":`)
	tagger.EventCallback()
	if len(tagger.rules) != 1 || tagger.rules["keep"].rule == nil {
		t.Fatal("invalid update discarded the previous working rules")
	}
}

func TestJobClassReloadRemovesDeletedParameters(t *testing.T) {
	tagger := classReloadFixture(t)
	writeReloadFile(t, tagger.cfgPath, "parameters.json", `{"threshold":3}`)
	tagger.EventCallback()
	if _, exists := tagger.parameters["removed"]; exists {
		t.Fatal("deleted parameter survived reload")
	}
	if tagger.parameters["threshold"] != float64(3) {
		t.Fatal("new parameter did not take effect")
	}
}

func TestAppReloadRejectsInvalidPattern(t *testing.T) {
	root := t.TempDir()
	writeReloadFile(t, root, "example.txt", "example")
	tagger := &AppTagger{cfgPath: root}
	tagger.EventCallback()
	writeReloadFile(t, root, "example.txt", "[")
	tagger.EventCallback()
	if len(tagger.apps) != 1 || len(tagger.apps[0].patterns) != 1 || !tagger.apps[0].patterns[0].MatchString("example") {
		t.Fatal("invalid update replaced the previous working patterns")
	}
}

func TestMissingTaggerDirectoryDoesNotExitProcess(t *testing.T) {
	if kind := os.Getenv("CC_TEST_MISSING_TAGGER_DIR"); kind != "" {
		root := filepath.Join(t.TempDir(), "missing")
		if kind == "app" {
			(&AppTagger{cfgPath: root}).EventCallback()
		} else {
			(&JobClassTagger{cfgPath: root}).EventCallback()
		}
		return
	}
	for _, kind := range []string{"app", "class"} {
		command := exec.Command(os.Args[0], "-test.run=^TestMissingTaggerDirectoryDoesNotExitProcess$")
		command.Env = append(os.Environ(), "CC_TEST_MISSING_TAGGER_DIR="+kind)
		if output, err := command.CombinedOutput(); err != nil {
			t.Errorf("%s reload terminated the process: %v\n%s", kind, err, output)
		}
	}
}

func TestJobClassReloadConcurrentWithClassification(t *testing.T) {
	tagger := classReloadFixture(t)
	id := int64(1)
	job := &schema.Job{ID: &id}
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		for range 100 {
			tagger.EventCallback()
		}
	}()
	go func() {
		defer workers.Done()
		for range 1000 {
			tagger.Match(job)
		}
	}()
	workers.Wait()
}

func TestAppReloadConcurrentWithMatching(t *testing.T) {
	cclog.Init("warn", true)
	// The historical race probe must not migrate or write the source fixture.
	noErr(t, repository.ResetConnection())
	database := filepath.Join(t.TempDir(), "jobs.db")
	data, err := os.ReadFile("../repository/testdata/job.db")
	noErr(t, err)
	noErr(t, os.WriteFile(database, data, 0o600))
	noErr(t, repository.MigrateDB(database))
	repository.Connect(database)
	t.Cleanup(func() { noErr(t, repository.ResetConnection()) })
	repo := repository.GetJobRepository()
	job, err := repo.FindByIDDirect(317)
	noErr(t, err)
	root := t.TempDir()
	writeReloadFile(t, root, "never.txt", "nevermatchthisjob")
	tagger := &AppTagger{cfgPath: root}
	tagger.EventCallback()
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		for range 100 {
			tagger.EventCallback()
		}
	}()
	go func() {
		defer workers.Done()
		for range 1000 {
			tagger.Match(job)
		}
	}()
	workers.Wait()
}
