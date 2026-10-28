// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import fs from 'node:fs';
import path from 'node:path';
import {parse} from 'svelte/compiler';
import {frontend} from './source.mjs';

function evaluate(source, node, context) {
  return Function(...Object.keys(context), `return (${source.slice(node.start, node.end)});`)(...Object.values(context));
}

function visit(node, callback) {
  if (!node || typeof node !== 'object') return;
  callback(node);
  for (const value of Object.values(node)) {
    if (Array.isArray(value)) value.forEach(n => visit(n, callback));
    else if (value && typeof value === 'object') visit(value, callback);
  }
}

export function drawCallbacks(relative, context) {
  const source = fs.readFileSync(path.join(frontend, relative), 'utf8');
  const callbacks = [];
  visit(parse(source).instance.content, node => {
    if (node.type === 'Property' && node.key.name === 'draw' && node.value.type === 'ArrayExpression') {
      callbacks.push(...node.value.elements.map(n => evaluate(source, n, context)));
    }
  });
  return callbacks;
}

export function componentPropValues(relative, component, prop, context) {
  const source = fs.readFileSync(path.join(frontend, relative), 'utf8');
  const values = [];
  visit(parse(source).html, node => {
    if (node.type === 'InlineComponent' && node.name === component) {
      const attribute = node.attributes.find(a => a.name === prop);
      if (attribute) values.push(evaluate(source, attribute.value[0].expression, context));
    }
  });
  return values;
}
