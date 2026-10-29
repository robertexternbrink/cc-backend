// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFreeTimestampValidationUsesClientErrors(t *testing.T) {
	for _, test := range []struct {
		name, query string
		status      int
	}{
		{"missing", "", http.StatusBadRequest},
		{"invalid", "?to=abc", http.StatusBadRequest},
		{"overflow", "?to=9223372036854775808", http.StatusBadRequest},
		{"valid", "?to=5", http.StatusOK},
		{"negative", "?to=-1", http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Empty selectors exercise input handling without deleting any buffers.
			request := httptest.NewRequest(http.MethodPost, "/api/free"+test.query, strings.NewReader("[]"))
			response := httptest.NewRecorder()
			freeMetrics(response, request)
			if response.Code != test.status {
				t.Fatalf("status %d, want %d: %s", response.Code, test.status, response.Body.String())
			}
		})
	}
}
