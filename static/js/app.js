/* RANTAU — global UI behaviour: nav, toasts, reveal, charts, wizards. */
(function () {
  'use strict';

  var Rantau = window.Rantau || {};

  if (!window.Rantau) window.Rantau = {};

  /* ---------- CSRF ---------- */
  function csrfToken() {
    var m = document.querySelector('meta[name="csrf"]');
    return m ? m.getAttribute('content') : '';
  }

  function post(url, data) {
    return fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
        'X-CSRF-Token': csrfToken()
      },
      credentials: 'same-origin',
      body: new URLSearchParams(data).toString()
    }).then(function (res) {
      if (res.status === 403) throw new Error('Session tidak valid, silakan muat ulang.');
      return res.json();
    });
  }

  /* ---------- Toast notifications ---------- */
  function notify(msg, type) {
    var wrap = document.getElementById('toastWrap');
    if (!wrap) return;
    var t = document.createElement('div');
    t.className = 'toast ' + (type === 'success' ? 'ok' : (type === 'error' ? 'err' : ''));
    t.innerHTML = '<span class="toast-icon">' +
      (type === 'success' ? '✓' : type === 'error' ? '⚠' : 'ℹ') +
      '</span><span></span>';
    t.lastElementChild.textContent = msg;
    wrap.appendChild(t);
    setTimeout(function () {
      t.classList.add('out');
      setTimeout(function () { t.remove(); }, 320);
    }, 2800);
  }

  /* ---------- Sticky navbar focus ---------- */
  var nav = document.getElementById('navbar');
  function onScroll() {
    if (!nav) return;
    if (window.scrollY > 8) nav.classList.add('scrolled');
    else nav.classList.remove('scrolled');
  }
  window.addEventListener('scroll', onScroll, { passive: true });
  onScroll();

  /* ---------- Account dropdown ---------- */
  var acctBtn = document.getElementById('acctToggle');
  var acctMenu = document.getElementById('acctMenu');
  if (acctBtn && acctMenu) {
    acctBtn.addEventListener('click', function (e) {
      e.stopPropagation();
      acctMenu.classList.toggle('open');
    });
    document.addEventListener('click', function (e) {
      if (!acctMenu.contains(e.target)) acctMenu.classList.remove('open');
    });
  }

  /* ---------- Search overlay + live suggestions ---------- */
  var srchTog = document.getElementById('searchToggle');
  var srchOv = document.getElementById('searchOverlay');
  var srchIn = document.getElementById('searchInput');
  var srchSugg = document.getElementById('searchSuggest');
  if (srchTog && srchOv) {
    srchTog.addEventListener('click', function () {
      srchOv.hidden = !srchOv.hidden;
      if (!srchOv.hidden && srchIn) {
        srchIn.focus();
        if (srchIn.value) suggest(srchIn.value);
      }
    });
  }
  if (srchIn && srchSugg) {
    var debounce;
    srchIn.addEventListener('input', function () {
      clearTimeout(debounce);
      debounce = setTimeout(function () { suggest(srchIn.value); }, 220);
    });
  }
  function suggest(q) {
    if (!q.trim()) { srchSugg.innerHTML = ''; return; }
    fetch('/api/menu/search?q=' + encodeURIComponent(q), { credentials: 'same-origin' })
      .then(function (r) { return r.json(); })
      .then(function (res) {
        if (!res.ok || !res.data) return;
        srchSugg.innerHTML = res.data.map(function (m) {
          return '<a class="suggest-item" href="/menu/' + m.id + '">' +
            '<img src="' + m.image + '" alt="" onerror="this.src=\'/static/images/menu/placeholder.svg\'">' +
            '<span class="sj-name">' + m.name + '</span>' +
            '<span class="sj-price">' + (window.Rantau.rupiah ? Rantau.rupiah(m.price) : m.price) + '</span>' +
            '</a>';
        }).join('');
      }).catch(function () {});
  }
  document.addEventListener('click', function (e) {
    if (srchOv && !srchOv.hidden && !srchOv.contains(e.target) && e.target !== srchTog) {
      srchOv.hidden = true;
    }
  });

  /* ---------- Reveal on scroll ---------- */
  var revObs;
  if ('IntersectionObserver' in window) {
    revObs = new IntersectionObserver(function (entries) {
      entries.forEach(function (en) {
        if (en.isIntersecting) { en.target.classList.add('in'); revObs.unobserve(en.target); }
      });
    }, { threshold: 0.12 });
    document.querySelectorAll('.reveal').forEach(function (el) { revObs.observe(el); });
  } else {
    document.querySelectorAll('.reveal').forEach(function (el) { el.classList.add('in'); });
  }

  /* ---------- Guarded element helper ---------- */
  function on(el, evt, fn) { if (el) el.addEventListener(evt, fn); }

  /* ---------- Sort select on menu page ---------- */
  var sortSel = document.getElementById('sortSelect');
  on(sortSel, 'change', function () {
    var url = '/menu?sort=' + encodeURIComponent(sortSel.value);
    var q = new URLSearchParams(window.location.search).get('q');
    var c = new URLSearchParams(window.location.search).get('category');
    if (q) url += '&q=' + encodeURIComponent(q);
    if (c) url += '&category=' + encodeURIComponent(c);
    window.location.href = url;
  });

  /* ---------- Confirm dialogs on destructive forms ---------- */
  document.body.addEventListener('submit', function (e) {
    var f = e.target.closest('form[data-confirm]');
    if (!f) return;
    var msg = f.getAttribute('data-confirm') || 'Yakin?';
    if (!window.confirm(msg)) e.preventDefault();
  });

  /* ---------- Checkout wizard ---------- */
  var checkoutForm = document.getElementById('checkoutForm');
  var orderTypeEl = document.getElementById('orderType');
  var paymentEl = document.getElementById('paymentMethod');
  if (checkoutForm) {
    var fee = parseInt(checkoutForm.getAttribute('data-fee') || '0', 10);
    var pans = Array.prototype.slice.call(checkoutForm.querySelectorAll('.checkout-step'));
    var stepsEls = Array.prototype.slice.call(document.querySelectorAll('.steps li'));
    var subs = checkoutForm.querySelectorAll('input[name="order_type_radio"]');
    var pays = checkoutForm.querySelectorAll('input[name="pay_radio"]');

    function subtotal() { return Rantau.currentSubtotal || 0; }
    function show(stepNum) {
      pans.forEach(function (p) { p.classList.toggle('active', p.getAttribute('data-pane') === stepNum); });
      stepsEls.forEach(function (s) { s.classList.toggle('active', s.getAttribute('data-step') === stepNum); });
      checkoutForm.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
    function currentType() { return orderTypeEl.value; }
    function updateInfoFields() {
      checkoutForm.querySelectorAll('[data-show]').forEach(function (el) {
        var showFor = (el.getAttribute('data-show') || '').split(',');
        el.style.display = showFor.indexOf(currentType()) >= 0 ? '' : 'none';
      });
    }

    checkoutForm.addEventListener('click', function (e) {
      var n = e.target.closest('[data-next]');
      var p = e.target.closest('[data-prev]');
      if (n) { e.preventDefault(); refreshTotals(); show(n.getAttribute('data-next')); }
      if (p) { e.preventDefault(); show(p.getAttribute('data-prev')); }
    });

    subs.forEach(function (r) {
      r.addEventListener('change', function () {
        orderTypeEl.value = r.value;
        setText('confirmType', typeLabel(r.value));
        updateInfoFields();
        refreshTotals();
      });
    });
    pays.forEach(function (r) {
      r.addEventListener('change', function () {
        paymentEl.value = r.value;
        setText('confirmPay', payLabel(r.value));
      });
    });

    var nameIn = checkoutForm.querySelector('#pickupName');
    var phoneIn = checkoutForm.querySelector('#pickupPhone');
    var addrIn = checkoutForm.querySelector('#address');
    var tableIn = checkoutForm.querySelector('#table');
    var noteIn = checkoutForm.querySelector('#notes');
    if (nameIn) nameIn.addEventListener('input', function () { setText('confirmInfo', nameIn.value); });
    if (phoneIn) phoneIn.addEventListener('input', function () { setText('confirmInfo', nameIn.value + ' · ' + phoneIn.value); });
    if (addrIn) addrIn.addEventListener('input', function () { setText('confirmInfo', addrIn.value); });
    if (noteIn) noteIn.addEventListener('input', function () { setText('confirmNote', noteIn.value || '—'); });
    if (tableIn) tableIn.addEventListener('input', function () { setText('confirmInfo', 'Meja ' + tableIn.value); });

    checkoutForm.addEventListener('submit', function (e) {
      var type = currentType();
      var needPhone = checkoutForm.querySelector('#pickupPhone');
      var needName = checkoutForm.querySelector('#pickupName');
      if ((type === 'take_away' || type === 'delivery') &&
          (!needName.value.trim() || (needPhone && !/^\+?[0-9]{9,15}$/.test(needPhone.value.trim())))) {
        e.preventDefault();
        Rantau.notify('Lengkapi nama pemesan dan nomor HP yang valid.', 'error');
        show('02');
        return;
      }
      if (type === 'dine_in' && tableIn && !tableIn.value.trim()) {
        e.preventDefault();
        Rantau.notify('Nomor meja wajib diisi.', 'error');
        show('02');
        return;
      }
      if (type === 'delivery' && (!addrIn.value.trim())) {
        e.preventDefault();
        Rantau.notify('Alamat lengkap wajib diisi untuk delivery.', 'error');
        show('02');
        return;
      }
    });

    function refreshTotals() {
      var type = currentType();
      var _fee = type === 'delivery' ? fee : 0;
      var ttl = subtotal() + _fee;
      setText('reviewFeeDesc', _fee ? Rantau.rupiah(_fee) : 'Gratis');
      setText('reviewTotal', Rantau.rupiah(ttl));
      setText('confirmFee', Rantau.rupiah(_fee));
      setText('confirmTotal', Rantau.rupiah(ttl));
      setText('confirmSubtotal', Rantau.rupiah(subtotal()));
    }
    updateInfoFields();
    refreshTotals();
  }

  function typeLabel(t) {
    return { dine_in: 'Makan di Tempat', take_away: 'Take Away', delivery: 'Delivery' }[t] || t;
  }
  function payLabel(m) {
    return { cash: 'Cash', bank_transfer: 'Bank Transfer', ewallet: 'E-Wallet' }[m] || m;
  }
  function setText(id, txt) {
    var el = document.getElementById(id);
    if (el) el.textContent = txt;
  }

  /* ---------- Currency helper for JS ---------- */
  function rupiah(n) {
    var neg = n < 0; if (neg) n = -n;
    var s = String(n).split('.').join('');
    var out = s.replace(/\B(?=(\d{3})+(?!\d))/g, '.');
    return 'Rp ' + (neg ? '-' : '') + out;
  }

  /* ---------- Bar chart ---------- */
  function barChart(id, data) {
    var el = document.getElementById(id);
    if (!el) return;
    if (!data || !data.length) {
      el.innerHTML = '<div class="muted" style="text-align:center;width:100%;align-self:center">Belum ada data.</div>';
      return;
    }
    var max = Math.max.apply(null, data.map(function (d) { return d.Revenue || 0; })) || 1;
    el.innerHTML = data.map(function (d) {
      var h = Math.max(Math.round(((d.Revenue || 0) / max) * 100), 3);
      var shortDay = String(d.Day || '').slice(5);
      return '<div class="bar-group">' +
        '<span class="bar-val">' + rupiah(d.Revenue || 0).replace('Rp ', 'Rp\u00a0') + '</span>' +
        '<div class="bar ' + (d.Revenue ? '' : 'empty') + '" style="height:' + h + '%"></div>' +
        '<span class="bar-label">' + shortDay + '</span></div>';
    }).join('');
  }

  /* ---------- Admin: availability switch ---------- */
  document.body.addEventListener('click', function (e) {
    var sw = e.target.closest('[data-toggle]');
    if (sw) {
      post('/admin/menu/toggle', { id: sw.getAttribute('data-toggle') })
        .then(function (res) {
          if (!res.ok) throw new Error(res.error || 'Gagal');
          sw.classList.toggle('on', res.data.available);
          Rantau.notify(res.data.message, 'success');
        }).catch(function (err) { Rantau.notify(err.message || 'Gagal memperbarui status.', 'error'); });
      return;
    }
  });

  /* ---------- Admin: order status change ---------- */
  document.body.addEventListener('change', function (e) {
    var sel = e.target.closest('.status-select');
    if (sel) {
      var id = sel.getAttribute('data-status');
      var no = sel.getAttribute('data-order');
      post('/admin/orders/' + id + '/status', { status: sel.value })
        .then(function (res) {
          if (res.ok) Rantau.notify(res.data.message, 'success');
          else Rantau.notify(res.error, 'error');
        }).catch(function (err) { Rantau.notify(err.message, 'error'); });
      return;
    }
    var role = e.target.closest('.role-select');
    if (role) {
      post('/admin/users/' + role.getAttribute('data-user') + '/role', { role: role.value })
        .then(function (res) {
          if (res.ok) Rantau.notify(res.data.message, 'success');
          else Rantau.notify(res.error, 'error');
        }).catch(function (err) { Rantau.notify(err.message, 'error'); });
    }
  });

  // Public API
  Rantau.post = post;
  Rantau.notify = notify;
  Rantau.rupiah = rupiah;
  Rantau.barChart = barChart;
  window.Rantau = Rantau;
})();