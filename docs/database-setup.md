# Database Auth Service

## Database dan schema

Docker Compose membuat satu database dengan nama dari `DB_NAME` (default
repository: `auth_service`). `migrations/init.sql` hanya membuat schema saat
PostgreSQL menginisialisasi volume database yang masih kosong. Seeder dummy
dijalankan hanya pada `APP_ENV=development`; environment lain tidak membuat
akun atau client demo.

Untuk database yang sudah ada, jalankan migration secara berurutan setelah
`20261118_profile_ownership.sql`:

```powershell
psql -h localhost -p <DB_PORT> -U <DB_USER> -d <DB_NAME> -f migrations/20261120_authz_principals_scopes.sql
psql -h localhost -p <DB_PORT> -U <DB_USER> -d <DB_NAME> -f migrations/20261121_permission_codes_resource_action.sql
psql -h localhost -p <DB_PORT> -U <DB_USER> -d <DB_NAME> -f migrations/20261122_client_uuid_identity.sql
```

Migration ini menambahkan scope pada assignment role, allowlist OAuth scope dan
audience client, access context client, serta meng-hash refresh token lama.
Assignment role `super_admin` yang sudah ada dipertahankan sebagai access
context `global`.
Migration berikutnya menormalkan permission lama dari `resource.action` ke
`resource:action`; selesaikan collision code bila migration melaporkan duplikasi.
Database baru sudah memperoleh schema yang sama dari `init.sql`.
Migration UUID client menghapus kolom alias `clients.client_id`; primary key
UUID `clients.id` menjadi satu-satunya identitas client.

Auth Service terhubung ke PostgreSQL melalui host `db` dan port internal `5432`.
`DB_PORT` pada konfigurasi database adalah port yang dipublikasikan ke host;
port internal antarkontainer tetap `5432`.

PostgreSQL hanya menjalankan berkas init saat volume pertama kali dibuat.
Mengubah `DB_USER` atau `DB_PASSWORD` di `.env` tidak mengubah role di volume
yang sudah berisi database. Untuk menjaga data, gunakan kembali kredensial yang
dipakai saat volume itu dibuat. Reset volume hanya untuk database lokal yang
datanya memang boleh dihapus.

## Akun dummy lokal

Pada mode development, seeder membuat akun dan service client berikut:

| Principal | ID | Secret/password | Akses |
|---|---|---|---|
| Super admin | `arjun0nur@gmail.com` (`arjun0nur`) | `admin` | Role `super_admin`, access context `global` |
| Service client | UUID `id` dari daftar client | `secret123` | `client_credentials`, scope `student:read`, audience `siak-legacy-api` |

Akun sampel lain menggunakan password `Password123!`. Semua kredensial tersebut
hanya untuk pengembangan lokal; jangan seed atau gunakan di staging/production.
Ganti kredensial client dan hapus akun demo sebelum membuka service ke jaringan
di luar lingkungan development.

Akun ini hanya untuk pengembangan lokal. Ganti atau hapus sebelum deployment.
