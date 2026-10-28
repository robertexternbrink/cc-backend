// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package tagger

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

)

func TestJobClassReloadPreservesNonJSONExtensionRules(t *testing.T) {
	tagger := classReloadFixture(t)
	noErr(t, os.Rename(filepath.Join(tagger.cfgPath, "rule.json"), filepath.Join(tagger.cfgPath, "rule.custom")))
	tagger.EventCallback()
	if len(tagger.rules) != 1 || tagger.rules["keep"].rule == nil {
		t.Fatal("historically accepted non-json-extension rule disappeared")
	}
}

func TestJobClassReloadKeepsCompletePreviousSnapshot(t *testing.T) {
	for _, invalid := range []string{`{"tag":`, `{"tag":"new","rule":"true","hint":"{{"}`, `{"tag":"new","rule":"missing("}`, `{"tag":"new","parameters":["unknown"],"rule":"false"}`} {
		t.Run(invalid, func(t *testing.T) {
			tagger := classReloadFixture(t)
			writeReloadFile(t, tagger.cfgPath, "parameters.json", `{"threshold":99}`)
			writeReloadFile(t, tagger.cfgPath, "rule.json", invalid)
			tagger.EventCallback()
			if len(tagger.rules) != 1 || tagger.rules["keep"].rule == nil || tagger.parameters["threshold"] != float64(1) || tagger.rules["keep"].env["threshold"] != float64(1) {
				t.Fatal("rejected reload mixed old rules with new parameters")
			}
		})
	}
}

func TestAppReloadRejectsScannerFailureAndUnopenableFile(t *testing.T) {
	for _, scannerFailure := range []bool{true, false} {
		root := t.TempDir()
		writeReloadFile(t, root, "example.txt", "example")
		tagger := &AppTagger{cfgPath: root}
		tagger.EventCallback()
		if scannerFailure {
			writeReloadFile(t, root, "example.txt", "audit\n"+strings.Repeat("x", bufio.MaxScanTokenSize))
		} else {
			noErr(t, os.Symlink(filepath.Join(root, "missing-target"), filepath.Join(root, "broken.txt")))
		}
		tagger.EventCallback()
		if len(tagger.apps) != 1 || len(tagger.apps[0].patterns) != 1 || !tagger.apps[0].patterns[0].MatchString("example") {
			t.Fatal("read failure partially replaced working patterns")
		}
	}
}

func TestJobClassReloadReflectsDeletedRules(t *testing.T) {
	tagger := classReloadFixture(t)
	noErr(t, os.Remove(filepath.Join(tagger.cfgPath, "rule.json")))
	tagger.EventCallback()
	if len(tagger.rules) != 0 || tagger.parameters["threshold"] != float64(1) {
		t.Fatal("successful empty rule directory did not replace the old rules")
	}
}

