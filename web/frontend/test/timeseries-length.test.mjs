// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from "node:assert/strict";

import test from "node:test";

import { businessFunction, derived } from "./helpers/source.mjs";

test("node plots retain exactly one timestamp per sample", () => {
    const movingAverage = businessFunction("src/generic/plots/smoothing.js", "movingAverage");
    const context = {
        longestSeries: 3,
        timestep: 60,
        forNode: true,
        sortedMetricData: [ {
            data: [ 1, 2, 3 ]
        } ],
        series: [ {
            data: [ 1, 2, 3 ]
        } ],
        useStatsSeries: false,
        usesMeanStatsSeries: false,
        smoothing: 1,
        movingAverage: movingAverage
    };
    for (const name of [ "DoubleMetricPlot", "MetricPlot" ]) {
        const data = derived(`src/generic/plots/${name}.svelte`, "plotData", context);
        assert.deepEqual(data[0], [ -180, -120, -60 ], name);
        assert.deepEqual(data[1], [ 1, 2, 3 ], name);
    }
});
