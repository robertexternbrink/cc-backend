// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {parse} from 'svelte/compiler';

export const frontend = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');

function parsed(relative) {
  let source = fs.readFileSync(path.join(frontend, relative), 'utf8');
  if (relative.endsWith('.js')) source = '<script>' + source + '</script>';
  return {source, ast: parse(source)};
}

function declared(relative, name) {
  const {source, ast} = parsed(relative);
  for (const entry of ast.instance.content.body) {
    const node = entry.type === 'ExportNamedDeclaration' ? entry.declaration : entry;
    if (node?.type === 'FunctionDeclaration' && node.id.name === name) return {source, node};
    if (node?.type === 'VariableDeclaration') {
      const node = (entry.declaration || entry).declarations.find(d => d.id.name === name);
      if (node) return {source, node: node.init};
    }
  }
  throw new Error(`No declaration ${name} in ${relative}`);
}

function evaluate(source, node, context) {
  return Function(...Object.keys(context), `return (${source.slice(node.start, node.end)});`)(...Object.values(context));
}

// Execute the actual declared business function, without bootstrapping a page.
export function businessFunction(relative, name, context = {}) {
  const {source, node} = declared(relative, name);
  return evaluate(source, node, context);
}

export function declarationSource(relative, name) {
  const {source, node} = declared(relative, name);
  return source.slice(node.start, node.end);
}

export function derived(relative, name, context = {}) {
  const {source, node} = declared(relative, name);
  const value = evaluate(source, node.arguments[0], context);
  return node.callee.type === 'MemberExpression' ? value() : value;
}

export function templateExpressions(relative, containing, context) {
  const {source, ast} = parsed(relative);
  const values = [];
  function visit(node) {
    if (!node || typeof node !== 'object') return;
    if (node.type === 'MustacheTag' && source.slice(node.expression.start, node.expression.end).includes(containing)) {
      values.push(evaluate(source, node.expression, context));
    }
    for (const value of Object.values(node)) {
      if (Array.isArray(value)) value.forEach(visit);
      else if (value && typeof value === 'object') visit(value);
    }
  }
  visit(ast.html);
  return values;
}
