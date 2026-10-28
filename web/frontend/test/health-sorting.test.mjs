// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from "node:assert/strict";

import test from "node:test";

import { runHealthDashboardFixture } from "./helpers/health-fixture.mjs";

test("health table keeps the selected ordering on a new response", async () => {
    const result = await runHealthDashboardFixture("normal");
    assert.deepEqual(result.errors, []);
    assert.deepEqual(result.selected, [ "alpha", "beta", "zeta" ]);
    assert.deepEqual(result.refreshed, result.selected);
    assert.deepEqual(result.descending, [ "zeta", "beta", "alpha" ]);
    assert.deepEqual(result.filtered, [ "alpha" ]);
    assert.deepEqual(result.healthFiltered, [ "zeta" ]);
});
