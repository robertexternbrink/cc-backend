// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from 'node:assert/strict';
import test from 'node:test';
import {businessFunction} from './helpers/source.mjs';

test('dashboard roofline point and job metadata use the same filtered positions', () => {
  const data = businessFunction('src/status/DashInternal.svelte', 'transformJobsStatsToData');
  const info = businessFunction('src/status/DashInternal.svelte', 'transformJobsStatsToInfo', {formatDurationTime: String});
  const job = (id, f, m) => ({id, jobId: id, numNodes: 1, duration: 60,
    stats: [{name: 'flops_any', data: {avg: f}}, {name: 'mem_bw', data: {avg: m}}]});
  for (const jobs of [[job(1, 0, 0), job(2, 10, 5)], [job(2, 10, 5), job(1, 0, 0)]]) {
    assert.deepEqual(data(jobs)[1], [[2], [10]]);
    assert.deepEqual(info(jobs).map(j => j.id), [2]);
  }
  assert.deepEqual(info([job(1, 0, 0), job(3, 10, 0)]), []);
});
