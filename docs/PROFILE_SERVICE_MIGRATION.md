# Pemindahan `full_name` ke Profile Service

Auth Service tidak lagi menerima, menyimpan, mengubah, atau mengembalikan
`full_name`. Nama lengkap dan atribut kontak dimiliki oleh Profile Service.
Email login, password/OIDC, user ID, dan status akun tetap milik Auth Service.

## Migrasi database Auth yang sudah ada

Jangan langsung menjalankan migration penghapusan kolom sebelum mengimpor
nama lama. Proses berikut mempertahankan nama yang telah diisi pengguna:

1. Jalankan Profile Service dan pastikan Auth Service dapat dijangkau di Docker
   network bersama.
2. Pada `ummu-profile-service/.env`, isi variabel `LEGACY_AUTH_DB_*` sementara
   dengan koneksi ke database Auth lama. Jangan commit file `.env`.
3. Jalankan importer satu kali:

   ```powershell
   docker compose -f ummu-profile-service/docker-compose.yml run --rm `
     --entrypoint ./import-auth-profiles profile_service
   ```

   Importer hanya mengisi nama jika profil belum punya nama; nama profil yang
   sudah diedit di Profile Service tidak ditimpa. Jalankan ulang dengan aman
   bila perlu.
4. Verifikasi jumlah profil dan beberapa hasil melalui
   `GET /api/v1/profiles/{user_id}` menggunakan token user dengan permission
   `profile.admin:read` pada access context `global`.
5. Setelah hasil diverifikasi dan backup Auth tersedia, jalankan
   `auth-service/migrations/20261118_profile_ownership.sql` pada database Auth.

Database baru menggunakan schema Auth tanpa kolom `full_name` dari awal.

## Perubahan payload Auth

Sebelumnya request registrasi dan response user memuat `full_name`:

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "example",
  "full_name": "Alice Example"
}
```

```json
{
  "id": "user-uuid",
  "username": "alice",
  "email": "alice@example.com",
  "full_name": "Alice Example",
  "status": "active"
}
```

Sekarang registrasi hanya mengelola kredensial/identitas Auth:

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "example"
}
```

Response user Auth sekarang berisi `id`, `username`, `email`, dan `status`.
Klien mengambil serta memperbarui nama melalui Profile Service, misalnya
`GET /api/v1/profiles/me` dan `PATCH /api/v1/profiles/me`.

## Token untuk Profile Service

Auth login dan refresh menerbitkan access token dengan audience
`profile-service`, claim RBAC `permissions`, dan `access_contexts` dari role
user. OAuth `scope` tidak digunakan sebagai permission user. Permission admin
yang dipakai Profile Service adalah `profile.admin:read` dan
`profile.admin:update`; endpoint admin juga memerlukan permission tersebut
dalam access context `global`. Seed `super_admin` sudah memperoleh kedua
permission tersebut. Setelah perubahan role/permission, pengguna perlu
login/refresh untuk mendapatkan token baru.

Claim `roles` pada token adalah array kode role string, sedangkan permission
efektif berada di claim `permissions`. OAuth scope hanya digunakan pada token
client. Contoh user token: `"roles": ["super_admin"]` dan
`"permissions": ["profile.admin:read", "profile.admin:update"]`.
