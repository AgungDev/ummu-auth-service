# UMMU Auth Service — API Documentation

Dokumen ini menjelaskan fungsi aplikasi, batas tanggung jawab, konfigurasi,
autentikasi API, dan kontrak endpoint yang tersedia.

## 1. Tentang aplikasi

UMMU Auth Service adalah layanan pusat untuk autentikasi dan otorisasi aplikasi
lain. Layanan ini mengelola akun autentikasi lokal, login Google OpenID Connect,
OAuth2 client, JWT access token, refresh token, session/logout, serta
authorization dasar berbasis role dan permission (RBAC).

Layanan ini **bukan** sumber data profil lengkap atau data bisnis. Data
mahasiswa, dosen, pegawai, akademik, keuangan, administrasi, dan permission
khusus setiap aplikasi dikelola oleh service masing-masing.

Implementasi mengikuti pemisahan clean architecture: domain dan DTO, use case,
repository, serta HTTP transport/controller. Konfigurasi aplikasi dan Google
OIDC dibaca dari konfigurasi service dan dependensi diinjeksikan saat server
dibangun.

## 2. Base URL dan format umum

Base URL pengembangan:

```text
http://localhost:9001
```

Sebagian besar API menggunakan prefix `/api/v1`. Endpoint Google OIDC berada
di luar prefix tersebut. Request dengan body JSON menggunakan:

```http
Content-Type: application/json
```

Endpoint yang memerlukan access token menggunakan:

```http
Authorization: Bearer <access_token>
```

Layanan menerima `X-Correlation-ID` untuk correlation ID; jika header tidak
dikirim, layanan membuat ID sendiri. Response menyertakan
`X-Correlation-ID`.

Auth Service menerbitkan JWT RS256 dengan issuer dari `JWT_ISSUER`, `kid` dari
`JWT_KEY_ID`, audience serta scope dari permintaan token yang diizinkan.
Resource Server mengambil kunci publik dari `/.well-known/jwks.json`; endpoint
ini hanya mengembalikan public key dan dapat di-cache.

Response standar API dibungkus sebagai:

```json
{
  "success": true,
  "message": "operation successful",
  "data": {}
}
```

Envelope tersebut digunakan oleh auth API, user management, dan token endpoint.
Controller role, permission, dan client mengembalikan objek `{ "message": ...,
"data": ... }` tanpa field `success`; error-nya umumnya berupa
`{ "error": "..." }`. Pemeriksaan permission langsung mengembalikan
`{ "allowed": true }`. Pada error envelope standar, `success` bernilai `false`,
dan `message` serta `error` berisi keterangan error.

Singkatan header pada tabel endpoint:

| Singkatan | Header yang dikirim |
|---|---|
| `-` | Tidak ada header khusus |
| `JSON` | `Content-Type: application/json` |
| `JWT` | `Authorization` header dengan credential JWT pada skema bearer |
| `JWT + JSON` | Header JWT dan `Content-Type: application/json` |

## 3. Ringkasan autentikasi endpoint

| Area | Endpoint | Access token |
|---|---|---|
| Registrasi, login, refresh, introspeksi | `/api/v1/auth/*` | Tidak |
| Profil dan authorization pengguna aktif | `/api/v1/auth/me`, `/api/v1/auth/check-permission`, `/api/v1/auth/permissions`, `/api/v1/auth/logout`, `/api/v1/auth/authorize` | Ya |
| Manajemen user, role, permission, dan client | `/api/v1/users/*`, `/api/v1/roles/*`, `/api/v1/permissions/*`, `/api/v1/clients/*` | Ya |
| Pertukaran token OAuth | `/api/v1/oauth/token` | Tidak; perlu client credentials |
| Google OpenID Connect | `/auth/google`, `/auth/google/callback` | Tidak; memakai state cookie dan code Google |
| Health check | `/api/v1/health` | Tidak |

### Health check

`GET /api/v1/health` memeriksa koneksi Auth Service ke database dan tidak
memerlukan access token. HTTP `200` berarti service dan database siap; HTTP
`503` berarti database tidak tersedia. Docker Compose menggunakan endpoint ini
untuk status health container Auth Service.

Middleware protected memvalidasi signature RS256, issuer, expiration, `jti`,
principal type, dan audience `auth-service`. Endpoint manajemen juga memeriksa
permission RBAC yang sesuai; API Gateway bukan pengganti pemeriksaan tersebut.

