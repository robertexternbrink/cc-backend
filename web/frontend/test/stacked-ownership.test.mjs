// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from 'node:assert/strict';
import test from 'node:test';
import {derived} from './helpers/source.mjs';

test('stacked state sorting does not write to the parent array', () => {
  const data = Object.freeze([{state: 'idle'}, {state: 'allocated'}]);
  const sorted = derived('src/generic/plots/Stacked.svelte', 'sortedData', {data});
  assert.deepEqual(sorted.map(s => s.state), ['allocated', 'idle']);
  assert.deepEqual(data.map(s => s.state), ['idle', 'allocated']);
  assert.notEqual(sorted, data);
});
