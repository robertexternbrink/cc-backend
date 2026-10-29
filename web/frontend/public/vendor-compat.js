// Copyright (c) 2026. ClusterCockpit vendor CSS compatibility.
(() => {
  if (typeof CSS === 'undefined' || typeof CSS.supports !== 'function') return;
  const rules = [];
  if (CSS.supports('-webkit-text-size-adjust', '100%')) {
    rules.push('body{-webkit-text-size-adjust:100%}');
  }
  if (CSS.supports('-moz-osx-font-smoothing', 'grayscale')) {
    rules.push(".bi::before,\n[class^=\"bi-\"]::before,\n[class*=\" bi-\"]::before{-moz-osx-font-smoothing:grayscale}");
  }
  if (CSS.supports('selector(.form-range::-webkit-slider-thumb:active)')) {
    rules.push(".form-range::-webkit-slider-thumb:active{background-color:#b6d4fe}");
  }
  if (rules.length === 0) return;
  const style = document.createElement('style');
  style.textContent = rules.join('\n');
  document.head.appendChild(style);
})();
