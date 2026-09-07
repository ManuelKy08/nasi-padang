#!/bin/bash
cd /data/data/com.termux/files/usr/tmp/opencode/rantau
go build -o rantau-app . || exit 1

# Reset to a deterministic seed state.
mysql -u rantau -prantau_secret_2026 rantau < database/rantau.sql || exit 1

./rantau-app > rantau-run.log 2>&1 &
SRV=$!
trap "kill $SRV 2>/dev/null" EXIT
sleep 2

B=http://127.0.0.1:8080
FAIL=0

echo "=== 1. unauth redirect /menu -> /login ==="
code=$(curl -s -o /dev/null -w '%{http_code}' $B/menu)
redir=$(curl -s -o /dev/null -w '%{redirect_url}' $B/menu)
echo "code=$code redirect=$redir"
[ "$code" = "303" ] || FAIL=1

echo "=== 2. login page renders ==="
curl -s -c cj.txt $B/login -o login.html
grep -q "Selamat datang kembali." login.html && echo OK || { echo "MISSING login text"; FAIL=1; }

echo "=== 3. login wrong password -> flash + 303 ==="
curl -s -b cj.txt -c cj.txt -X POST $B/login --data-urlencode email=customer@rantau.local --data-urlencode password=wrong -o /dev/null -w "%{http_code}" > code.txt
echo "code=$(cat code.txt) (expect 303)"
c1=$(curl -s -b cj.txt -c cj.txt $B/login | grep -c "Email atau password salah")
echo "flash_count=$c1"
[ "$c1" = "1" ] || FAIL=1

echo "=== 4. login correct password ==="
curl -s -b cj.txt -c cj.txt -X POST $B/login --data-urlencode email=customer@rantau.local --data-urlencode password=customer123 -o /dev/null -w "%{http_code}\n"
echo "--- now dashboard ---"
curl -s -b cj.txt -c cj.txt $B/dashboard -o dash.html
grep -q "SALAM RANTAU" dash.html && grep -q "Budi Santoso" dash.html && echo "DASH OK" || { echo "DASH FAIL"; FAIL=1; }

echo "=== 5. menu page ==="
curl -s -b cj.txt -c cj.txt $B/menu -o menu.html
grep -q "Rendang" menu.html && echo "MENU OK" || { echo "MENU FAIL"; FAIL=1; }
grep -q "Rp 25.000" menu.html && echo "PRICE OK" || { echo "PRICE FAIL"; FAIL=1; }

echo "=== 6. menu filter + search ==="
curl -s -b cj.txt -c cj.txt "$B/menu?category=minuman" -o menmu.html
grep -q "Es Teh" menmu.html && echo "CAT OK" || { echo "CAT FAIL"; FAIL=1; }
curl -s -b cj.txt -c cj.txt "$B/menu?q=rendang" -o msearch.html
grep -q "Rendang" msearch.html && echo "SEARCH OK" || { echo "SEARCH FAIL"; FAIL=1; }

echo "=== 7. live search api ==="
CSRF=$(grep -o 'name="csrf" content="[^"]*"' dash.html | head -1 | sed 's/.*content="//;s/"//')
echo "csrf=$CSRF"
JSON=$(curl -s -b cj.txt -c cj.txt "$B/api/menu/search?q=ayam")
echo "$JSON" | grep -q '"ok":true' && echo "API SEARCH OK" || { echo "API FAIL: $JSON"; FAIL=1; }

echo "=== 8. add to cart ==="
RES=$(curl -s -b cj.txt -c cj.txt -X POST $B/cart/add --data-urlencode menu_id=2 --data-urlencode qty=2 -H "X-CSRF-Token: $CSRF")
echo "$RES"
echo "$RES" | grep -q '"ok":true' || FAIL=1
RES2=$(curl -s -b cj.txt -c cj.txt -X POST $B/cart/add --data-urlencode menu_id=3 --data-urlencode qty=1 -H "X-CSRF-Token: $CSRF")
echo "$RES2"

echo "=== 9. cart page ==="
curl -s -b cj.txt -c cj.txt $B/cart -o cart.html
grep -q "Rendang" cart.html && echo "CART RENDER OK" || { echo "CART FAIL"; FAIL=1; }