## 4. Authentication API

### 4.1 Registrasi akun lokal

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/auth/register` | JSON | Ya | HTTP `201`, envelope user |

Contoh body:

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "a"
}
```

`username`, `email`, dan `password` wajib. Email harus valid. Nama lengkap
dikelola terpisah oleh Profile Service melalui `/api/v1/profiles/me`.
Password tidak memiliki batas minimal panjang dan di-hash sebelum disimpan.
Implementasi bcrypt menerima maksimal 72 byte; password yang lebih panjang
akan gagal saat hashing.

Sukses: HTTP `201`, `data` berisi user (`id`, `username`, `email`, `status`).
Password maupun hash password tidak dikembalikan.
Body invalid atau kegagalan registrasi: HTTP `400`.

### 4.2 Login lokal

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/auth/login` | JSON | Ya | HTTP `200`, envelope token dan user |

```json
{
  "username": "alice",
  "password": "a"
}
```

Kedua field wajib. Sukses: HTTP `200`, envelope berisi `access_token`,
`token_type`, `expires_in`, `refresh_token`, dan objek `user`. Kredensial
invalid mengembalikan HTTP `401`; body invalid mengembalikan HTTP `400`.

Access token user memakai `principal_type=user` dan berisi claims `roles`,
`permissions`, serta `access_contexts`. Data access context berasal dari role
assignment (misalnya `faculty`, `department`, `self`, atau `global`); tidak ada
OAuth `scope` pada user token. Login user menghasilkan audience
`auth-service` dan `profile-service`.

```json
{
  "iss": "https://auth.ummu.ac.id",
  "principal_type": "user",
  "sub": "user-uuid",
  "aud": ["auth-service", "profile-service"],
  "iat": 1790000000,
  "exp": 1790000900,
  "roles": ["lecturer"],
  "permissions": ["student:read", "academic.grade:read"],
  "access_contexts": [
    {
      "role": "lecturer",
      "scope": {"type": "faculty", "id": "faculty-uuid"},
      "permissions": ["student:read", "academic.grade:read"]
    }
  ],
  "jti": "token-uuid"
}
```

### 4.3 Refresh token

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/auth/refresh` | JSON | Ya | HTTP `200`, envelope token baru |

```json
{
  "refresh_token": "<refresh_token>"
}
```

`refresh_token` wajib. Refresh token disimpan sebagai hash SHA-256 dan dirotasi
secara atomik pada setiap refresh. Sukses: HTTP `200` dengan data token baru
dalam bentuk yang sama seperti login. Token tidak valid/kedaluwarsa atau sudah
dirotasi mengembalikan HTTP `401`; body invalid mengembalikan HTTP `400`.

### 4.4 Introspeksi access token

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/auth/introspect` | JSON | Ya | HTTP `200`, envelope status token |

```json
{
  "token": "<access_token>"
}
```

`token` wajib. Sukses: HTTP `200`, envelope berisi status token `active`, serta
informasi user/role/permission yang tersedia. Body invalid mengembalikan HTTP
`400`; token yang tidak dapat divalidasi mengembalikan HTTP `401`.

### 4.5 Profil pengguna dari token

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `GET` | `/api/v1/auth/me` | JWT | Tidak | HTTP `200`, envelope user |

Sukses: HTTP `200`, envelope berisi profil autentikasi pemilik token. Token
tidak valid mengembalikan HTTP `401`; user tidak ditemukan mengembalikan HTTP
`404`.

### 4.6 Permission efektif pengguna aktif

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `GET` | `/api/v1/auth/permissions` | JWT | Tidak | HTTP `200`, envelope `data.permissions` |

Sukses: HTTP `200`, envelope berisi `data.permissions`, yaitu permission
efektif untuk user pada token (direct permission dan permission melalui role).

### 4.7 Memeriksa permission pengguna aktif

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/auth/check-permission` | JWT + JSON | Ya | HTTP `200`, `{ "allowed": true }` |

```json
{
  "permission": "article:publish"
}
```

`permission` wajib. Sukses: HTTP `200` dengan response:

```json
{
  "allowed": true
}
```

Body invalid mengembalikan HTTP `400`; token invalid mengembalikan HTTP `401`.

