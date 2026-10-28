// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from 'node:assert/strict';
import test from 'node:test';
import {businessFunction, derived} from './helpers/source.mjs';

test('plot legends and axes do not invent optional unit text', () => {
  const context = {publicMode: false, sortedMetricData: [{name: 'load', unit: null}, {name: 'memory', unit: null}],
    lineWidth: 1, fixedLineColors: ['blue', 'red'], timestep: 60, maxX: 180, forNode: true,
    timeIncrs: () => [], legendAsTooltipPlugin: () => ({}), cluster: 'c', numNodes: 1};
  context.plotSeries = derived('src/generic/plots/DoubleMetricPlot.svelte', 'plotSeries', context);
  for (const series of context.plotSeries) assert.doesNotMatch(series.label, /undefined/);
  const opts = businessFunction('src/generic/plots/DoubleMetricPlot.svelte', 'getOpts', context)(400, 300);
  assert.doesNotMatch(opts.axes[1].label, /undefined/);
  assert.doesNotMatch(opts.axes[2].label, /undefined/);
});
