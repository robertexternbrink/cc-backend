// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from "node:assert/strict";

import test from "node:test";

import { runHealthDashboardFixture } from "./helpers/health-fixture.mjs";

test("health legends use the rendered chart palette in both color modes", async () => {
    for (const mode of [ "normal", "cb" ]) {
        const result = await runHealthDashboardFixture(mode);
        assert.deepEqual(result.errors, []);
        assert.deepEqual(result.colorComparison.charts, result.colorComparison.legends);
    }
});