### 4.8 Otorisasi berdasarkan user dan permission

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/auth/authorize` | JWT + JSON | Ya | HTTP `200`, envelope hasil authorization |

```json
{
  "user_id": "<UUID-user>",
  "permission": "article:publish"
}
```

Kedua field wajib. Request hanya dapat mengotorisasi user yang sesuai dengan
subject token. Sukses: HTTP `200`, envelope berisi hasil authorization,
`user_id`, dan permission. Ketidakcocokan user atau akses yang ditolak
mengembalikan HTTP `403`.

### 4.9 Logout

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/auth/logout` | JWT | Tidak | HTTP `200`, envelope dengan `data: null` |

Mencabut sesi/refresh token milik user yang sedang login. Sukses: HTTP `200`
dengan envelope `data: null`; token invalid mengembalikan HTTP `401`.

## 5. Google OpenID Connect

Login menggunakan authorization code flow dengan scope `openid`, `email`, dan
`profile`. Google identity diverifikasi melalui OIDC, termasuk state, nonce,
issuer, audience, masa berlaku token, dan email terverifikasi. Akun lokal yang
memiliki email terverifikasi yang cocok dapat ditautkan; akun baru dibuat
sebagai akun autentikasi. Service tidak membuat profil bisnis.

Konfigurasi service:

```env
APP_NAME=UMMU Auth Service
APP_PORT=9001
GOOGLE_CLIENT_ID=<OAuth client ID>
GOOGLE_CLIENT_SECRET=<OAuth client secret>
GOOGLE_REDIRECT_URL=http://localhost:7000/auth/google/callback
GOOGLE_FRONTEND_CALLBACK_URL=http://localhost:5173/oauth/callback
```

Di Google Cloud, gunakan OAuth client tipe **Web application**, nama aplikasi
**UMMU Auth Service**, dan authorized redirect URI persis
`http://localhost:7000/auth/google/callback`. Isi client ID dan secret
bersamaan. Jangan masukkan secret ke repository atau Postman collection.

### 5.1 Memulai login

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `GET` | `/auth/google` | - | Tidak | HTTP `302`, redirect ke Google dan set cookie state |

Sukses: HTTP `302` redirect ke Google dan mengatur cookie HttpOnly
`google_oauth_state` (berumur lima menit, SameSite=Lax). Tambahkan query
`?response=frontend` untuk meminta hasil login dikembalikan ke
`GOOGLE_FRONTEND_CALLBACK_URL`; tanpa parameter tersebut, alur tetap ditujukan
untuk respons JSON/Postman. OIDC yang belum dikonfigurasi mengembalikan HTTP
`503`.

### 5.2 Callback Google

| Method | Path | Header | Query | Body | Response sukses |
|---|---|---|---|---|---|
| `GET` | `/auth/google/callback` | Cookie state dari awal flow | `code`, `state` (atau `error` saat consent ditolak) | Tidak | HTTP `200` JSON atau HTTP `302` ke frontend |

Browser diarahkan Google ke callback ini. Service mencocokkan cookie dengan
query `state`, menukarkan `code`, kemudian mengembalikan HTTP `200` berupa
envelope login dengan access token, refresh token, dan user pada alur default.
Untuk `response=frontend`, service mengarahkan browser kembali ke URL frontend
yang dikonfigurasi; token dikirim sebagai URL fragment agar tidak ikut pada
request HTTP berikutnya. Frontend harus segera membersihkan fragment dari URL.

Error umum: HTTP `400` untuk state invalid atau code hilang; HTTP `401` untuk
login yang ditolak atau identitas/token Google tidak valid; HTTP `503` bila
OIDC belum dikonfigurasi.

## 6. OAuth2 client dan token

### 6.1 Registrasi dan pengelolaan client

