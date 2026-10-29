// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-backend.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
import assert from "node:assert/strict";

import test from "node:test";

import { businessFunction } from "./helpers/source.mjs";

test("tag search finds an exact existing name independent of case", () => {
    const fuzzyMatch = businessFunction("src/generic/utils.js", "fuzzyMatch");
    const search = businessFunction("src/generic/utils.js", "fuzzySearchTags", {
        fuzzyMatch: fuzzyMatch
    });
    const tags = Object.freeze([ {
        type: "Kind",
        name: "Alpha"
    } ]);
    for (const term of [ "Alpha", "alpha", "ALPHA", " Kind : Alpha ", "kind:alpha" ]) {
        assert.deepEqual(search(term, tags), tags, term);
    }
    assert.deepEqual(search("absent", tags), []);
    assert.deepEqual(search("", tags), tags);
});