echo "=== 10. checkout page ==="
curl -s -b cj.txt -c cj.txt $B/checkout -o co.html
grep -q "Selesaikan Pesanan" co.html && echo "CHECKOUT OK" || { echo "CHECKOUT FAIL"; FAIL=1; }

echo "=== 11. place order (delivery) ==="
POSTBODY="order_type=delivery&payment_method=cash&table_number=&pickup_name=Budi&pickup_phone=081234567890&address=Jl+Merdeka+10+Padang&notes=tanpa+pedas"
curl -s -b cj.txt -c cj.txt -X POST $B/checkout --data-urlencode order_type=delivery --data-urlencode payment_method=cash --data-urlencode pickup_name="Budi Santoso" --data-urlencode pickup_phone="081234567890" --data-urlencode address="Jl. Merdeka No 10 Padang" --data-urlencode notes="tanpa pedas" -H "X-CSRF-Token: $CSRF" -o /dev/null -w "%{http_code} -> %{redirect_url}\n"

echo "=== 12. receipt ==="
curl -s -b cj.txt -c cj.txt $B/receipt/RN-20260907-003 -o receipt.html
grep -q "TOTAL" receipt.html && echo "RECEIPT OK" || { echo "RECEIPT FAIL"; FAIL=1; }
grep -q "PRINT RECEIPT" receipt.html && echo "PRINT BTN OK" || true

echo "=== 13. orders history ==="
curl -s -b cj.txt -c cj.txt $B/orders -o orders.html
grep -q "RN-" orders.html && echo "ORDERS OK" || { echo "ORDERS FAIL"; FAIL=1; }

echo "=== 14. order tracking ==="
curl -s -b cj.txt -c cj.txt $B/orders/5 -o track.html
grep -q "Perkembangan" track.html && echo "TRACK OK" || { echo "TRACK FAIL"; FAIL=1; }

echo "=== 15. favorites toggle ==="
curl -s -b cj.txt -c cj.txt -X POST $B/favorites/toggle --data-urlencode menu_id=4 -H "X-CSRF-Token: $CSRF"
echo ""
curl -s -b cj.txt -c cj.txt $B/favorites -o favs.html
grep -q "Ayam Bakar" favs.html && echo "FAV OK" || { echo "FAV FAIL"; FAIL=1; }

echo "=== 16. profile page + update ==="
curl -s -b cj.txt -c cj.txt $B/profile -o prof.html
grep -q "Profil Saya" prof.html && echo "PROFILE OK" || { echo "PROFILE FAIL"; FAIL=1; }
curl -s -b cj.txt -c cj.txt -X POST $B/profile --data-urlencode name="Budi Santoso" --data-urlencode phone="081234567890" --data-urlencode address="Jl Merdeka 10 Padang" -H "X-CSRF-Token: $CSRF" -o /dev/null -w "profile code %{http_code}\n"

echo "=== 17. customer blocked from /admin ==="
code=$(curl -s -b cj.txt -c cj.txt -o /dev/null -w '%{http_code}' $B/admin)
echo "admin code=$code (expect 403)"
[ "$code" = "403" ] || FAIL=1

echo "=== 18. customer blocked from admin api ==="
code2=$(curl -s -b cj.txt -c cj.txt -o /dev/null -w '%{http_code}' $B/admin/reports)
echo "reports code=$code2 (expect 403)"
[ "$code2" = "403" ] || FAIL=1

echo "=== 19. admin login ==="
curl -s -c cj2.txt $B/login -o /dev/null
curl -s -b cj2.txt -c cj2.txt -X POST $B/login --data-urlencode email=admin@rantau.local --data-urlencode password=admin123 -o /dev/null -w "admin login %{http_code}\n"
curl -s -b cj2.txt -c cj2.txt $B/admin -o adm.html
grep -q "Dashboard Restoran" adm.html && grep -q "Total Menu" adm.html && echo "ADMIN DASH OK" || { echo "ADMIN DASH FAIL"; FAIL=1; }

