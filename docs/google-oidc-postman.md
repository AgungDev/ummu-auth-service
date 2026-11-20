# Google OpenID Connect

Dokumen ini menjelaskan konfigurasi dan pengujian login Google OIDC pada UMMU
Auth Service menggunakan Postman.

## Konfigurasi Google Cloud

1. Buat OAuth client dengan tipe **Web application**.
2. Atur nama aplikasi/consent screen menjadi **UMMU Auth Service**.
3. Tambahkan authorized redirect URI persis:
   `http://localhost:7000/auth/google/callback`
4. Simpan OAuth client ID dan client secret.

## Konfigurasi service

Set variabel berikut pada `.env` yang digunakan service:

```env
APP_NAME=UMMU Auth Service
APP_PORT=9001
GOOGLE_CLIENT_ID=<OAuth client ID>
GOOGLE_CLIENT_SECRET=<OAuth client secret>
GOOGLE_REDIRECT_URL=http://localhost:7000/auth/google/callback
GOOGLE_FRONTEND_CALLBACK_URL=http://localhost:5173/oauth/callback
```

Client ID dan secret harus diisi bersama. Jangan simpan secret di repository,
Postman collection, atau file yang dibagikan. Saat kedua credential Google
tidak diisi, endpoint OIDC mengembalikan `503 Service Unavailable`.

Dengan callback `localhost` di atas, login Google hanya berfungsi jika browser
membuka frontend di komputer yang menjalankan service, misalnya
`http://localhost:5173`. Jika frontend dibuka dari perangkat lain melalui Wi-Fi,
`localhost` menunjuk ke perangkat tersebut, bukan komputer server. Untuk akses
Wi-Fi gunakan login username/password; Google login dari perangkat lain
memerlukan hostname HTTPS yang valid atau HTTPS tunnel dan URL callback yang
sama persis di Google Cloud Console.

Untuk database yang sudah dibuat sebelum dukungan Google ditambahkan, jalankan
SQL pada [`../migrations/20261115_google_oidc.sql`](../migrations/20261115_google_oidc.sql)
terhadap database Auth Service sebelum login Google digunakan. Instalasi baru
menggunakan schema di [`../migrations/init.sql`](../migrations/init.sql).

Restart service setelah konfigurasi dan migrasi selesai.

## Alur login

1. Untuk respons JSON/Postman, buka `http://localhost:7000/auth/google`.
   Untuk alur login SPA, buka
   `http://localhost:7000/auth/google?response=frontend`.
2. Service membuat cookie HttpOnly `google_oauth_state` berumur lima menit
   (SameSite=Lax) dan mengarahkan pengguna ke Google dengan scope `openid`,
   `email`, dan `profile`.
3. Setelah consent, Google mengarahkan browser ke callback yang terdaftar.
4. Service memeriksa cookie dan parameter `state`, menukarkan authorization
   code, lalu memverifikasi tanda tangan, issuer, audience, masa berlaku,
   `nonce`, dan email terverifikasi pada ID token.
5. Akun autentikasi dicari berdasarkan Google `sub`. Akun lokal yang cocok
   berdasarkan email terverifikasi dapat ditautkan; jika akun belum ada,
   service membuat akun autentikasi baru. Data profil mahasiswa, dosen,
   pegawai, dan data bisnis tidak dibuat atau disimpan oleh alur ini.
6. Untuk request biasa, callback mengembalikan access token JWT dan refresh
   token sebagai JSON. Untuk `response=frontend`, callback mengarahkan browser
   kembali ke `GOOGLE_FRONTEND_CALLBACK_URL` dan meneruskan token pada URL
   fragment. Frontend harus segera menghapus fragment setelah membaca token.

Gunakan URL frontend yang tetap dan tepercaya untuk
`GOOGLE_FRONTEND_CALLBACK_URL`. Jangan menaruh token pada query string.

## Pengujian dengan Postman

Import collection
[`UMMU Auth Service.postman_collection.json`](./UMMU%20Auth%20Service.postman_collection.json).
Collection menggunakan `http://localhost:9001` secara default; ubah variabel
collection `baseUrl` bila service berjalan di host atau port lain.

### OIDC menggunakan collection

Postman dan browser memiliki cookie jar berbeda. Gunakan prosedur berikut agar
cookie state yang dibuat oleh Postman tetap tersedia untuk permintaan callback:

1. Jalankan **Google OIDC / 1 - Start login** di Postman. Pastikan respons `302`
   dan `google_state` tersimpan otomatis dari URL redirect.
2. Salin URL pada header `Location` respons tersebut dan buka di browser yang
   sudah login ke akun Google yang akan diuji.
3. Google akan kembali ke callback service. Karena callback pertama dibuka di
   browser yang tidak memiliki cookie milik Postman, respons browser akan
   menampilkan `invalid Google OAuth state`. Ini terjadi sebelum authorization
   code ditukarkan; jangan memulai flow Google baru.
4. Salin nilai `code` dan `state` dari URL callback di address bar ke variabel
   collection `google_code` dan `google_state` di Postman.
5. Jalankan **Google OIDC / 2 - Complete callback** di Postman. Postman
   mengirim cookie state yang disimpan pada langkah pertama. Jika sukses,
   skrip request menyimpan `access_token` dan `refresh_token` ke variabel
   collection.
6. Jalankan **Auth / Get current user** untuk memeriksa access token.

Authorization code hanya dapat digunakan sekali dan berumur singkat. Jika
kedaluwarsa atau sudah tertukarkan, mulai dari langkah pertama.

### Endpoint lain dalam collection

- **Auth / Register** membuat akun lokal untuk pengujian.
- **Auth / Login** melakukan login username/password dan menyimpan token dari
  respons secara otomatis.
- **Auth / Refresh token** menggunakan `refresh_token` collection variable dan
  memperbarui kedua token setelah berhasil.
- **Auth / Get current user** dan **Auth / Logout** menggunakan bearer token.
- **Google OIDC / 1 - Start login** dan **Google OIDC / 2 - Complete callback**
  menguji flow Google sebagaimana dijelaskan di atas.
- Folder **RBAC** memuat contoh CRUD role/permission dan operasi relasi.
  Atur collection variables `user_id`, `role_id`, dan `permission_id` sebelum
  menjalankan request relasi. Dokumentasi semua endpoint ada di
  [`API_DOCUMENTATION.md`](./API_DOCUMENTATION.md).

Collection variable `username`, `password`, `google_code`, dan `google_state`
adalah contoh input pengujian. Ganti nilainya sesuai kebutuhan; jangan
menyimpan credential Google di collection.

## Respons dan error

Login sukses menggunakan envelope API berikut:

```json
{
  "success": true,
  "message": "login success",
  "data": {
    "access_token": "<JWT>",
    "token_type": "Bearer",
    "expires_in": 900,
    "refresh_token": "<refresh token>",
    "user": {
      "id": "<UUID>",
      "username": "<username>",
      "email": "<email>",
      "status": "active"
    }
  }
}
```

Beberapa error yang mungkin dijumpai:

- `503`: Google OIDC belum dikonfigurasi.
- `400 invalid Google OAuth state`: cookie hilang/kedaluwarsa atau `state`
  tidak cocok.
- `401 Google authentication failed`: code, nonce, token, email, atau status
  akun tidak valid.
- `401 Google login was not authorized`: pengguna menolak consent Google.
