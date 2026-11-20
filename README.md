# auth_service — Layanan Otentikasi untuk Sistem Informasi Akademik

> Auth service khusus buat Universitas Muhammadiyah Maluku Utara. Semua proses login / otorisasi lewat sini. Ringkas, aman (usahakan), dan modular.

---

## Ringkasan singkat

`auth_service` adalah microservice bertanggung jawab penuh atas:

* Otentikasi (login) pengguna
* Otorisasi (RBAC — role & permission)
* Mengeluarkan user access token dan service access token berbasis JWT
* Manajemen client application (API clients yang dipakai internal seperti `gateway`, `portal-mahasiswa`, `admin-panel`)
* Endpoint introspeksi/validasi token untuk service lain

Panduan konfigurasi Google OpenID Connect dan Postman collection tersedia di
[`docs/google-oidc-postman.md`](docs/google-oidc-postman.md) dan
[`docs/UMMU Auth Service.postman_collection.json`](docs/UMMU%20Auth%20Service.postman_collection.json).
Dokumentasi API lengkap, termasuk authentication, OIDC, client, dan RBAC,
tersedia di [`docs/API_DOCUMENTATION.md`](docs/API_DOCUMENTATION.md).
Panduan pemindahan nama lengkap ke Profile Service tersedia di
[`docs/PROFILE_SERVICE_MIGRATION.md`](docs/PROFILE_SERVICE_MIGRATION.md).
Konfigurasi database dan akun dummy lokal tersedia di
[`docs/database-setup.md`](docs/database-setup.md).

Tech stack utama:

* Golang (minimal Go 1.20)
* Clean Architecture (domain/usecase/repository/controller)
* GORM (ORM, PostgreSQL)
* OAuth client credentials untuk komunikasi antarlayanan; password grant tidak didukung
* JWT (signed using RS256 — gunakan keypair)
* Docker + docker-compose untuk dev

---

## Tujuan & Ruang Lingkup

* Menyediakan central auth untuk seluruh microservices kampus
* Menegakkan kebijakan akses: siapa boleh akses apa
* Mudah di-integrasikan oleh API Gateway (mis. `api_gateway` akan memanggil endpoint introspect)
* User login memakai endpoint Auth/OIDC first-party yang sudah ada; OAuth Client Credentials hanya untuk service client. Password Grant tidak didukung.
* Authorization Code + PKCE belum tersedia untuk OAuth public clients.

---

## Konsep domain (secara ringkas)

* **User** — entitas pengguna (student, lecturer, staff, superadmin)
* **Role** — kumpulan permission dengan satu atau lebih data access context
* **Permission** — aksi spesifik bagi user principal (contoh: `student:read`)
* **OAuth scope** — izin client service (contoh: `student:read`), terpisah dari RBAC permission
* **Client** — principal service dengan allowlist audience dan OAuth scope
* **Token** — access token (JWT), refresh token (opaque atau juga JWT)

---

## Struktur repository (saran)

```
auth_service/
├─ cmd/
│  └─ server/             # entrypoint
├─ internal/
│  ├─ domain/             # entity definitions
│  ├─ repository/         # GORM implementations
│  ├─ usecase/            # business logic
│  ├─ transport/
│  │  ├─ http/            # http handlers, middlewares
│  │  └─ grpc/ (opsional)
│  └─ config/             # env + config loader
├─ migrations/            # sql / goose / golang-migrate
├─ docker/
├─ scripts/
├─ pkg/                   # shared helpers (jwt, crypto)
└─ Makefile
```

---

## Database schema (singkat)

Tabel utama:

* `users` (id, username, email, password_hash, status, created_at, updated_at)
* `roles` (id, name, description)
* `permissions` (id, name, description)
* `role_permissions` (role_id, permission_id)
* `user_roles` (user_id, role_id, scope_type, scope_id)
* `clients` plus allowlist tables `client_scopes`, `client_audiences`, and `client_access_contexts`
* `client_grants` and `client_redirect_uris`
* `refresh_tokens` (user_id, client_id, refresh_token_hash, audience, scope, expires_at)

