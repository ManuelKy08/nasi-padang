# RANTAU — Rumah Makan Padang

**Raso Minang, Dihati.**

A fully functional **online ordering & restaurant management system** for a
Padang (Minangkabau) rice restaurant, built with **Go**, **MySQL/MariaDB** and
vanilla HTML/CSS/JS.

**Created by Risky Manuel Tamba**

---

## 1. Project Overview

RANTAU lets customers register, browse a menu, favourite dishes, order online
(dine-in / take-away / delivery), track their order, get a printable digital
receipt and manage their profile — while admins manage the menu, orders, users
and view sales reports. The whole thing runs server-side with a modern,
Minangkabau-inspired light & dark UI.

## 2. Requirements

- Go 1.22+ (developed on 1.26)
- MySQL 8 / MariaDB 10.4+
- Browser with ES2020 (modern Chrome/Firefox/Safari)

## 3. Installation

```bash
git clone <this-repo> rantau
cd rantau
go mod tidy
```

## 4. Database Setup

Start your MySQL / MariaDB server, then run:

```bash
mysql -u root -p < database/rantau.sql
```

This creates the `rantau` database with tables, seed menu items and demo data.
Finally create the app user (or adjust `.env`):

```sql
CREATE USER IF NOT EXISTS 'rantau'@'localhost' IDENTIFIED BY 'rantau_secret_2026';
GRANT ALL PRIVILEGES ON rantau.* TO 'rantau'@'localhost';
FLUSH PRIVILEGES;
```

## 5. Environment Variables

Copy `.env.example` to `.env` and adjust:

```bash
cp .env.example .env
```

| Variable       | Default                    | Description                              |
|----------------|----------------------------|------------------------------------------|
| `DB_HOST`      | `127.0.0.1`                | Database host                            |
| `DB_PORT`      | `3306`                     | Database port                            |
| `DB_USER`      | `rantau`                   | Database user                            |
| `DB_PASSWORD`  | `rantau_secret_2026`       | Database password                        |
| `DB_NAME`      | `rantau`                   | Database name                            |
| `SESSION_KEY`  | *(insecure default)*       | Secret for signed/encrypted cookies      |
| `PORT`         | `:8080`                    | HTTP listen address                      |
| `OPEN_HOUR`    | `10`                       | Restaurant opening hour (server clock)   |
| `CLOSE_HOUR`   | `22`                       | Restaurant closing hour                  |
| `DELIVERY_FEE` | `5000`                     | Delivery fee in Rupiah                   |

> **Important:** set a strong `SESSION_KEY` in production.

## 6. Running Locally

```bash
go mod tidy
go run .
# or
go build
./rantau
```

Then open http://localhost:8080

## 7. Demo Accounts

| Role     | Email                  | Password      |
|----------|------------------------|---------------|
| Admin    | `admin@rantau.local`   | `admin123`    |
| Customer | `customer@rantau.local`| `customer123` |
| Customer | `gadang@rantau.local`  | `gadang123`   |

Passwords are stored as **bcrypt hashes** in the database.

## 8. Project Structure

```
rantau/
├── main.go
├── config/database.go
├── handlers/            # HTTP handlers
├── models/              # data structs
├── middleware/          # auth, admin, csrf
├── repositories/        # persistence layer (SQL)
├── templates/           # Go html/template pages
├── static/              # css, js, images
└── database/rantau.sql  # schema + seed
```

## 9. Routes

| Method | Route                     | Access     | Description            |
|--------|---------------------------|------------|------------------------|
| GET    | `/login` `/register`      | public     | Auth pages             |
| POST   | `/login` `/register`      | public     | Auth actions           |
| POST   | `/logout`                 | any user   | End session            |
| GET    | `/dashboard`              | any user   | Personalised home      |
| GET    | `/about`                  | any user   | Brand story            |
| GET    | `/menu`, `/menu/{id}`     | any user   | Browse / detail        |
| GET    | `/favorites`              | any user   | Saved dishes           |
| GET    | `/cart`, `/checkout`      | any user   | Cart & checkout        |
| GET    | `/orders`, `/orders/{id}` | any user   | History & tracking     |
| GET    | `/receipt/{number}`       | owner      | Digital receipt        |
| GET    | `/profile`                | any user   | Edit profile           |
| GET    | `/admin*`                 | admin only | Dashboard, menu, orders, users, reports |
| POST   | various `/cart/*`, `/favorites/*`, `/admin/*` | auth/admin | actions (CSRF protected) |

## 10. Admin Access

Sign in as `admin@rantau.local` / `admin123`, then visit:

- `/admin` — headline stats & revenue chart
- `/admin/menu` — create / edit / delete / toggle menu availability + upload photos
- `/admin/orders` — view all orders and update status live
- `/admin/users` — manage customers/roles
- `/admin/reports` — today/week/month sales, top menu & categories, AOV

## 11. Security Notes

- bcrypt password hashing
- signed + encrypted (HMAC/AES-GCM) cookies via `gorilla/sessions`
- CSRF tokens on every mutating form & AJAX call
- server-side role-based authorization (not just hidden UI)
- prepared statements everywhere (SQL injection safe)
- server-side input validation + HTML auto-escaping
- validated image upload (type + size)
- secrets via environment variables / `.env`

## 12. Notes

- Operating hours use the **server clock** (`OPEN_HOUR` / `CLOSE_HOUR`).
- Images are polished SVG placeholders; replace them in `static/images/menu/`
  by re-uploading via the admin panel.
- Payment is **simulated** — no real gateway is contacted.

---

© 2026 RANTAU · Created by **Risky Manuel Tamba**