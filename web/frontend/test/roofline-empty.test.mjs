// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from 'node:assert/strict';
import test from 'node:test';
import {businessFunction} from './helpers/source.mjs';

test('roofline transformers accept empty metric series as no data', () => {
  const job = businessFunction('src/generic/utils.js', 'transformDataForRoofline');
  const nodes = businessFunction('src/generic/utils.js', 'transformPerNodeDataForRoofline');
  assert.equal(job({series: []}, {series: []}), null);
  assert.equal(nodes([{metrics: [
    {name: 'flops_any', scope: 'node', metric: {series: []}},
    {name: 'mem_bw', scope: 'node', metric: {series: []}},
  ]}]), null);
  const ordinary = job({series: [{data: [10]}]}, {series: [{data: [5]}]});
  assert.deepEqual(ordinary, [null, [[2], [10]], [0]]);
});
