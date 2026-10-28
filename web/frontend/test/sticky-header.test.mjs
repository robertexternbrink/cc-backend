// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from 'node:assert/strict';
import test from 'node:test';
import {businessFunction} from './helpers/source.mjs';

function fixture() {
  const frames = new Map();
  const padding = [];
  let scroll, destroy, id = 0;
  let table = {getBoundingClientRect: () => ({top: 10})};
  const document = {
    querySelector: selector => selector === 'header > nav.navbar' ? {clientHeight: 30} : table,
    addEventListener: (_, fn) => { scroll = fn; },
    removeEventListener: (_, fn) => { assert.equal(scroll, fn); scroll = null; },
  };
  const window = {requestAnimationFrame: fn => { frames.set(id, fn); return id++; },
    cancelAnimationFrame: id => frames.delete(id)};
  const sticky = businessFunction('src/generic/utils.js', 'stickyHeader', {
    document, window, onDestroy: fn => { destroy = fn; },
  });
  sticky('#table-head', value => padding.push(value));
  return {frames, padding, scroll: () => scroll(), destroy: () => destroy(),
    removeTable: () => { table = null; },
    flush: () => { const callbacks = [...frames.values()]; frames.clear(); callbacks.forEach(fn => fn()); }};
}

test('sticky header cancels an already queued frame on teardown', () => {
  const f = fixture();
  f.scroll();
  f.scroll();
  assert.equal(f.frames.size, 1);
  f.removeTable();
  f.destroy();
  assert.equal(f.frames.size, 0);
  f.flush();
  assert.deepEqual(f.padding, []);
});

test('sticky header skips a missing table and preserves normal padding', () => {
  const f = fixture();
  f.scroll();
  f.flush();
  assert.deepEqual(f.padding, [30]);
  f.removeTable();
  f.scroll();
  assert.doesNotThrow(f.flush);
  assert.deepEqual(f.padding, [30]);
  f.destroy();
});
