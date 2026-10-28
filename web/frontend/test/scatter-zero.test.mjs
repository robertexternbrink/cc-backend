// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from "node:assert/strict";

import test from "node:test";

import { businessFunction } from "./helpers/source.mjs";

test("scatter maps valid points onto finite coordinates on either zero axis", () => {
    const context = {
        paddingLeft: 40,
        paddingRight: 10,
        paddingTop: 10,
        paddingBottom: 50,
        axesColor: "gray",
        fontSize: 12,
        fontFamily: "sans-serif",
        formatNumber: String
    };
    context.getStepSize = businessFunction("src/generic/plots/Scatter.svelte", "getStepSize");
    const render = businessFunction("src/generic/plots/Scatter.svelte", "render", context);
    for (const [x, y] of [ [ [ 0, 0 ], [ 1, 2 ] ], [ [ 1, 2 ], [ 0, 0 ] ], [ [ 0, 0 ], [ 0, 0 ] ] ]) {
        const arcs = [];
        const ctx = {
            beginPath() {},
            fill() {},
            stroke() {},
            moveTo() {},
            lineTo() {},
            fillText() {},
            save() {},
            restore() {},
            translate() {},
            rotate() {},
            measureText: () => ({
                width: 1
            }),
            arc: (...values) => {
                if (values.slice(0, 5).every(Number.isFinite)) arcs.push(values);
            }
        };
        render(ctx, x, y, null, "blue", "X", "Y", 250, 300);
        assert.equal(arcs.length, 2, JSON.stringify([ x, y ]));
    }
});
