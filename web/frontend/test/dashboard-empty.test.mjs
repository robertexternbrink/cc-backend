// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from 'node:assert/strict';
import test from 'node:test';
import {renderHeader} from './helpers/source.mjs';

test('dashboard headers render only metrics present in the response', async () => {
  const metric = {name: 'flops_any', unit: {prefix: 'G', base: 'F/s'}};
  for (const metrics of [[], [metric], [metric, {...metric, name: 'mem_bw'}]]) {
    const text = await renderHeader('src/DashPublic.root.svelte', 'Cluster Utilization', {sortedClusterMetrics: metrics});
    assert.doesNotMatch(text, /undefined/);
    if (metrics.length === 0) assert.equal(text, '');
    else assert.ok(text.includes('flops_any (GF/s)'));
    if (metrics.length === 2) assert.ok(text.includes('mem_bw (GF/s)'));
  }
});
