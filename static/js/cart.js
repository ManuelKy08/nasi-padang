/* RANTAU — cart drawer, quick add, quantity updates and review UI. */
(function () {
  'use strict';

  var Rantau = window.Rantau || {};

  var drawer = document.getElementById('cartDrawer');
  var scrim = document.getElementById('cartScrim');
  var openBtns = Array.prototype.slice.call(document.querySelectorAll('#cartOpen, #mnCartOpen'));
  var closeBtn = document.getElementById('cartClose');
  var badge = document.getElementById('cartBadge');
  var bodyEl = document.getElementById('cartDrawerBody');
  var footEl = document.getElementById('cartDrawerFoot');

  function openDrawer() {
    if (!drawer) return;
    drawer.classList.add('open');
    drawer.setAttribute('aria-hidden', 'false');
    if (scrim) scrim.hidden = false;
    document.body.style.overflow = 'hidden';
    loadCart();
  }
  function closeDrawer() {
    if (!drawer) return;
    drawer.classList.remove('open');
    drawer.setAttribute('aria-hidden', 'true');
    if (scrim) scrim.hidden = true;
    document.body.style.overflow = '';
  }

  openBtns.forEach(function (b) {
    if (b) b.addEventListener('click', function (e) { e.preventDefault(); openDrawer(); });
  });
  if (closeBtn) closeBtn.addEventListener('click', closeDrawer);
  if (scrim) scrim.addEventListener('click', closeDrawer);
  document.addEventListener('keydown', function (e) { if (e.key === 'Escape') closeDrawer(); });

  function setBadge(n) {
    if (!badge) return;
    if (n > 0) { badge.textContent = n; badge.hidden = false; }
    else badge.hidden = true;
  }

  function loadCart() {
    fetch('/cart/data', { credentials: 'same-origin' })
      .then(function (r) { return r.json(); })
      .then(function (res) {
        if (!res.ok || !res.data) return;
        render(res.data);
      }).catch(function () {});
  }

  function render(d) {
    if (!bodyEl) return;
    var items = d.items || [];
    setBadge(d.count || 0);
    if (!items.length) {
      bodyEl.innerHTML =
        '<div class="cart-empty">' +
        '<p>Belum ada pesanan.</p><p class="muted">Yuk pilih menu favoritmu dulu.</p>' +
        '<a href="/menu" class="btn btn-primary btn-sm">Lihat Menu</a></div>';
      if (footEl) footEl.hidden = true;
      return;
    }
    if (footEl) footEl.hidden = false;
    var totalEl = document.getElementById('cartDrawerTotal');
    if (totalEl) totalEl.textContent = Rantau.rupiah(d.subtotal || 0);
    Rantau.currentSubtotal = d.subtotal || 0;

    bodyEl.innerHTML =
      '<ul class="drawer-items">' + items.map(function (c) {
        var line = c.Menu ? c.Menu : c.menu;
        var q = c.Quantity || c.quantity;
        var price = line.Price || line.price;
        var img = line.Image || line.image || '/static/images/menu/placeholder.svg';
        var name = line.Name || line.name;
        var id = c.ID || c.id;
        return '<li class="drawer-item" data-line="' + id + '">' +
          '<img src="' + img + '" alt="" onerror="this.src=\'/static/images/menu/placeholder.svg\'">' +
          '<div class="di-info">' +
            '<strong>' + name + '</strong>' +
            '<span>' + Rantau.rupiah(price) + '</span>' +
            '<div class="di-line">' +
              '<div class="qty-stepper">' +
                '<button type="button" class="qty-btn" data-dec="' + id + '" aria-label="Kurangi">−</button>' +
                '<input type="number" min="1" max="50" value="' + q + '" data-line-qty="' + id + '" aria-label="Jumlah">' +
                '<button type="button" class="qty-btn" data-inc="' + id + '" aria-label="Tambah">+</button>' +
              '</div>' +
              '<button type="button" class="di-remove" data-remove="' + id + '" aria-label="Hapus">✕</button>' +
            '</div>' +
          '</div>' +
        '</li>';
      }).join('') + '</ul>';
  }

  /* ---------- Add to cart (from menu cards / detail) ---------- */
  document.body.addEventListener('click', function (e) {
    var qtyInput = document.getElementById('qtyInput');
    var btn = e.target.closest('[data-add]');
    if (btn) {
      var qty = 1;
      if (qtyInput && qtyInput.value) qty = parseInt(qtyInput.value, 10) || 1;
      var id = btn.getAttribute('data-add');
      var name = btn.getAttribute('data-name') || 'Menu';
      var note = document.getElementById('noteInput');
      Rantau.post('/cart/add', { menu_id: id, qty: String(qty) })
        .then(function (res) {
          if (!res.ok) { Rantau.notify(res.error || 'Menu sedang tidak tersedia.', 'error'); return; }
          Rantau.notify('✓ ' + res.data.message, 'success');
          setBadge(res.data.count);
          Rantau.currentSubtotal = (Rantau.currentSubtotal || 0) + 0;
          openDrawer();
        }).catch(function (err) { Rantau.notify(err.message || 'Gagal menambahkan.', 'error'); });
      return;
    }

    var dec = e.target.closest('[data-dec]');
    var inc = e.target.closest('[data-inc]');
    if (dec || inc) {
      var lineId = (dec || inc).getAttribute('data-' + (dec ? 'dec' : 'inc'));
      changeQty(lineId, dec ? -1 : 1);
      return;
    }
    if (e.target.closest('[data-remove]')) {
      var rmId = e.target.closest('[data-remove]').getAttribute('data-remove');
      Rantau.post('/cart/remove', { item_id: rmId })
        .then(function (res) {
          if (res.ok) { Rantau.notify(res.data.message, 'success'); loadCart(); }
          else Rantau.notify(res.error, 'error');
        }).catch(function (err) { Rantau.notify(err.message, 'error'); });
    }
  });

  document.body.addEventListener('change', function (e) {
    var inQty = e.target.closest('[data-line-qty]');
    if (inQty) {
      var v = parseInt(inQty.value, 10) || 1;
      if (v < 1) v = 1;
      if (v > 50) v = 50;
      inQty.value = v;
      Rantau.post('/cart/update', { item_id: inQty.getAttribute('data-line-qty'), qty: String(v) })
        .then(function (res) {
          if (res.ok) { Rantau.currentSubtotal = res.data.subtotal; loadCart(); }
        }).catch(function (err) { Rantau.notify(err.message, 'error'); });
      return;
    }
    // Full cart page steppers
    var pageQty = e.target.closest('[data-qty]');
    if (pageQty) {
      var v2 = parseInt(pageQty.value, 10) || 1;
      if (v2 < 1) v2 = 1;
      if (v2 > 50) v2 = 50;
      pageQty.value = v2;
      updatePageLine(pageQty.getAttribute('data-item'), v2);
    }
  });

  document.body.addEventListener('click', function (e) {
    var pdec = e.target.closest('[data-qty-dec]');
    var pinc = e.target.closest('[data-qty-inc]');
    if (pdec) { var li = pdec.closest('.cart-item'); var i = li ? li.querySelector('[data-qty]') : null; if (i) { var v = Math.max(1, (parseInt(i.value, 10) || 1) - 1); i.value = v; updatePageLine(i.getAttribute('data-item'), v); } return; }
    if (pinc) { var l2 = pinc.closest('.cart-item'); var i2 = l2 ? l2.querySelector('[data-qty]') : null; if (i2) { var v2 = Math.min(50, (parseInt(i2.value, 10) || 1) + 1); i2.value = v2; updatePageLine(i2.getAttribute('data-item'), v2); } return; }
    if (e.target.closest('[data-cart-remove]')) {
      var id = e.target.closest('[data-cart-remove]').getAttribute('data-cart-remove');
      Rantau.post('/cart/remove', { item_id: id })
        .then(function (res) {
          if (res.ok) { Rantau.notify(res.data.message, 'success'); window.location.reload(); }
        }).catch(function (err) { Rantau.notify(err.message, 'error'); });
    }
  });

  function changeQty(lineId, delta) {
    var input = document.querySelector('[data-line-qty="' + lineId + '"]');
    var v = input ? (parseInt(input.value, 10) || 1) : 1;
    v = v + delta;
    if (v < 1) v = 1;
    if (v > 50) v = 50;
    Rantau.post('/cart/update', { item_id: lineId, qty: String(v) })
      .then(function (res) {
        if (res.ok) { Rantau.currentSubtotal = res.data.subtotal; setBadge(res.data.count); loadCart(); }
        else Rantau.notify(res.error, 'error');
      }).catch(function (err) { Rantau.notify(err.message, 'error'); });
  }

  function updatePageLine(itemID, qty) {
    Rantau.post('/cart/update', { item_id: itemID, qty: String(qty) })
      .then(function (res) {
        if (res.ok) {
          var s = document.getElementById('sumSubtotal');
          var t = document.getElementById('sumTotal');
          if (s) s.textContent = Rantau.rupiah(res.data.subtotal);
          if (t) t.textContent = Rantau.rupiah(res.data.subtotal);
        }
      }).catch(function (err) { Rantau.notify(err.message, 'error'); });
  }

  /* ---------- Favorites toggle ---------- */
  document.body.addEventListener('click', function (e) {
    var fav = e.target.closest('[data-fav]');
    if (fav) {
      var id = fav.getAttribute('data-fav');
      Rantau.post('/favorites/toggle', { menu_id: id })
        .then(function (res) {
          if (!res.ok) { Rantau.notify(res.error, 'error'); return; }
          var on = res.data.favorite;
          fav.classList.toggle('is-fav', on);
          fav.querySelector('svg').setAttribute('fill', on ? 'currentColor' : 'none');
          Rantau.notify(on ? '✓ Ditambahkan ke favorit.' : 'Dihapus dari favorit.', on ? 'success' : '');
        }).catch(function (err) { Rantau.notify(err.message, 'error'); });
    }
  });

  /* ---------- Print receipt ---------- */
  var printBtn = document.getElementById('printReceipt');
  if (printBtn) printBtn.addEventListener('click', function () { window.print(); });

  // Expose
  Rantau.cart = { load: loadCart, open: openDrawer, close: closeDrawer };
  window.Rantau = Rantau;
})();