> Tips: jangan simpan password atau client secret secara plain — hash dengan argon2id / bcrypt.

---

## Alur Otentikasi & Otorisasi (singkat)

1. **User authentication**

   * Aplikasi pihak pertama menggunakan `POST /api/v1/auth/login` atau Google OIDC.
   * Token user memuat `principal_type=user`, `roles`, `permissions`, dan `access_contexts`.
2. **Service-to-service authentication**

   * Client confidential menukar credential melalui `POST /api/v1/oauth/token` dengan `grant_type=client_credentials`.
   * Token client memuat `principal_type=client`, `client_id` (UUID client), `aud`, `scope`, dan optional `access_contexts`; tidak memuat role atau permission user.
3. **Validasi token oleh service lain**

   * API Gateway/service memanggil `/introspect` atau memverifikasi JWT menggunakan public key (recommended)

Security notes: gunakan HTTPS, RS256 untuk JWT (private key aman di KMS / vault), expirasi singkat untuk access token (mis. 15m - 1h), refresh token lebih panjang tapi revokable.

---

## RBAC: contoh implementasi

Permissions use `resource:action`, contoh:

* `course:read`
* `course:write`
* `grade:read`
* `grade:write`
* `user:manage`

Role -> Permission mapping disimpan di `role_permissions`.

Saat membuat token, sertakan `roles` (atau `permissions`) di claim JWT untuk mempermudah pengecekan.

Contoh claims minimal:

```json
{
  "sub": "user-uuid",
  "iss": "https://auth.ummu.ac.id",
  "principal_type": "user",
  "exp": 1790000900,
  "jti": "token-uuid",
  "roles": ["student"],
  "permissions": ["course:read", "profile:read"],
  "access_contexts": [
    {
      "role": "student",
      "scope": {"type": "self", "id": "user-uuid"},
      "permissions": ["course:read", "profile:read"]
    }
  ]
}
```

---

## Endpoint API (draft)

### Public / OAuth endpoints

* `GET /auth/google` — memulai login Google OpenID Connect
* `GET /auth/google/callback` — callback Google (redirect URI: `http://localhost:7000/auth/google/callback`)
* `POST /api/v1/oauth/token` — client credentials; password grant ditolak
* `GET /api/v1/auth/authorize` — belum tersedia; Authorization Code + PKCE belum didukung
* `POST /oauth/revoke` — revocation
* `POST /oauth/introspect` — introspeksi token (opsional jika service lain tidak mau/ bisa verifikasi JWT)

### Management (butuh admin role)

* `GET  /api/v1/users` — list users
* `GET  /api/v1/users/{id}`
* `POST /api/v1/users` — create user
* `PUT  /api/v1/users/{id}` — update user
* `DELETE /api/v1/users/{id}` — deactivate user
* `GET  /api/v1/roles`
* `POST /api/v1/roles`
* `POST /api/v1/roles/{id}/permissions` — assign permission
* `GET  /api/v1/clients`
* `POST /api/v1/clients` — register client (for internal apps)
* `POST /api/v1/tokens/revoke`

> Semua endpoint management harus diproteksi oleh role `admin` / `superadmin`.

---

## Contoh .env (development)

```
APP_ENV=development
APP_PORT=8080
DATABASE_URL=postgres://auth:password@db:5432/authdb?sslmode=disable
JWT_PRIVATE_KEY_PATH=/run/secrets/jwt_private.pem
JWT_PUBLIC_KEY_PATH=/run/secrets/jwt_public.pem
ACCESS_TOKEN_EXPIRE_MINUTES=15
REFRESH_TOKEN_EXPIRE_DAYS=30
SIG_KEY_ALGO=RS256
```

