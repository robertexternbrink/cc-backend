// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from 'node:assert/strict';
import test from 'node:test';
import {derived} from './helpers/source.mjs';

test('power units survive CPU-only nodes before GPU nodes', () => {
  const rate = {value: 1, unit: {prefix: '', base: 'B/s'}};
  const config = {clusters: [{name: 'c', subClusters: [{name: 'cpu', numberOfNodes: 2,
    socketsPerNode: 1, coresPerSocket: 2, processorType: 'Test', topology: {accelerators: []},
    flopRateScalar: rate, flopRateSimd: rate, memoryBandwidth: rate}]}]};
  const power = name => ({name, metric: {unit: {prefix: '', base: 'W'}, series: [{statistics: {avg: 250}}]}});
  const cpu = {metrics: [power('cpu_power')]};
  const gpu = {metrics: [power('acc_power')]};
  for (const nodes of [[cpu, gpu], [gpu, cpu], [{metrics: []}, cpu, gpu]]) {
    const info = derived('src/DashPublic.root.svelte', 'clusterInfo', {
      $initq: {data: config}, $statusQuery: {data: {nodeMetrics: nodes}}, presetCluster: 'c',
    });
    assert.equal(info.cpuPwr, 250);
    assert.equal(info.gpuPwr, 250);
    assert.equal(info.cpuPwrUnitBase, 'W');
    assert.equal(info.gpuPwrUnitBase, 'W');
  }
});