echo "=== 20. admin menu mgmt ==="
curl -s -b cj2.txt -c cj2.txt $B/admin/menu -o admmenu.html
grep -q "Manajemen Menu" admmenu.html && grep -q "Tambah Menu" admmenu.html && echo "ADMIN MENU OK" || { echo "ADMIN MENU FAIL"; FAIL=1; }
CSRF2=$(grep -o 'name="csrf" content="[^"]*"' adm.html | head -1 | sed 's/.*content="//;s/"//')
echo "$CSRF2" > csrf2.txt
CODE=$(curl -s -b cj2.txt -c cj2.txt -X POST $B/admin/menu/create -H "X-CSRF-Token: $CSRF2" --data-urlencode category_id=2 --data-urlencode name="Test Coba Masak" --data-urlencode description="Menu hasil uji coba" --data-urlencode price=15000 -o /dev/null -w '%{http_code}')
echo "create menu code=$CODE (expect 303)"
curl -s -b cj2.txt -c cj2.txt "$B/admin/menu?q=x" -o /dev/null
curl -s -b cj2.txt -c cj2.txt $B/admin/menu -o admmenu2.html
grep -q "Test Coba Masak" admmenu2.html && echo "MENU CREATE OK" || { echo "MENU CREATE FAIL"; FAIL=1; }

echo "=== 21. admin toggle availability ==="
curl -s -b cj2.txt -c cj2.txt -X POST $B/admin/menu/toggle -H "X-CSRF-Token: $CSRF2" --data-urlencode id=28
echo ""

echo "=== 22. admin orders ==="
curl -s -b cj2.txt -c cj2.txt "$B/admin/orders?status=all" -o admorders.html
grep -q "Semua Pesanan" admorders.html && echo "ADMIN ORDERS OK" || { echo "ADMIN ORDERS FAIL"; FAIL=1; }

echo "=== 23. admin status update ==="
curl -s -b cj2.txt -c cj2.txt -X POST $B/admin/orders/5/status -H "X-CSRF-Token: $CSRF2" --data-urlencode status=cooking
echo ""

echo "=== 24. admin users ==="
curl -s -b cj2.txt -c cj2.txt $B/admin/users -o admusers.html
grep -q "Manajemen Pengguna" admusers.html && echo "ADMIN USERS OK" || { echo "ADMIN USERS FAIL"; FAIL=1; }

echo "=== 25. admin reports ==="
curl -s -b cj2.txt -c cj2.txt $B/admin/reports -o admrep.html
grep -q "Laporan Penjualan" admrep.html && grep -q "Penjualan Hari Ini" admrep.html && echo "ADMIN REPORTS OK" || { echo "ADMIN REPORTS FAIL"; FAIL=1; }

echo "=== 26. register flow ==="
curl -s -c cj3.txt $B/register -o reg.html
grep -q "Buat akun baru" reg.html && echo "REG PAGE OK" || { echo "REG PAGE FAIL"; FAIL=1; }
CSRF3=$(grep -o 'name="csrf" content="[^"]*"' reg.html | head -1 | sed 's/.*content="//;s/"//')
curl -s -b cj3.txt -c cj3.txt -X POST $B/register -H "X-CSRF-Token: $CSRF3" --data-urlencode name="Orang Baru" --data-urlencode email=baru@rantau.local --data-urlencode phone=087812345678 --data-urlencode password=password123 --data-urlencode password_confirm=password123 -o /dev/null -w "register code %{http_code}\n"
curl -s -b cj3.txt -c cj3.txt $B/login -o login2.html
grep -q "Akun berhasil dibuat" login2.html && echo "REG FLASH OK" || { echo "REG FLASH FAIL"; FAIL=1; }

echo "=== 27. logout ==="
curl -s -b cj.txt -c cj.txt -X POST $B/logout -H "X-CSRF-Token: $CSRF" -o /dev/null -w "logout %{http_code} -> %{redirect_url}\n"
code=$(curl -s -b cj.txt -o /dev/null -w '%{http_code}' $B/dashboard)
echo "dashboard after logout code=$code (expect 303)"
[ "$code" = "303" ] || FAIL=1

echo ""
if [ "$FAIL" = "0" ]; then echo "ALL TESTS PASSED"; else echo "SOME TESTS FAILED"; fi
