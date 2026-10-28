# Frontend regression tests

Run `npm ci`, then `npx playwright install chromium` once and `npm test` from
`web/frontend`. Set `CC_CHROME` to an existing Chromium/Chrome executable to use
that browser instead. Playwright is an exact development-only dependency.

The dashboard fixtures compile the actual current components with Svelte and
render their Chart.js canvases in a browser. Query stores, refresh callbacks and
presentation wrappers are controlled doubles. They use synthetic data and
`page.setContent`; no backend server, account, cluster, or external endpoint is
required. Temporary generated components are removed after each fixture.

Small business-function tests parse the real declarations with the Svelte
compiler and execute only those functions with explicit input contexts. Header
tests render the actual markup with Svelte's server renderer. These checks do
not assert source text patterns or claim backend integration coverage.