Endpoint berikut memerlukan bearer token user dengan audience `auth-service`
dan permission `system.client:manage`:

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/clients` | JWT + JSON | Ya | HTTP `201`, `{message, data: client}` |
| `GET` | `/api/v1/clients` | JWT | Tidak | HTTP `200`, `{message, data: client[]}` |
| `GET` | `/api/v1/clients/{id}` | JWT | Tidak | HTTP `200`, `{message, data: client}` |
| `PUT` | `/api/v1/clients/{id}` | JWT + JSON | Ya | HTTP `200`, `{message}` |
| `DELETE` | `/api/v1/clients/{id}` | JWT | Tidak | HTTP `200`, `{message}` |

Body create/update:

```json
{
  "name": "SIAK Legacy API",
  "client_secret": "replace-with-a-secret",
  "redirect_uris": [],
  "grants": ["client_credentials"],
  "scopes": ["student:read"],
  "audiences": ["siak-legacy-api"],
  "access_contexts": [
    {"scope_type": "faculty", "scope_id": "faculty-uuid"}
  ],
  "is_confidential": true,
  "status": "active"
}
```

`name`, `client_secret`, `grants`, `scopes`, dan `audiences` wajib.
Server membuat UUID `id` saat client dibuat; gunakan UUID ini sebagai OAuth
`client_id`. `redirect_uris` hanya diperlukan untuk grant redirect-based. Grant
`password` ditolak; `client_credentials` hanya tersedia bagi confidential
client. Client baru default `inactive`, sehingga token hanya dapat diterbitkan
setelah status diaktifkan. Client secret di-hash sebelum disimpan.

`PUT` mengganti konfigurasi client dan memerlukan secret baru. `scopes` dan
`audiences` adalah allowlist OAuth client yang terpisah dari RBAC permission.
`access_contexts` opsional dan membatasi data client; context kosong tidak
diubah menjadi `global`. Secret maupun hash secret tidak pernah dikembalikan.

### 6.2 Client Credentials token

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/oauth/token` | Form URL-encoded (JSON juga diterima) | Ya | HTTP `200`, OAuth token object |

Endpoint ini tidak memerlukan bearer token. Client harus aktif, confidential,
memiliki grant `client_credentials`, dan mengautentikasi dengan HTTP Basic atau
`client_id`/`client_secret` pada body.

Request:

```text
grant_type=client_credentials
&client_id=CLIENT_UUID
&client_secret=CLIENT_SECRET
&audience=siak-legacy-api
&scope=student:read
```

`CLIENT_UUID` adalah nilai `id` dari respons pembuatan client atau daftar
`GET /api/v1/clients`. `audience` harus diizinkan oleh client dan terdaftar di
`JWT_ALLOWED_AUDIENCES`. Semua scope yang diminta harus ada pada allowlist
client; scope tidak diperoleh dari permission user. Password grant dan refresh
grant di endpoint ini ditolak. Refresh user dilakukan melalui
`POST /api/v1/auth/refresh`.

Response sukses (HTTP `200`, tanpa envelope atau refresh token):

```json
{
  "access_token": "<JWT>",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "student:read"
}
```

Token client berisi `principal_type=client`, `sub=client_id` (UUID), `client_id` (UUID),
`aud`, `scope`, dan optional `access_contexts`; token tidak berisi role maupun
permission user. Scope tak terdaftar mengembalikan `400 invalid_scope`, client
tak valid `401 invalid_client`, grant tak diizinkan `400 unauthorized_client`.
Jangan mencatat client secret, password, access token, atau refresh token.

Konfigurasi penandatanganan resource token:

| Environment | Keterangan |
|---|---|
| `JWT_ISSUER` | URL issuer absolut; harus cocok dengan `AUTH_ISSUER` pada Resource Server. |
| `JWT_KEY_ID` | ID kunci RS256 yang dicantumkan sebagai `kid` dan JWKS. |
| `JWT_ALLOWED_AUDIENCES` | Daftar audience dipisahkan koma; harus mencakup audience target yang diminta client, misalnya `siak-legacy-api` atau `profile-service`. |
| `JWT_PRIVATE_KEY_PATH` / `JWT_PUBLIC_KEY_PATH` | Pasangan kunci RS256; hanya public key tersedia melalui JWKS. |

Permission user memakai format `resource:action`, misalnya `student:read` atau
`academic.grade:update`. OAuth scope client dikelola terpisah,
misalnya `student:read`; jangan menganggap keduanya setara atau memetakan scope
client dari role user.

## 7. User management

Semua endpoint user management memerlukan bearer token. Endpoint ini mengelola
akun autentikasi, bukan data profil lengkap pengguna.

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/users` | JWT + `user:create` | Ya | HTTP `201`, envelope user |
| `GET` | `/api/v1/users` | JWT + `user:read` | Tidak | HTTP `200`, envelope daftar user |
| `GET` | `/api/v1/users/{userId}` | JWT + `user:read` | Tidak | HTTP `200`, envelope user |
| `PUT` | `/api/v1/users/{userId}` | JWT + `user:update` | Ya | HTTP `200`, envelope `data: null` |
| `DELETE` | `/api/v1/users/{userId}` | JWT + `user:delete` | Tidak | HTTP `200`, envelope `data: null` |

Create memakai body yang sama dengan registrasi lokal:

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "a"
}
```

