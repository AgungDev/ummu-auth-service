# Database Auth Service

## Database dan schema

Docker Compose membuat satu database dengan nama dari `DB_NAME` (default
repository: `auth_service`). `migrations/init.sql` membuat schema dan akun dummy
saat PostgreSQL menginisialisasi volume database yang masih kosong. Seeder
Auth Service mengisi client, role, dan permission awal yang belum tersedia.

Auth Service terhubung ke PostgreSQL melalui host `db` dan port internal `5432`.
`DB_PORT` pada konfigurasi database adalah port yang dipublikasikan ke host;
port internal antarkontainer tetap `5432`.

PostgreSQL hanya menjalankan berkas init saat volume pertama kali dibuat.
Mengubah `DB_USER` atau `DB_PASSWORD` di `.env` tidak mengubah role di volume
yang sudah berisi database. Untuk menjaga data, gunakan kembali kredensial yang
dipakai saat volume itu dibuat. Reset volume hanya untuk database lokal yang
datanya memang boleh dihapus.

## Akun dummy lokal

Pada database baru tersedia akun berikut:

| Username | Password | Email |
|---|---|---|
| `admin` | `Password123!` | `admin@example.com` |

Akun ini hanya untuk pengembangan lokal. Ganti atau hapus sebelum deployment.
