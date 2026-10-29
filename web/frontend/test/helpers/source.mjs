// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import fs from "node:fs";

import os from "node:os";

import path from "node:path";

import { fileURLToPath } from "node:url";

import { compile, parse } from "svelte/compiler";

import { render } from "svelte/server";

export const frontend = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

function parsed(relative) {
    let source = fs.readFileSync(path.join(frontend, relative), "utf8");
    if (relative.endsWith(".js")) source = "<script>" + source + "<\/script>";
    return {
        source: source,
        ast: parse(source)
    };
}

function declared(relative, name) {
    const {source: source, ast: ast} = parsed(relative);
    for (const entry of ast.instance.content.body) {
        const node = entry.type === "ExportNamedDeclaration" ? entry.declaration : entry;
        if (node?.type === "FunctionDeclaration" && node.id.name === name) return {
            source: source,
            node: node
        };
        if (node?.type === "VariableDeclaration") {
            const node = (entry.declaration || entry).declarations.find(d => d.id.name === name);
            if (node) return {
                source: source,
                node: node.init
            };
        }
    }
    throw new Error(`No declaration ${name} in ${relative}`);
}

function evaluate(source, node, context) {
    return Function(...Object.keys(context), `return (${source.slice(node.start, node.end)});`)(...Object.values(context));
}

// Execute the actual declared business function, without bootstrapping a page.
export function businessFunction(relative, name, context = {}) {
    const {source: source, node: node} = declared(relative, name);
    return evaluate(source, node, context);
}

export function declarationSource(relative, name) {
    const {source: source, node: node} = declared(relative, name);
    return source.slice(node.start, node.end);
}

export function derived(relative, name, context = {}) {
    const {source: source, node: node} = declared(relative, name);
    const value = evaluate(source, node.arguments[0], context);
    return node.callee.type === "MemberExpression" ? value() : value;
}

export function templateExpressions(relative, containing, context) {
    const {source: source, ast: ast} = parsed(relative);
    const values = [];
    function visit(node) {
        if (!node || typeof node !== "object") return;
        if (node.type === "MustacheTag" && source.slice(node.expression.start, node.expression.end).includes(containing)) {
            values.push(evaluate(source, node.expression, context));
        }
        for (const value of Object.values(node)) {
            if (Array.isArray(value)) value.forEach(visit); else if (value && typeof value === "object") visit(value);
        }
    }
    visit(ast.html);
    return values;
}

// Render the actual header markup with Svelte's server renderer.
export async function renderHeader(relative, containing, props) {
    const {source: source, ast: ast} = parsed(relative);
    let element;
    function visit(node) {
        if (!node || typeof node !== "object") return;
        if (node.type === "Element" && node.name === "h5" && source.slice(node.start, node.end).includes(containing)) element = node;
        for (const value of Object.values(node)) {
            if (Array.isArray(value)) value.forEach(visit); else if (value && typeof value === "object") visit(value);
        }
    }
    visit(ast.html);
    if (!element) throw new Error("Header fixture not found");
    const markup = "<script>let {sortedClusterMetrics}=$props();<\/script>" + source.slice(element.start, element.end);
    let code = compile(markup, {
        generate: "server"
    }).js.code;
    code = code.replace(/from ['"](svelte[^'"]+)['"]/g, (_, name) => `from ${JSON.stringify(import.meta.resolve(name))}`);
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "cc-header-"));
    try {
        const file = path.join(dir, "Header.mjs");
        fs.writeFileSync(file, code);
        const {default: Header} = await import(file);
        return render(Header, {
            props: props
        }).body.replace(/<[^>]*>/g, "").replace(/\s+/g, " ").trim();
    } finally {
        fs.rmSync(dir, {
            recursive: true,
            force: true
        });
    }
}
