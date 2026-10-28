// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from 'node:assert/strict';
import test from 'node:test';
import {businessFunction} from './helpers/source.mjs';

test('statistics entries sort a local array without mutating parent state', () => {
  const data = Object.freeze([{id: '10', data: {avg: 1}}, {id: '2', data: {avg: 3}}]);
  const sort = businessFunction('src/job/statstab/StatsTableEntry.svelte', 'updateData');
  const sorted = sort(data, 'id', 'down');
  assert.deepEqual(sorted.map(s => s.id), ['2', '10']);
  assert.deepEqual(data.map(s => s.id), ['10', '2']);
  assert.notEqual(sorted, data);
  assert.deepEqual(sort(data, 'avg', 'up').map(s => s.id), ['2', '10']);
});
