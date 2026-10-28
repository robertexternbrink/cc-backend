// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from 'node:assert/strict';
import test from 'node:test';
import {componentPropValues} from './helpers/plot-source.mjs';

test('both dashboards forward the actual cluster metric timestep', () => {
  for (const file of ['src/DashPublic.root.svelte', 'src/status/DashInternal.svelte']) {
    for (const timestep of [30, 60, 120, 300]) {
      const context = {$statusQuery: {data: {clusterMetrics: {metrics: [{timestep}]}}}};
      assert.deepEqual(componentPropValues(file, 'DoubleMetric', 'timestep', context), [timestep], file);
    }
  }
});
