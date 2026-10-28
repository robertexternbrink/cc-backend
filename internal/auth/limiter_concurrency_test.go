// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ClusterCockpit/cc-backend/internal/repository"
	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

type rejectedAuditAuthenticator struct {
	attempts atomic.Int64
}

func (a *rejectedAuditAuthenticator) CanLogin(user *schema.User, username string, _ http.ResponseWriter, _ *http.Request) (*schema.User, bool) {
	return user, username != ""
}

func (a *rejectedAuditAuthenticator) Login(_ *schema.User, _ http.ResponseWriter, _ *http.Request) (*schema.User, error) {
	a.attempts.Add(1)
	return nil, errors.New("synthetic credential rejection")
}

func auditLoginRepository(t *testing.T) {
	t.Helper()
	fixture, err := os.ReadFile(filepath.Join("..", "repository", "testdata", "job.db"))
	if err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(t.TempDir(), "audit-limiter.db")
	if err := os.WriteFile(dbPath, fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := repository.MigrateDB(dbPath); err != nil {
		t.Fatal(err)
	}
	repository.Connect(dbPath)
	t.Cleanup(func() { _ = repository.ResetConnection() })
}

func auditRejectedLoginHandler() (http.Handler, *rejectedAuditAuthenticator) {
	rejected := &rejectedAuditAuthenticator{}
	a := &Authentication{authenticators: []Authenticator{rejected}}
	return a.Login(func(w http.ResponseWriter, _ *http.Request, err error) {
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}), rejected
}

func auditLoginAttempt(h http.Handler, username, remoteIP string) int {
	form := url.Values{"username": {username}, "password": {"audit-invalid-password"}}
	r := httptest.NewRequest(http.MethodPost, "http://audit.invalid/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.RemoteAddr = remoteIP + ":43210"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code
}

// TestLoginLimiterConcurrentExistingAndCleanup uses the real Login handler and
// the exact cleanup cutoff called by the production hourly ticker.
func TestLoginLimiterConcurrentExistingAndCleanup(t *testing.T) {
	auditLoginRepository(t)
	h, rejected := auditRejectedLoginHandler()
	username, ip := "audit-existing-limiter", "192.0.2.69"
	getIPUserLimiter(ip, username)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 12 {
		wg.Go(func() {
			<-start
			for range 30 {
				if status := auditLoginAttempt(h, username, ip); status != http.StatusUnauthorized {
					t.Errorf("rejected credentials status=%d", status)
				}
			}
		})
	}
	wg.Go(func() {
		<-start
		for range 360 {
			cleanupOldRateLimiters(time.Now().Add(-24 * time.Hour))
		}
	})
	close(start)
	wg.Wait()
	if got := rejected.attempts.Load(); got != 5 {
		t.Fatalf("authentication attempts=%d, want 5", got)
	}
}