Update menerima:

```json
{
  "email": "alice@example.com"
}
```

Response user tidak menyertakan password/hash. Create sukses HTTP `201`;
operasi lain sukses HTTP `200` dalam envelope standar. Path ID harus berupa
UUID.

## 8. Roles, permissions, dan relasi RBAC

Role mengelompokkan permission; satu user dapat memiliki banyak role dan
beberapa scope context per role. Permission efektif user menggabungkan
permission role dan direct permission legacy. Direct permission tidak otomatis
mendapat access context. Permission memakai format `resource:action`, sedangkan
OAuth scope client memakai allowlist terpisah.

User access token memuat permission efektif dan `access_contexts`. Setiap
context berisi kode role, scope, dan permission yang memang berasal dari role
tersebut, sehingga Resource Server dapat menghindari penggabungan scope global
dari role yang berbeda. Untuk data profil sendiri, Profile Service memakai
`sub`; operasi profil admin memerlukan permission dan context `global`.

### 8.1 Role

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/roles` | JWT + `system.role:manage` | Ya | HTTP `201`, `{message, data: role}` |
| `GET` | `/api/v1/roles` | JWT + `system.role:read` | Tidak | HTTP `200`, `{message, data: role[]}` |
| `GET` | `/api/v1/roles/{roleId}` | JWT + `system.role:read` | Tidak | HTTP `200`, `{message, data: role}` |
| `PUT` | `/api/v1/roles/{roleId}` | JWT + `system.role:manage` | Ya | HTTP `200`, `{message}` |
| `DELETE` | `/api/v1/roles/{roleId}` | JWT + `system.role:manage` | Tidak | HTTP `200`, `{message}` |

Create/update:

```json
{
  "code": "content_manager",
  "name": "Content Manager",
  "description": "Mengelola konten aplikasi"
}
```

`code` dan `name` wajib dan unik. `PUT` mengganti seluruh field request;
`description` dapat dihapus dengan mengirim `""`.

### 8.2 Permission

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/permissions` | JWT + `system.permission:manage` | Ya | HTTP `201`, `{message, data: permission}` |
| `GET` | `/api/v1/permissions` | JWT + `system.permission:read` | Tidak | HTTP `200`, `{message, data: permission[]}` |
| `GET` | `/api/v1/permissions/{permissionId}` | JWT + `system.permission:read` | Tidak | HTTP `200`, `{message, data: permission}` |
| `PUT` | `/api/v1/permissions/{permissionId}` | JWT + `system.permission:manage` | Ya | HTTP `200`, `{message}` |
| `DELETE` | `/api/v1/permissions/{permissionId}` | JWT + `system.permission:manage` | Tidak | HTTP `200`, `{message}` |

Create/update:

```json
{
  "code": "article:publish",
  "name": "Publish article",
  "description": "Menerbitkan artikel"
}
```

`code` dan `name` wajib dan unik.

### 8.3 User-role

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `GET` | `/api/v1/users/{userId}/roles` | JWT | Tidak | HTTP `200`, envelope data role |
| `POST` | `/api/v1/users/{userId}/roles` | JWT + JSON | Ya | HTTP `200`, envelope `data: null` |
| `PUT` | `/api/v1/users/{userId}/roles` | JWT + JSON | Ya | HTTP `200`, envelope `data: null` |
| `DELETE` | `/api/v1/users/{userId}/roles/{roleId}` | JWT | Tidak | HTTP `200`, envelope `data: null` |

POST body:

```json
{
  "assignments": [
    {"role_id": "<UUID-role>", "scope_type": "faculty", "scope_id": "faculty-uuid"},
    {"role_id": "<UUID-role>", "scope_type": "department", "scope_id": "department-uuid"}
  ]
}
```

PUT body:

```json
{
  "assignments": [
    {"role_id": "<UUID-role-1>", "scope_type": "global"},
    {"role_id": "<UUID-role-2>", "scope_type": "self"}
  ]
}
```

Payload `role_ids` lama tetap diterima sebagai assignment `none` yang tidak
mendapat scope data; tidak pernah dianggap `global`. Scope yang didukung:
`global`, `faculty`, `department`, `unit`, `self`, dan `assigned`. Scope
organisasi membutuhkan `scope_id`; `self` selalu diikat ke ID user target.
Kirim array `assignments` kosong pada PUT untuk menghapus seluruh assignment.