Untuk mengaktifkan Google OIDC, isi `GOOGLE_CLIENT_ID` dan
`GOOGLE_CLIENT_SECRET` dari OAuth client bertipe Web application. Atur nama
consent screen menjadi `UMMU Auth Service` dan daftarkan redirect URI persis
`http://localhost:7000/auth/google/callback`. Untuk mengembalikan hasil login ke
SPA, atur `GOOGLE_FRONTEND_CALLBACK_URL` ke URL frontend, misalnya
`http://localhost:5173/oauth/callback`. Login menggunakan scope `openid`,
`email`, dan `profile`; akun autentikasi hanya menyimpan subject Google.
Pengguna baru dibuat berdasarkan email terverifikasi Google, dan akun lokal
dengan email yang sama akan ditautkan ke identitas tersebut.
Untuk database yang sudah ada, jalankan `migrations/20261115_google_oidc.sql`
sebelum mengaktifkan login Google; schema instalasi baru sudah memuat indeksnya.
Untuk database yang sudah ada, jalankan migration terbaru secara berurutan,
termasuk `migrations/20261120_authz_principals_scopes.sql` dan
`migrations/20261121_permission_codes_resource_action.sql` serta
`migrations/20261122_client_uuid_identity.sql`. Refresh token yang
telah tersimpan dalam bentuk plaintext akan di-hash oleh migration
authorization; migration berikutnya menormalkan kode permission legacy.

---

## Docker & docker-compose (saran singkat)

* `docker-compose.yml` untuk dev: postgres, migration, auth_service
* Gunakan secrets untuk private key dan client secrets

### Menjalankan aplikasi

1. Salin `.env.example` menjadi `.env`, lalu sesuaikan konfigurasi database:

   ```powershell
   Copy-Item .env.example .env
   ```

2. Buat key JWT di root repository (membutuhkan OpenSSL):

   ```powershell
   openssl genrsa -out jwt_private.pem 2048
   openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem
   ```

3. Build dan jalankan database serta aplikasi:

   ```powershell
   docker compose up --build db auth_service
   ```

   Setelah mengubah source code, jalankan kembali perintah tersebut agar image
   aplikasi dibangun ulang. Log aplikasi dapat dilihat dengan
   `docker compose logs -f auth_service`.

---

## Migrasi & Seed

Database Compose menggunakan satu nama dari `DB_NAME`. Schema dimuat oleh
`migrations/init.sql`; seeder hanya membuat akun dan service client demo saat
`APP_ENV=development`. Lihat
[`docs/database-setup.md`](docs/database-setup.md).

---

## Contoh alur pembuatan token (curl)

```
# Client Credentials grant (development only)
curl -X POST http://localhost:9001/api/v1/oauth/token `
  -H "Content-Type: application/x-www-form-urlencoded" `
  -d "grant_type=client_credentials&client_id=CLIENT_UUID&client_secret=secret123&audience=siak-legacy-api&scope=student%3Aread"

# Response: {"access_token":"...","token_type":"Bearer","expires_in":900,"scope":"student:read"}
```

---

## Tips implementasi - hal yang sering salah

1. **Jangan** sign JWT dengan HMAC using shared secret untuk production — gunakan RS256 dan simpan private key aman.
2. **Jangan** masukkan permissions terlalu banyak di token jika sensitif; cukup roles, dan gunakan introspect untuk permission-full check bila perlu.
3. **Jangan** simpan refresh token tanpa cara revoke; simpan memungkinkan blacklist/ revocation list.
4. **Selalu** gunakan HTTPS di production.

---

## Testing

* Unit test untuk usecase bisnis
* Integration test: run Postgres (docker-compose), jalankan migration, tes endpoint oauth/token, introspect
* Contract tests untuk API Gateway (walau nanti)

---

## CI / CD (singkat)

* Build image, run lint, run unit tests
* Scan vuln image
* Deploy ke staging, jalankan smoke tests

---

## Roadmap (fase)

1. **Fase 1 (MVP)**: Password grant, JWT RS256, user/role management, client management, introspect
2. **Fase 2**: Authorization Code + PKCE, consent screen, OIDC discovery (`/.well-known/openid-configuration`)
3. **Fase 3**: Integrasi KMS (key rotation), audit logs, 2FA (TOTP)

---

## References & Komponen cocok

* `github.com/go-oauth2/oauth2`
* `gorm.io/gorm`
* `github.com/golang-jwt/jwt/v5`
* `github.com/alexedwards/argon2id` (atau `bcrypt`)
