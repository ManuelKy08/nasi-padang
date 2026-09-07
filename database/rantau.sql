-- ============================================================
-- RANTAU — Rumah Makan Padang
-- Database schema & seed data
-- MariaDB / MySQL 8+
-- ============================================================

CREATE DATABASE IF NOT EXISTS rantau
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE rantau;

SET FOREIGN_KEY_CHECKS = 0;
DROP TABLE IF EXISTS favorites;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS carts;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS menu_items;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS users;
SET FOREIGN_KEY_CHECKS = 1;

-- ------------------------------------------------------------
-- USERS
-- ------------------------------------------------------------
CREATE TABLE users (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name       VARCHAR(120)    NOT NULL,
  email      VARCHAR(190)    NOT NULL,
  phone      VARCHAR(30)     NOT NULL DEFAULT '',
  password   VARCHAR(255)    NOT NULL,
  role       ENUM('customer','admin') NOT NULL DEFAULT 'customer',
  address    TEXT            NULL,
  created_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------------------------
-- CATEGORIES
-- ------------------------------------------------------------
CREATE TABLE categories (
  id   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(60)     NOT NULL,
  slug VARCHAR(60)     NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_categories_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------------------------
-- MENU ITEMS
-- ------------------------------------------------------------
CREATE TABLE menu_items (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  category_id BIGINT UNSIGNED NOT NULL,
  name        VARCHAR(120)    NOT NULL,
  slug        VARCHAR(140)    NOT NULL,
  description TEXT            NULL,
  price       BIGINT UNSIGNED NOT NULL DEFAULT 0,
  image       VARCHAR(255)    NOT NULL DEFAULT '/static/images/menu/placeholder.svg',
  available   TINYINT(1)      NOT NULL DEFAULT 1,
  created_at  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_menu_category (category_id),
  KEY idx_menu_available (available),
  CONSTRAINT fk_menu_category FOREIGN KEY (category_id)
    REFERENCES categories (id) ON DELETE RESTRICT ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------------------------
-- FAVORITES
-- ------------------------------------------------------------
CREATE TABLE favorites (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id      BIGINT UNSIGNED NOT NULL,
  menu_item_id BIGINT UNSIGNED NOT NULL,
  created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_fav_user_menu (user_id, menu_item_id),
  KEY idx_fav_menu (menu_item_id),
  CONSTRAINT fk_fav_user FOREIGN KEY (user_id)
    REFERENCES users (id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_fav_menu FOREIGN KEY (menu_item_id)
    REFERENCES menu_items (id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------------------------
-- CARTS
-- ------------------------------------------------------------
CREATE TABLE carts (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id      BIGINT UNSIGNED NOT NULL,
  menu_item_id BIGINT UNSIGNED NOT NULL,
  quantity     INT UNSIGNED    NOT NULL DEFAULT 1,
  created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_cart_user_menu (user_id, menu_item_id),
  KEY idx_cart_menu (menu_item_id),
  CONSTRAINT fk_cart_user FOREIGN KEY (user_id)
    REFERENCES users (id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_cart_menu FOREIGN KEY (menu_item_id)
    REFERENCES menu_items (id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------------------------
-- ORDERS
-- ------------------------------------------------------------
CREATE TABLE orders (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id        BIGINT UNSIGNED NOT NULL,
  order_number   VARCHAR(24)     NOT NULL,
  order_type     ENUM('dine_in','take_away','delivery') NOT NULL DEFAULT 'dine_in',
  subtotal       BIGINT UNSIGNED NOT NULL DEFAULT 0,
  delivery_fee   BIGINT UNSIGNED NOT NULL DEFAULT 0,
  total          BIGINT UNSIGNED NOT NULL DEFAULT 0,
  payment_method ENUM('cash','bank_transfer','ewallet') NOT NULL DEFAULT 'cash',
  status         ENUM('pending','processing','cooking','ready','delivering','served','completed','cancelled')
                 NOT NULL DEFAULT 'pending',
  address        TEXT            NULL,
  table_number   VARCHAR(30)     NULL,
  notes          TEXT            NULL,
  created_at     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_orders_number (order_number),
  KEY idx_orders_user (user_id, created_at),
  KEY idx_orders_status (status),
  KEY idx_orders_created (created_at),
  CONSTRAINT fk_order_user FOREIGN KEY (user_id)
    REFERENCES users (id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ------------------------------------------------------------
-- ORDER ITEMS
-- ------------------------------------------------------------
CREATE TABLE order_items (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_id     BIGINT UNSIGNED NOT NULL,
  menu_item_id BIGINT UNSIGNED NOT NULL,
  quantity     INT UNSIGNED    NOT NULL DEFAULT 1,
  price        BIGINT UNSIGNED NOT NULL DEFAULT 0,
  PRIMARY KEY (id),
  KEY idx_order_items_order (order_id),
  KEY idx_order_items_menu (menu_item_id),
  CONSTRAINT fk_oi_order FOREIGN KEY (order_id)
    REFERENCES orders (id) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_oi_menu FOREIGN KEY (menu_item_id)
    REFERENCES menu_items (id) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- SEED DATA
-- ============================================================

-- Demo accounts. Passwords below are bcrypt hashes:
--   admin@rantau.local    / admin123
--   customer@rantau.local / customer123
--   gadang@rantau.local   / gadang123
INSERT INTO users (name, email, phone, password, role, address) VALUES
('Admin Rantau', 'admin@rantau.local', '0812-0000-0001',
 '$2a$10$0LW/QhZU3B0Et/XuVR9zgOg9VghwQK6hVKHI9ZeJGABVfStaDJEbK', 'admin', 'Jl. Simpang Padang No. 1'),
('Budi Santoso', 'customer@rantau.local', '0812-3456-7890',
 '$2a$10$5sxN8E5hqL1jueALvQeOxeK28ffv47ESwDh8M7MaX9Ua4hJnB49uG', 'customer', 'Jl. Merdeka No. 10, Padang'),
('Siti Aminah', 'gadang@rantau.local', '0813-1111-2222',
 '$2a$10$1bmDw49Y53phZ5CZtvZVd.qywoQvQF5Ot.qRhp6wqMjiW2gQa.oI.', 'customer', 'Jl. Pemuda No. 5, Bukittinggi');

INSERT INTO categories (name, slug) VALUES
('Nasi',    'nasi'),
('Lauk',    'lauk'),
('Sayur',   'sayur'),
('Sambal',  'sambal'),
('Minuman', 'minuman'),
('Paket',   'paket');

-- NASI & LAUK
INSERT INTO menu_items (category_id, name, slug, description, price, image, available) VALUES
(1, 'Nasi Putih',       'nasi-putih',        'Nasi putih pulen hangat, cocok dengan segala lauk.', 5000,  '/static/images/menu/nasi-putih.svg', 1),
(2, 'Rendang',          'rendang',           'Daging sapi dimasak perlahan dengan bumbu rempah khas Minang hingga kering dan pekat.', 25000, '/static/images/menu/rendang.svg', 1),
(2, 'Ayam Pop',         'ayam-pop',          'Ayam kampung goreng putih yang gurih, disajikan dengan sambal dan air kelapa.', 20000, '/static/images/menu/ayam-pop.svg', 1),
(2, 'Ayam Bakar',       'ayam-bakar',        'Ayam bakar bumbu merah, manis-pedas dengan aroma daun jeruk.', 22000, '/static/images/menu/ayam-bakar.svg', 1),
(2, 'Dendeng Balado',   'dendeng-balado',    'Irisan daging sapi kering digoreng, diaduk dengan cabai balado.', 26000, '/static/images/menu/dendeng-balado.svg', 1),
(2, 'Dendeng Batokok',  'dendeng-batokok',   'Daging sapi dipukul tipis, dibakar lalu disiram sambal ijo khas.', 26000, '/static/images/menu/dendeng-batokok.svg', 1),
(2, 'Gulai Ayam',       'gulai-ayam',        'Ayam dalam kuah santan kuning pekat dengan rasa rempah yang dalam.', 22000, '/static/images/menu/gulai-ayam.svg', 1),
(2, 'Gulai Ikan',       'gulai-ikan',        'Ikan segar dimasak gulai santan dengan cabai hijau.', 24000, '/static/images/menu/gulai-ikan.svg', 1),
(2, 'Gulai Tunjang',    'gulai-tunjang',     'Kikil sapi lembut dalam gulai santan kaya rempah.', 23000, '/static/images/menu/gulai-tunjang.svg', 1),
(2, 'Telur Balado',     'telur-balado',      'Telur ayam digoreng lalu diaduk sambal balado pedas.', 12000, '/static/images/menu/telur-balado.svg', 1),
(2, 'Telur Dadar',      'telur-dadar',       'Telur dadar tebal, gurih, dengan taburan bawang goreng.', 10000, '/static/images/menu/telur-dadar.svg', 1),
(2, 'Ayam Goreng',      'ayam-goreng',       'Ayam goreng bumbu kuning yang renyah di luar, juicy di dalam.', 20000, '/static/images/menu/ayam-goreng.svg', 1),
(2, 'Paru Goreng',      'paru-goreng',       'Paru sapi goreng kering dengan sambal merah.', 20000, '/static/images/menu/paru-goreng.svg', 1),
(2, 'Gulai Nangka',     'gulai-nangka',      'Nangka muda dalam kuah santan kental, gurih dan hangat.', 18000, '/static/images/menu/gulai-nangka.svg', 1);

-- SAYUR
INSERT INTO menu_items (category_id, name, slug, description, price, image, available) VALUES
(3, 'Daun Singkong',   'daun-singkong',   'Daun singkong rebus lembut, disiram kuah santan gurih.', 8000,  '/static/images/menu/daun-singkong.svg', 1),
(3, 'Sayur Nangka',    'sayur-nangka',    'Nangka muda dimasak santan kuning dengan kacang panjang.', 8000,  '/static/images/menu/sayur-nangka.svg', 1),
(3, 'Sayur Kol',       'sayur-kol',       'Sayur kol tumis sederhana, segar dan ringan.', 7000,  '/static/images/menu/sayur-kol.svg', 1);

-- SAMBAL
INSERT INTO menu_items (category_id, name, slug, description, price, image, available) VALUES
(4, 'Sambal Ijo',      'sambal-ijo',      'Sambal cabai hijau khas Padang, pedas segar.', 4000,  '/static/images/menu/sambal-ijo.svg', 1),
(4, 'Sambal Merah',    'sambal-merah',    'Sambal cabai merah matang dengan tomat.', 4000,  '/static/images/menu/sambal-merah.svg', 1);

-- MINUMAN
INSERT INTO menu_items (category_id, name, slug, description, price, image, available) VALUES
(5, 'Es Teh',          'es-teh',          'Teh manis dingin yang menyegarkan.', 5000,   '/static/images/menu/es-teh.svg', 1),
(5, 'Teh Hangat',      'teh-hangat',      'Teh manis hangat, teman santap yang klasik.', 4000,   '/static/images/menu/teh-hangat.svg', 1),
(5, 'Jeruk Es',        'jeruk-es',        'Perasan jeruk manis dingin dengan es batu.', 8000,   '/static/images/menu/jeruk-es.svg', 1),
(5, 'Jeruk Hangat',    'jeruk-hangat',    'Perasan jeruk manis hangat.', 7000,   '/static/images/menu/jeruk-hangat.svg', 1),
(5, 'Kopi',            'kopi',            'Kopi hitam khas Nusantara, pekat dan aromatik.', 10000,  '/static/images/menu/kopi.svg', 1);

-- PAKET
INSERT INTO menu_items (category_id, name, slug, description, price, image, available) VALUES
(6, 'Paket Rendang',         'paket-rendang',         'Nasi, rendang, sayur nangka, dan sambal ijo.', 35000, '/static/images/menu/paket-rendang.svg', 1),
(6, 'Paket Ayam Pop + Teh',  'paket-ayam-pop-teh',    'Nasi, ayam pop, sambal merah, dan es teh manis.', 28000, '/static/images/menu/paket-ayam-pop.svg', 1),
(6, 'Paket Daging Ekstra',   'paket-daging-ekstra',   'Nasi, dendeng batokok, telur balado, dan gulai nangka.', 38000, '/static/images/menu/paket-daging-ekstra.svg', 1);

-- Popular sample: Budi's favorites feed the dashboard ranking.
INSERT INTO favorites (user_id, menu_item_id) VALUES
(2, 2), (2, 3), (2, 5), (3, 2), (3, 4);

-- ============================================================
-- SAMPLE ORDERS (for dashboard & reports demo)
-- ============================================================
INSERT INTO orders (user_id, order_number, order_type, subtotal, delivery_fee, total, payment_method, status, address, table_number, notes, created_at) VALUES
(2, 'RN-20260906-001', 'dine_in',  55000, 0, 55000,  'cash',         'completed',  NULL,      '12', 'Tidak terlalu pedas', DATE_SUB(NOW(), INTERVAL 2 DAY)),
(2, 'RN-20260906-002', 'delivery', 51000, 5000, 56000, 'bank_transfer', 'completed', 'Jl. Merdeka No. 10, Padang', NULL, 'Pesan sebelum jam 5 sore', DATE_SUB(NOW(), INTERVAL 1 DAY)),
(3, 'RN-20260907-001', 'take_away',80000, 0, 80000,   'ewallet',      'delivering', NULL,      NULL, 'Tambahkan sambal ijo ekstra', NOW() - INTERVAL 90 MINUTE),
(2, 'RN-20260907-002', 'dine_in',  68000, 0, 68000,   'cash',         'cooking',    NULL,      '4',  NULL, NOW() - INTERVAL 45 MINUTE);

INSERT INTO order_items (order_id, menu_item_id, quantity, price) VALUES
(1, 2, 1, 25000), (1, 3, 1, 20000), (1, 12, 1, 10000),
(2, 5, 1, 26000), (2, 20, 1, 5000), (2, 7, 1, 20000),
(3, 2, 2, 25000), (3, 5, 1, 26000), (3, 18, 1, 4000),
(4, 3, 2, 20000), (4, 8, 1, 24000), (4, 17, 1, 4000);