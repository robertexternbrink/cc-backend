// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from 'node:assert/strict';
import test from 'node:test';
import {runUsageDashboardFixture} from './helpers/usage-fixture.mjs';

test('usage refresh requests new top statistics as well as histograms', async () => {
  const result = await runUsageDashboardFixture('normal');
  assert.deepEqual(result.errors, []);
  assert.equal(result.initial.requests.filter(r => r.kind === 'top').length, 1);
  assert.equal(result.refreshed.requests.filter(r => r.kind === 'hist').length, 2);
  assert.equal(result.refreshed.requests.filter(r => r.kind === 'top').length, 2);
  assert.ok(result.refreshed.charts.some(c => c.labels.includes('fresh')));
});
