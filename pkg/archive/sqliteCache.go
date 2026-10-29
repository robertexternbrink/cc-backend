// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package archive

import (
	"fmt"
	"sync/atomic"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

var sqlitePayloadCacheID atomic.Uint64

func (sa *SqliteArchive) payloadCacheKey(job *schema.Job, validate bool) string {
	id := sa.cacheID.Load()
	if id == 0 {
		sa.cacheID.CompareAndSwap(0, sqlitePayloadCacheID.Add(1))
		id = sa.cacheID.Load()
	}
	return fmt.Sprintf("sqlite:%d:%t:%s:%d:%d", id, validate, job.Cluster, job.JobID, job.StartTime)
}

// The caller holds payloadMu exclusively so an earlier cache miss cannot
// publish an outdated result after the successful mutation is invalidated.
func (sa *SqliteArchive) invalidateJobPayloads(job *schema.Job) {
	for _, validate := range []bool{false, true} {
		key := sa.payloadCacheKey(job, validate)
		cache.Del(key)
		statsCache.Del(key)
	}
}