### 8.4 User-permission langsung

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `GET` | `/api/v1/users/{userId}/permissions` | JWT | Tidak | HTTP `200`, envelope data permission |
| `POST` | `/api/v1/users/{userId}/permissions` | JWT + JSON | Ya | HTTP `200`, envelope `data: null` |
| `PUT` | `/api/v1/users/{userId}/permissions` | JWT + JSON | Ya | HTTP `200`, envelope `data: null` |
| `DELETE` | `/api/v1/users/{userId}/permissions/{permissionId}` | JWT | Tidak | HTTP `200`, envelope `data: null` |

POST body:

```json
{
  "permission_ids": ["<UUID-permission>"]
}
```

PUT menggunakan bentuk body yang sama. Kirim
`{"permission_ids":[]}` pada PUT untuk menghapus semua direct permission;
permission yang didapat dari role tidak terpengaruh. GET di sini hanya
menampilkan direct permission; gunakan `GET /api/v1/auth/permissions` untuk
permission efektif user dari access token.

### 8.5 Role-permission

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `GET` | `/api/v1/roles/{roleId}/permissions` | JWT | Tidak | HTTP `200`, `{message, data: permission[]}` |
| `POST` | `/api/v1/roles/{roleId}/permissions` | JWT + JSON | Ya | HTTP `200`, `{message}` |
| `PUT` | `/api/v1/roles/{roleId}/permissions` | JWT + JSON | Ya | HTTP `200`, `{message}` |
| `DELETE` | `/api/v1/roles/{roleId}/permissions/{permissionId}` | JWT | Tidak | HTTP `200`, `{message}` |

POST/PUT memakai body:

```json
{
  "permission_ids": ["<UUID-permission>"]
}
```

Kirim `{"permission_ids":[]}` pada PUT untuk menghapus seluruh relasi
role-permission.

### 8.6 Validasi dan respons RBAC

- ID path dan ID dalam array harus berupa UUID.
- Semua ID yang direferensikan harus ada; bulk assignment dibatalkan seluruhnya
  jika ada ID yang tidak ditemukan.
- POST assignment idempoten dan tidak menggandakan relasi yang telah ada.
- PUT assignment mengganti relasi secara atomik.
- DELETE pada endpoint relasi menghapus relasi, bukan resource terkait.
- Menghapus role atau permission menghapus relasi terkait melalui foreign key.
- Body invalid mengembalikan HTTP `400`, resource/ID tidak ditemukan `404`,
  nilai unik duplikat `409`, dan token tidak valid `401`.
- Create role/permission sukses HTTP `201`; operasi lain sukses HTTP `200`.
  Role/permission/client mengembalikan `{message, data}` atau `{message}`,
  sedangkan user assignment memakai envelope API standar. Endpoint manajemen
  memerlukan audience `auth-service` dan permission admin pada access context
  `global`.

## 9. Postman

Import collection
[`UMMU Auth Service.postman_collection.json`](./UMMU%20Auth%20Service.postman_collection.json)
dan atur variable `baseUrl` bila service tidak berjalan di
`http://localhost:9001`. Folder koleksi mencakup Google OIDC, Auth, OAuth Client
Credentials, serta RBAC dengan subfolder `user`, `roles`, `permission`, dan
`client`.

Untuk operasi relasi, isi `user_id`, `role_id`, dan `permission_id` dengan UUID
resource yang sudah dibuat. Request client menggunakan
UUID `id` sebagai `client_id` dan path parameter. Ikuti langkah pengujian OIDC di koleksi atau mulai login
melalui `/auth/google` pada browser; callback memakai cookie state yang
dibuat saat flow dimulai.

## 10. Keamanan dan batasan

- Jangan mencatat password, client secret, authorization code, JWT, atau refresh
  token ke log.
- Gunakan HTTPS di luar localhost, simpan secret di secret manager/environment,
  dan lindungi private signing key.
- Endpoint manajemen hanya menerima permission `resource:action` yang tercantum dalam
  access context `global`; service client tidak dapat memakai endpoint tersebut.
- `/api/v1/oauth/token` hanya menerima `client_credentials`; user login tetap
  menggunakan endpoint Auth/OIDC yang sudah ada. Authorization Code + PKCE
  untuk public OAuth clients belum tersedia.
- Auth Service menyimpan identitas dan status autentikasi, bukan data profil
  lengkap atau data akademik/bisnis.
