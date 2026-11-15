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

Middleware protected memeriksa format dan validitas JWT bearer. Endpoint
manajemen saat ini belum menerapkan pembatasan role/permission admin secara
khusus; jangan mengeksposnya ke publik tanpa authorization tambahan, misalnya
melalui API gateway.

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
  "password": "a",
  "full_name": "Alice Example"
}
```

`username`, `email`, `password`, dan `full_name` wajib. Email harus valid.
Password tidak memiliki batas minimal panjang dan di-hash sebelum disimpan.
Implementasi bcrypt menerima maksimal 72 byte; password yang lebih panjang
akan gagal saat hashing.

Sukses: HTTP `201`, `data` berisi user (`id`, `username`, `email`, `full_name`,
`status`). Password maupun hash password tidak dikembalikan.
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

### 4.3 Refresh token

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/auth/refresh` | JSON | Ya | HTTP `200`, envelope token baru |

```json
{
  "refresh_token": "<refresh_token>"
}
```

`refresh_token` wajib. Sukses: HTTP `200` dengan data token baru dalam bentuk
yang sama seperti login. Token tidak valid/kedaluwarsa mengembalikan HTTP
`401`; body invalid mengembalikan HTTP `400`.

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
GOOGLE_REDIRECT_URL=http://localhost:9001/auth/google/callback
```

Di Google Cloud, gunakan OAuth client tipe **Web application**, nama aplikasi
**UMMU Auth Service**, dan authorized redirect URI persis
`http://localhost:9001/auth/google/callback`. Isi client ID dan secret
bersamaan. Jangan masukkan secret ke repository atau Postman collection.

### 5.1 Memulai login

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `GET` | `/auth/google` | - | Tidak | HTTP `302`, redirect ke Google dan set cookie state |

Sukses: HTTP `302` redirect ke Google dan mengatur cookie HttpOnly
`google_oauth_state` (berumur lima menit, SameSite=Lax). Body tidak
disertakan. OIDC yang belum dikonfigurasi mengembalikan HTTP `503`.

### 5.2 Callback Google

| Method | Path | Header | Query | Body | Response sukses |
|---|---|---|---|---|---|
| `GET` | `/auth/google/callback` | Cookie `google_oauth_state` dari awal flow | `code`, `state` (atau `error` saat consent ditolak) | Tidak | HTTP `200`, envelope login/token |

Browser diarahkan Google ke callback ini. Service mencocokkan cookie dengan
query `state`, menukarkan `code`, kemudian mengembalikan HTTP `200` berupa
envelope login dengan access token, refresh token, dan user. Callback
mengembalikan JSON, bukan halaman frontend.

Error umum: HTTP `400` untuk state invalid atau code hilang; HTTP `401` untuk
login yang ditolak atau identitas/token Google tidak valid; HTTP `503` bila
OIDC belum dikonfigurasi.

## 6. OAuth2 client dan token

### 6.1 Registrasi dan pengelolaan client

Endpoint berikut memerlukan bearer token:

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
  "client_id": "web-client",
  "name": "Web Application",
  "client_secret": "replace-with-a-secret",
  "redirect_uris": [
    "http://localhost:3000/auth/callback"
  ],
  "grants": [
    "password",
    "refresh_token"
  ],
  "is_confidential": true
}
```

`client_id`, `name`, `client_secret`, `redirect_uris` (minimal satu string),
dan `grants` (minimal satu string) wajib. `is_confidential` boolean dan
default `false`. Secret di-hash sebelum disimpan.

`PUT` mengganti seluruh konfigurasi, termasuk daftar URI dan grant; kirim
seluruh body dan secret baru. Response client berisi `id`, `client_id`, `name`,
`redirect_uris`, `grants`, dan `is_confidential`. Secret dan hash secret tidak
pernah dikembalikan. Create sukses HTTP `201`; read/update/delete sukses HTTP
`200`; body invalid HTTP `400`; ID tidak ada HTTP `404`; client ID duplikat
HTTP `409`.

### 6.2 Token endpoint berbasis client

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/oauth/token` | JSON | Ya | HTTP `200`, envelope token |

Endpoint ini tidak memerlukan bearer token. Client harus terdaftar dan
menyediakan credential yang cocok.

Password grant:

```json
{
  "grant_type": "password",
  "client_id": "web-client",
  "client_secret": "replace-with-a-secret",
  "username": "alice",
  "password": "a"
}
```

Refresh-token grant:

```json
{
  "grant_type": "refresh_token",
  "client_id": "web-client",
  "client_secret": "replace-with-a-secret",
  "refresh_token": "<refresh_token>"
}
```

Grant yang diterima adalah `password` dan `refresh_token`. Sukses: HTTP `200`
dengan response token:

```json
{
  "success": true,
  "message": "token granted",
  "data": {
    "access_token": "<JWT>",
    "token_type": "Bearer",
    "expires_in": 900,
    "refresh_token": "<refresh_token>"
  }
}
```

