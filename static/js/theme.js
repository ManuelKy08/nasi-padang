/* RANTAU — theme switching (light/dark) with localStorage persistence. */
(function () {
  'use strict';

  function apply(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    try { localStorage.setItem('rantau-theme', theme); } catch (e) {}
    var mt = document.querySelector('meta[name="theme-color"]');
    if (mt) mt.setAttribute('content', theme === 'dark' ? '#140d0b' : '#7a1b16');
  }

  function current() {
    return document.documentElement.getAttribute('data-theme') || 'light';
  }

  var btn = document.getElementById('themeToggle');
  if (btn) {
    btn.addEventListener('click', function () {
      apply(current() === 'dark' ? 'light' : 'dark');
    });
  }

  window.RantauTheme = { apply: apply, current: current };
})();