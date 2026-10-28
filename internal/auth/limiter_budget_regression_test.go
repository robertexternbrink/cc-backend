// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

// TestLoginLimiterConcurrentFirstUseBudget checks the advertised five-attempt
// budget at first use of a new identity, through actual handler invocations.
func TestLoginLimiterConcurrentFirstUseBudget(t *testing.T) {
	auditLoginRepository(t)
	for round := range 100 {
		h, rejected := auditRejectedLoginHandler()
		username := fmt.Sprintf("audit-first-use-%d", round)
		start := make(chan struct{})
		var wg sync.WaitGroup
		for range 32 {
			wg.Go(func() {
				<-start
				if status := auditLoginAttempt(h, username, "192.0.2.70"); status != http.StatusUnauthorized {
					t.Errorf("rejected credentials status=%d", status)
				}
			})
		}
		close(start)
		wg.Wait()
		if got := rejected.attempts.Load(); got != 5 {
			t.Fatalf("round %d: fresh identity admitted %d credential attempts, want five", round, got)
		}
	}
}
