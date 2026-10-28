// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from 'node:assert/strict';
import test from 'node:test';
import {derived} from './helpers/source.mjs';

test('pie owns the arrays that Chart.js is allowed to instrument', () => {
  const entities = Object.freeze(['allocated', 'idle']);
  const quantities = Object.freeze([1, 3]);
  const data = derived('src/generic/plots/Pie.svelte', 'data', {
    entities, quantities, sliceLabel: 'Nodes', colorPalette: ['green', 'blue'],
  });
  assert.notEqual(data.labels, entities);
  assert.notEqual(data.datasets[0].data, quantities);
  Object.defineProperty(data.datasets[0].data, '_chartjs', {value: {}, configurable: true});
  data.labels.push('down');
  assert.deepEqual(entities, ['allocated', 'idle']);
  assert.deepEqual(quantities, [1, 3]);
  assert.deepEqual(data.datasets[0].data, [1, 3]);
});
