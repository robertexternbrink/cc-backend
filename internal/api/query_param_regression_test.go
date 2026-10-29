// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package api

import (
	"net/url"
	"testing"
)

func TestWriteClusterParameterPreservesURLSemantics(t *testing.T) {
	for _, raw := range []string{
		"cluster=testcluster", "cluster=test%63luster", "cl%75ster=testcluster",
		"cluster=a+b", "cluster=a%2Bb", "cluster=%ZZ&cluster=valid",
		"cluster=bad;value&cluster=valid", "ignored=%ZZ&cluster=valid",
		"cluster=first&cluster=second", "cluster=", "other=value",
	} {
		values, _ := url.ParseQuery(raw)
		if got, want := queryParam(raw, "cluster"), values.Get("cluster"); got != want {
			t.Errorf("query %q: got %q; want %q", raw, got, want)
		}
	}
}

func FuzzWriteClusterParameterMatchesURLParsing(f *testing.F) {
	for _, raw := range []string{"cluster=testcluster", "cluster=a%2Bb", "cluster=%ZZ&cluster=valid", "a=b;cluster=c"} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		values, _ := url.ParseQuery(raw)
		if got, want := queryParam(raw, "cluster"), values.Get("cluster"); got != want {
			t.Fatalf("query %q: got %q; want %q", raw, got, want)
		}
	})
}
