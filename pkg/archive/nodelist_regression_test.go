// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package archive

import (
	"reflect"
	"testing"
)

func TestNodeListExpandsAllRangeDimensionsAndSuffixes(t *testing.T) {
	list, err := ParseNodeList("rack[01-02]node[01-02]gpu,login01")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"rack01node01gpu", "rack01node02gpu", "rack02node01gpu", "rack02node02gpu", "login01"}
	if got := list.PrintList(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expanded list %v; want %v", got, want)
	}
	if got := list.NodeCount(); got != len(want) {
		t.Fatalf("node count %d; want %d", got, len(want))
	}
	for _, host := range want {
		if !list.Contains(host) {
			t.Fatalf("expanded hostname %s does not match the original pattern", host)
		}
	}
}

func TestNodeListRejectsEmptyTermsAndDescendingRanges(t *testing.T) {
	for _, text := range []string{",node01", "node01,,node02", "node[09-01]"} {
		if _, err := ParseNodeList(text); err == nil {
			t.Errorf("accepted malformed node expression %q", text)
		}
	}
}
