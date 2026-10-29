// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from "node:assert/strict";

import test from "node:test";

import { run } from "./helpers/scatter-fixture.mjs";

test("scatter redraws data, weights, labels and color without resizing", async () => {
    const result = await run();
    assert.equal(result.afterUnmountError, null);
    assert.deepEqual(result.initial.arcs, [ [ 140, 105, 200 / 30 ], [ 240, 10, 200 / 15 ] ]);
    assert.deepEqual(result.data.arcs, [ [ 90, 10, 200 / 15 ], [ 240, 153, 200 / 60 ] ]);
    assert.equal(result.initial.width, result.data.width);
    assert.equal(result.initial.height, result.data.height);
    assert.ok(result.data.texts.includes("updated X"));
    assert.deepEqual(result.weights.arcs, [ [ 90, 10, 200 / 60 ], [ 240, 153, 200 / 15 ] ]);
    assert.ok(result.labels.texts.includes("labels only X"));
    assert.equal(result.color.colors[0], "#ff0000");
});
