// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from 'node:assert/strict';
import test from 'node:test';
import {drawCallbacks} from './helpers/plot-source.mjs';

test('metric draw restores the incoming Canvas state', () => {
  const callbacks = drawCallbacks('src/generic/plots/MetricPlot.svelte', {
    scope: 'node', plotSeries: [{}, {}], useStatsSeries: false, usesMeanStatsSeries: false,
    metricConfig: {scope: 'node'}, isShared: false, forNode: true,
    thresholds: {normal: 10}, lineWidth: 1, normalLineColor: 'green',
  });
  assert.equal(callbacks.length, 1);
  let depth = 0;
  const ctx = {save: () => depth++, restore: () => depth--,
    fillText() {}, setLineDash() {}, beginPath() {}, moveTo() {}, lineTo() {}, stroke() {}};
  for (let draw = 0; draw < 3; draw++) {
    callbacks[0]({ctx, bbox: {left: 0, top: 0, width: 100}, valToPos: () => 20});
    assert.equal(depth, 0);
  }
});
