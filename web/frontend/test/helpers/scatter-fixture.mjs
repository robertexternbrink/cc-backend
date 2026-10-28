// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import os from "node:os";

import { frontend } from "./source.mjs";

import fs from "node:fs";

import path from "node:path";

import { createRequire } from "node:module";

const require = createRequire(import.meta.url);

const cache = path.join(frontend, "node_modules");

const {rollup: rollup} = await import(cache + "/rollup/dist/es/rollup.js");

const {nodeResolve: nodeResolve} = await import(cache + "/@rollup/plugin-node-resolve/dist/es/index.js");

const compiler = require(cache + "/svelte/compiler");

const {chromium: chromium} = require(process.env.CC_PLAYWRIGHT_MODULE || "playwright");

const original = path.resolve(frontend, "../..");

export async function run(sourceRoot = original) {
    const output = fs.mkdtempSync(path.join(os.tmpdir(), "cc-scatter-"));
    const harness = `<script>\nimport Scatter from 'assigned-scatter';\nlet X=$state([1,2]); let Y=$state([2,4]); let S=$state([1,2]); let color=$state('#0066cc');let xLabel=$state('initial X');let yLabel=$state('initial Y');\nexport function update(values){if('X'in values)X=values.X;if('Y'in values)Y=values.Y;if('S'in values)S=values.S;if('color'in values)color=values.color;if('xLabel'in values)xLabel=values.xLabel;if('yLabel'in values)yLabel=values.yLabel;}\n<\/script>\n<Scatter {X} {Y} {S} {color} {xLabel} {yLabel} width={250} height={250}/>\n`;
    const entry = `import {mount,unmount} from 'svelte';import Harness from './Harness.svelte';window.fixture=mount(Harness,{target:document.getElementById('fixture')});window.removeFixture=()=>unmount(window.fixture);`;
    fs.writeFileSync(output + "/Harness.svelte", harness);
    fs.writeFileSync(output + "/entry.mjs", entry);
    const warnings = [];
    const build = await rollup({
        input: output + "/entry.mjs",
        plugins: [ {
            name: "assigned-source",
            resolveId(id, importer) {
                if (id === "assigned-scatter") return sourceRoot + "/web/frontend/src/generic/plots/Scatter.svelte";
                if (id === "../units.js" && importer?.endsWith("/Scatter.svelte")) return original + "/web/frontend/src/generic/units.js";
            },
            transform(code, id) {
                if (id.endsWith(".svelte")) {
                    const result = compiler.compile(code, {
                        filename: id,
                        generate: "client",
                        css: "injected"
                    });
                    warnings.push(...result.warnings.map(w => ({
                        code: w.code,
                        message: w.message
                    })));
                    return result.js;
                }
            }
        }, nodeResolve({
            browser: true,
            exportConditions: [ "browser" ],
            modulePaths: [ cache ]
        }) ],
        onwarn(w) {
            warnings.push({
                code: w.code,
                message: w.message
            });
        }
    });
    const generated = await build.generate({
        format: "iife",
        name: "AuditPlotFixture"
    });
    await build.close();
    const script = generated.output[0].code;
    fs.writeFileSync(output + "/bundle.js", script);
    const browser = await chromium.launch({
        headless: true,
        chromiumSandbox: true,
        ...process.env.CC_CHROME ? {
            executablePath: process.env.CC_CHROME
        } : {}
    });
    let page;
    try {
        page = await browser.newPage({
            viewport: {
                width: 640,
                height: 480
            }
        });
        const errors = [];
        page.on("pageerror", e => errors.push(e.message));
        await page.setContent('<!doctype html><html><head><style>#fixture,.cc-plot{width:250px}</style></head><body><div id="fixture"></div></body></html>');
        await page.addScriptTag({
            content: `window.spy={arcs:[],texts:[],colors:[],frames:[]};for(const name of ['arc','fillText','stroke']){const original=CanvasRenderingContext2D.prototype[name];CanvasRenderingContext2D.prototype[name]=function(...args){if(name==='arc'){spy.arcs.push(args.slice(0,3));spy.colors.push(this.fillStyle);}if(name==='fillText')spy.texts.push(args[0]);if(name==='stroke'){spy.frames.push({arcs:spy.arcs,texts:spy.texts,colors:spy.colors,width:this.canvas.width,height:this.canvas.height});spy.arcs=[];spy.texts=[];spy.colors=[];}return original.apply(this,args);};}`
        });
        await page.addScriptTag({
            content: script
        });
        await page.waitForTimeout(450);
        const initial = await page.evaluate("spy.frames.at(-1)");
        await page.evaluate("fixture.update({X:[1,4],Y:[4,1],S:[4,1],xLabel:'updated X',yLabel:'updated Y'})");
        await page.waitForTimeout(350);
        const data = await page.evaluate("spy.frames.at(-1)");
        await page.evaluate("fixture.update({S:[1,4]})");
        await page.waitForTimeout(350);
        const weights = await page.evaluate("spy.frames.at(-1)");
        await page.evaluate("fixture.update({xLabel:'labels only X',yLabel:'labels only Y'})");
        await page.waitForTimeout(350);
        const labels = await page.evaluate("spy.frames.at(-1)");
        await page.evaluate("fixture.update({color:'#ff0000'})");
        await page.waitForTimeout(350);
        const color = await page.evaluate("spy.frames.at(-1)");
        await page.evaluate("fixture.update({X:[2,8],Y:[8,2],S:[1,4]});fixture.update({X:[1,4],Y:[4,1],S:[4,1]})");
        await page.waitForTimeout(350);
        const coalesced = await page.evaluate("spy.frames.at(-1)");
        await page.evaluate("fixture.update({xLabel:'unmounted pending'});removeFixture()");
        await page.waitForTimeout(350);
        const result = {
            sourceExecution: "compiled Svelte component in Chromium DOM",
            sourceRoot: sourceRoot,
            compiler: compiler.VERSION,
            runtime: require(cache + "/svelte/package.json").version,
            rollup: require(cache + "/rollup/package.json").version,
            playwright: require((process.env.CC_PLAYWRIGHT_MODULE || "playwright") + "/package.json").version,
            browser: browser.version(),
            browserExecutable: process.env.CC_CHROME || "playwright Chromium",
            warnings: warnings,
            initial: initial,
            data: data,
            weights: weights,
            labels: labels,
            color: color,
            coalesced: coalesced,
            afterUnmountError: errors.length ? errors.join("; ") : null
        };
        fs.writeFileSync(output + "/observations.json", JSON.stringify(result, null, 2) + "\n");
        return result;
    } finally {
        await browser.close();
        fs.rmSync(output, {
            recursive: true,
            force: true
        });
    }
}
