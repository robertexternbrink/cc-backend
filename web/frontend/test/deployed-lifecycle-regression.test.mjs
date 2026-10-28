// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import { parse } from "svelte/compiler";

const frontend = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
function script(relative) {
  const raw = fs.readFileSync(path.join(frontend, "src", relative), "utf8");
  const source = relative.endsWith(".js") ? `<script>${raw}</script>` : raw;
  return { source, body: parse(source).instance.content.body };
}
function evaluate(source, node, context) {
  return Function(...Object.keys(context), `return (${source.slice(node.start,node.end)});`)(...Object.values(context));
}
function hook(relative, name, context) {
  const { source, body } = script(relative);
  const call = body.find(node => node.type === "ExpressionStatement" && node.expression.type === "CallExpression" && node.expression.callee.name === name);
  assert.ok(call, `missing ${name} hook in ${relative}`);
  return evaluate(source, call.expression.arguments[0], context);
}

for (const component of ["Pie", "Polar"]) {
  test(`chart cleanup: ${component} releases its owned chart`, () => {
    let alive = 0;
    class Chart {
      constructor() { alive++; }
      destroy() { alive--; }
    }
    const mount = hook(`generic/plots/${component}.svelte`, "onMount", {
      Chart, canvasId: "owned-canvas", document: { getElementById: () => ({}) }, data: {}, options: {},
    });
    for (let cycle = 0; cycle < 3; cycle++) {
      const cleanup = mount();
      assert.equal(alive, 1);
      assert.equal(typeof cleanup, "function", "mounted chart has no cleanup");
      cleanup();
      assert.equal(alive, 0, "chart survives unmount");
    }
  });
}
for (const [component, storeName, field] of [["generic/JobList.svelte", "$jobsStore", "jobs"], ["systems/NodeList.svelte", "$nodesStore", "nodeMetricsList"]]) {
  test(`scroll cleanup: ${component} removes exactly its listener`, () => {
    const listeners = new Set();
    const window = { addEventListener(type, fn) { assert.equal(type, "scroll"); listeners.add(fn); }, removeEventListener(type, fn) { assert.equal(type,"scroll"); listeners.delete(fn); } };
    const effect = hook(component, "$effect", { usePaging: false, window, document: { documentElement: { scrollTop: 0, scrollHeight: 100, clientHeight: 100 } }, [storeName]: { data: { [field]: { hasNextPage: true } } }, page: 1 });
    for (let cycle = 0; cycle < 3; cycle++) {
      const cleanup = effect();
      assert.equal(listeners.size, 1);
      assert.equal(typeof cleanup, "function", "scroll listener has no cleanup");
      cleanup();
      assert.equal(listeners.size, 0);
    }
  });
}
test("timer cleanup: Refresher clears its active timer including handle zero", () => {
  const { source, body } = script("generic/helper/Refresher.svelte");
  const code = body.filter(node => node.type !== "ImportDeclaration").map(node => source.slice(node.start,node.end)).join("\n");
  const owned = new Set(), destroy = [];
  const context = {
    $props: () => ({ initially: 1, onRefresh() {} }), $derived: value => value, $effect: fn => fn(),
    onDestroy: fn => destroy.push(fn), setInterval: () => { owned.add(0); return 0; }, clearInterval: id => owned.delete(id),
  };
  Function(...Object.keys(context), code)(...Object.values(context));
  assert.equal(owned.size, 1);
  assert.equal(destroy.length, 1, "periodic refresh has no unmount cleanup");
  destroy.forEach(fn => fn());
  assert.equal(owned.size, 0);
});
test("display listener cleanup: Roofline removes its pixel ratio listener", () => {
  const ratio = () => {};
  const listeners = new Set([ratio]);
  const cleanup = hook("generic/plots/Roofline.svelte", "onDestroy", { window: { removeEventListener(type, fn) { assert.equal(type,"dppxchange"); listeners.delete(fn); } }, setPxRatio: ratio, uplot: null, timeoutId: null, clearTimeout() {} });
  cleanup();
  assert.equal(listeners.size, 0, "pixel ratio listener survives unmount");
});
test("historical models: missing current cluster/subcluster definitions remain absent", async () => {
  const { source, body } = script("generic/utils.js");
  const exported = body.find(node => node.type === "ExportNamedDeclaration" && node.declaration?.id?.name === "init");
  assert.ok(exported);
  const config = { name: "load" }, topology = { node: [0] };
  const contexts = new Map();
  const response = { data: { tags: [], globalMetrics: [], clusters: [{ name:"current", subClusters:[{name:"cpu",metricConfig:[config],topology}] }] } };
  class Client { query() { return { toPromise: () => Promise.resolve(response) }; } }
  const init = evaluate(source, exported.declaration, { Client, window:{location:{origin:"http://fixture.invalid"}}, expiringCacheExchange:()=>({}), fetchExchange:()=>{}, setContextClient(){}, setContext:(key,value)=>contexts.set(key,value), readable:()=>({}), tick:()=>Promise.resolve() });
  init();
  await Promise.resolve();
  assert.equal(contexts.get("getMetricConfig")("current","cpu","load"), config);
  assert.equal(contexts.get("getHardwareTopology")("current","cpu"), topology);
  for (const args of [["removed-cluster","cpu"],["current","removed-subcluster"],[null,"cpu"]]) {
    assert.equal(contexts.get("getMetricConfig")(...args,"load"), undefined);
    assert.equal(contexts.get("getHardwareTopology")(...args), undefined);
  }
});