`client_secret` tetap diwajibkan dan diverifikasi, termasuk jika
`is_confidential` bernilai `false`. Saat ini endpoint memvalidasi grant yang
didukung, tetapi tidak membandingkannya dengan daftar `grants` pada client.
Karena itu, konfigurasi `grants` dan `is_confidential` belum menjadi kontrol
akses efektif pada pertukaran token. Jangan mencatat client secret atau refresh
token di log.

## 7. User management

Semua endpoint user management memerlukan bearer token. Endpoint ini mengelola
akun autentikasi, bukan data profil lengkap pengguna.

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/users` | JWT + JSON | Ya | HTTP `201`, envelope user |
| `GET` | `/api/v1/users` | JWT | Tidak | HTTP `200`, envelope daftar user |
| `GET` | `/api/v1/users/{userId}` | JWT | Tidak | HTTP `200`, envelope user |
| `PUT` | `/api/v1/users/{userId}` | JWT + JSON | Ya | HTTP `200`, envelope `data: null` |
| `DELETE` | `/api/v1/users/{userId}` | JWT | Tidak | HTTP `200`, envelope `data: null` |

Create memakai body yang sama dengan registrasi lokal:

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "a",
  "full_name": "Alice Example"
}
```

Update menerima:

```json
{
  "email": "alice@example.com",
  "full_name": "Alice Example"
}
```

Response user tidak menyertakan password/hash. Create sukses HTTP `201`;
operasi lain sukses HTTP `200` dalam envelope standar. Path ID harus berupa
UUID.

## 8. Roles, permissions, dan relasi RBAC

Role mengelompokkan permission; satu user dapat memiliki banyak role dan
permission langsung. Permission efektif user menggabungkan direct permission
dengan permission role. Pengelolaan berikut memerlukan bearer token.

### 8.1 Role

| Method | Path | Header | Body | Response sukses |
|---|---|---|---|---|
| `POST` | `/api/v1/roles` | JWT + JSON | Ya | HTTP `201`, `{message, data: role}` |
| `GET` | `/api/v1/roles` | JWT | Tidak | HTTP `200`, `{message, data: role[]}` |
| `GET` | `/api/v1/roles/{roleId}` | JWT | Tidak | HTTP `200`, `{message, data: role}` |
| `PUT` | `/api/v1/roles/{roleId}` | JWT + JSON | Ya | HTTP `200`, `{message}` |
| `DELETE` | `/api/v1/roles/{roleId}` | JWT | Tidak | HTTP `200`, `{message}` |

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
| `POST` | `/api/v1/permissions` | JWT + JSON | Ya | HTTP `201`, `{message, data: permission}` |
| `GET` | `/api/v1/permissions` | JWT | Tidak | HTTP `200`, `{message, data: permission[]}` |
| `GET` | `/api/v1/permissions/{permissionId}` | JWT | Tidak | HTTP `200`, `{message, data: permission}` |
| `PUT` | `/api/v1/permissions/{permissionId}` | JWT + JSON | Ya | HTTP `200`, `{message}` |
| `DELETE` | `/api/v1/permissions/{permissionId}` | JWT | Tidak | HTTP `200`, `{message}` |

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
  "role_ids": ["<UUID-role>"]
}
```

PUT body:

```json
{
  "role_ids": ["<UUID-role-1>", "<UUID-role-2>"]
}
```

Kirim `{"role_ids":[]}` pada PUT untuk menghapus seluruh relasi user-role.
POST memerlukan minimal satu role ID.

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
  sedangkan user assignment memakai envelope API standar.

## 9. Postman

Import collection
[`UMMU Auth Service.postman_collection.json`](./UMMU%20Auth%20Service.postman_collection.json)
dan atur variable `baseUrl` bila service tidak berjalan di
`http://localhost:9001`. Folder koleksi mencakup Google OIDC, Auth, serta RBAC
dengan subfolder `user`, `roles`, `permission`, dan `client`.

Untuk operasi relasi, isi `user_id`, `role_id`, dan `permission_id` dengan UUID
resource yang sudah dibuat. Request client menggunakan
`client_internal_id`. Ikuti langkah pengujian OIDC di koleksi atau mulai login
melalui `/auth/google` pada browser; callback memakai cookie state yang
dibuat saat flow dimulai.

## 10. Keamanan dan batasan

- Jangan mencatat password, client secret, authorization code, JWT, atau refresh
  token ke log.
- Gunakan HTTPS di luar localhost, simpan secret di secret manager/environment,
  dan lindungi private signing key.
- Tambahkan authorization admin khusus untuk endpoint manajemen user, role,
  permission, dan client sebelum layanan dipublikasikan.
- Token endpoint saat ini menyediakan password grant dan refresh-token grant;
  jangan menganggap konfigurasi `grants` client sebagai enforcement sampai
  validasi client-grant diterapkan.
- Auth Service menyimpan identitas dan status autentikasi, bukan data profil
  lengkap atau data akademik/bisnis.
