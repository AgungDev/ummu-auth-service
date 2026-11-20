# UMMU Authorization Architecture

Dokumentasi standar Authentication & Authorization untuk ekosistem microservices Universitas Muhammadiyah Maluku Utara (UMMU).

Dokumen ini menjadi acuan implementasi Auth Service, RBAC, API Gateway, dan seluruh Resource Server.

## 1. Architecture Principles

UMMU menggunakan pendekatan centralized authentication dengan distributed authorization enforcement.

Prinsip utama:

- Auth Service menangani autentikasi dan penerbitan token.
- User dan Client memiliki identitas autentikasi yang berbeda.
- RBAC mengelola Role dan Permission pengguna.
- OAuth Scope mengatur izin OAuth untuk aplikasi dan service.
- Data Access Scope membatasi cakupan data yang boleh diakses.
- Setiap Resource Server wajib memvalidasi authorization sebelum memberikan akses data.

## 2. Authorization Concepts

Jangan mencampuradukkan empat konsep berikut.

| Concept | Purpose | Example |
|---|---|---|
| Role | Identitas kewenangan pengguna | `student`, `lecturer`, `admin` |
| Permission | Hak melakukan tindakan | `student:read`, `student:update` |
| OAuth Scope | Izin yang diminta/diberikan kepada OAuth client | `student:read`, `openid`, `profile` |
| Data Access Scope | Batas cakupan data | `global`, `faculty`, `department`, `self` |

### Role

Role merupakan kumpulan permission yang diberikan kepada pengguna.

Contoh:

- `student`
- `lecturer`
- `faculty_admin`
- `department_admin`
- `university_admin`
- `super_admin`

### Permission

Permission menentukan tindakan yang diperbolehkan.

Format:

`resource:action`

Contoh:

```text
student:read
student:create
student:update
student:delete

course:read
course:create
course:update

grade:read
grade:create
grade:update
```

Permission dikelola secara terpusat melalui RBAC.

### OAuth Scope

OAuth Scope merupakan izin OAuth yang diminta oleh client ketika melakukan proses authorization.

Contoh:

```text
openid
profile
email
student:read
student:write
```

Untuk Client Credentials, OAuth Scope menentukan izin service yang diberikan oleh Authorization Server.

OAuth Scope dan Permission boleh menggunakan nama yang sama, tetapi keduanya merupakan konsep berbeda.

### Data Access Scope

Data Access Scope menentukan batas data yang dapat diakses.

Jenis scope standar UMMU:

| Scope | Description |
|---|---|
| `global` | Seluruh data institusi |
| `faculty` | Data fakultas tertentu |
| `department` | Data program studi/departemen tertentu |
| `self` | Data milik pengguna sendiri |

Scope harus memiliki identifier jika cakupannya spesifik.

Contoh:

```json
{
  "type": "faculty",
  "id": "faculty-uuid"
}
```

Global:

```json
{
  "type": "global",
  "id": null
}
```

Self:

```json
{
  "type": "self",
  "id": "user-uuid"
}
```

Data Access Scope bukan Permission dan tidak boleh digunakan untuk menggantikan Permission.

## 3. Principal Types

UMMU memiliki dua jenis principal:

1. User Principal: manusia yang menggunakan aplikasi.
2. Client Principal: aplikasi atau microservice yang meminta akses.

### 3.1 User Authorization

User menggunakan:

- User ID
- Role
- Permission
- Data Access Scope

Contoh JWT:

```json
{
  "iss": "https://auth.ummu.ac.id",
  "sub": "user-uuid",
  "aud": ["siak-api"],
  "iat": 1790000000,
  "exp": 1790000900,
  "jti": "token-uuid",
  "principal_type": "user",
  "roles": [
    "lecturer"
  ],
  "permissions": [
    "student:read",
    "course:read",
    "grade:read"
  ],
  "access_contexts": [
    {
      "role": "lecturer",
      "scope": {
        "type": "faculty",
        "id": "faculty-uuid"
      },
      "permissions": ["student:read", "course:read", "grade:read"]
    },
    {
      "role": "lecturer",
      "scope": {
        "type": "department",
        "id": "department-uuid"
      },
      "permissions": ["student:read", "course:read", "grade:read"]
    }
  ]
}
```

Ketentuan:

- Satu user dapat memiliki beberapa role.
- Satu user dapat memiliki beberapa access context.
- Permission ditentukan melalui RBAC.
- Implementasi menaruh subset permission role pada setiap access context agar
  Resource Server dapat mengikat izin ke scope yang benar.
- Data Access Scope membatasi akses berdasarkan organisasi.
- Role tidak boleh menjadi satu-satunya dasar authorization.

### 3.2 Client Authorization

Client menggunakan:

- Client ID
- OAuth Scope
- Data Access Scope (opsional)

Contoh JWT:

```json
{
  "iss": "https://auth.ummu.ac.id",
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "aud": ["siak-legacy-api"],
  "iat": 1790000000,
  "exp": 1790000300,
  "jti": "token-uuid",
  "principal_type": "client",
  "scope": "student:read",
  "access_contexts": [
    {
      "scope": {
        "type": "faculty",
        "id": "faculty-uuid"
      }
    }
  ]
}
```

Ketentuan:

- Client tidak menggunakan User ID sebagai identitasnya.
- Client Credentials tidak mewakili pengguna manusia.
- OAuth Scope menentukan izin client.
- Data Access Scope bersifat opsional dan digunakan ketika service perlu dibatasi berdasarkan organisasi atau data tertentu.
- Client hanya boleh memperoleh scope yang telah diizinkan untuk client tersebut.

Jika client tidak memiliki batasan organisasi, Data Access Scope tidak perlu ditambahkan secara otomatis.

## 4. OAuth 2.0 Grant Types

### Authorization Code + PKCE

Digunakan untuk aplikasi web dan mobile yang bertindak atas nama pengguna.

Request authorization:

```http
GET /api/v1/auth/authorize
```

Contoh parameter:

```text
response_type=code
&client_id=<client-uuid>
&redirect_uri=https://siak.ummu.ac.id/callback
&scope=openid profile email
&state=random-state
&code_challenge=PKCE_CHALLENGE
&code_challenge_method=S256
```

Gunakan Authorization Code Flow dengan PKCE.

### Client Credentials

Digunakan untuk komunikasi antarmicroservice.

Request:

```http
POST /api/v1/oauth/token
Content-Type: application/x-www-form-urlencoded
```

```text
grant_type=client_credentials
&client_id=<client-uuid>
&client_secret=CLIENT_SECRET
&scope=student:read
```

Response:

```json
{
  "access_token": "service-access-token",
  "token_type": "Bearer",
  "expires_in": 300,
  "scope": "student:read"
}
```

Jangan menggunakan Password Grant.

## 5. Authorization Flow

### User Request

1. User login melalui Auth Service.
2. Auth Service menerbitkan access token.
3. User mengirim token ke API Gateway.
4. API Gateway memvalidasi token.
5. Resource Server memeriksa permission.
6. Resource Server memeriksa Data Access Scope.
7. Query dibatasi sesuai access context.

Contoh:

Dosen meminta data mahasiswa.

```text
Permission: student:read
Data Access Scope: faculty
Faculty ID: faculty-uuid
```

Hasil: dosen hanya dapat membaca mahasiswa dari fakultas yang diizinkan.

### Client Request

1. Client melakukan autentikasi menggunakan Client Credentials.
2. Auth Service memvalidasi client ID dan client secret.
3. Auth Service memeriksa OAuth Scope yang diizinkan.
4. Auth Service menerbitkan service access token.
5. Client mengakses Resource Server.
6. Resource Server memeriksa client identity, audience, OAuth Scope, dan Data Access Scope jika tersedia.

## 6. Authorization Rules

Setiap Resource Server wajib menerapkan aturan berikut:

1. Validasi JWT signature.
2. Validasi issuer (`iss`).
3. Validasi audience (`aud`).
4. Validasi expiration (`exp`).
5. Identifikasi principal type.
6. Periksa permission untuk User Principal.
7. Periksa OAuth Scope untuk Client Principal.
8. Periksa Data Access Scope jika operasi membutuhkan pembatasan data.
9. Terapkan least privilege.

Jangan mempercayai data authorization dari frontend.

Jangan mengizinkan client mengirim atau mengubah scope melalui request body untuk meningkatkan hak aksesnya.

Jangan menggabungkan beberapa access context menjadi akses global secara otomatis.

## 7. Security Requirements

- Gunakan JWT RS256 atau algoritma asimetris yang sesuai.
- Private key hanya tersedia untuk Authorization Server.
- Resource Server menggunakan public key melalui JWKS.
- Access token memiliki masa berlaku singkat.
- Refresh token disimpan dalam bentuk hash.
- Terapkan refresh token rotation.
- Client secret disimpan dalam bentuk hash jika memungkinkan untuk mekanisme verifikasi.
- Terapkan rate limiting pada endpoint autentikasi.
- Gunakan HTTPS.
- Pisahkan user token dan service token.
- Validasi audience untuk mencegah token digunakan pada service yang tidak dituju.
- Hindari menyimpan informasi pribadi yang tidak diperlukan di JWT.

## 8. Architecture Responsibility

| Service | Responsibility |
|---|---|
| Auth Service | Authentication, OAuth 2.0, OIDC, token issuance |
| RBAC Service | Role, Permission, Role Assignment |
| Profile Service | User profile |
| Academic Service | Student, Lecturer, Faculty, Department |
| HR Service | Employee and staff information |
| API Gateway | Token validation and request routing |
| Resource Server | Authorization enforcement and data filtering |

Auth Service tidak bertanggung jawab menyimpan seluruh data profil mahasiswa, dosen, maupun pegawai.

Academic Service dan HR Service tetap menjadi sumber data institusional.

## 9. Implementation Requirements

Saat mengimplementasikan atau memperbarui sistem:

- Periksa terlebih dahulu struktur kode dan database yang sudah tersedia.
- Pertahankan kompatibilitas API yang masih digunakan.
- Pisahkan model User Principal dan Client Principal.
- Pisahkan OAuth Scope, Permission, Role, dan Data Access Scope.
- Dukung multi-role dan multi-access-context.
- Dukung client dengan beberapa OAuth Scope.
- Dukung Data Access Scope opsional untuk client.
- Uji skenario privilege escalation.
- buat dummy data terpisah untuk client yang ingin di daftarkan
- buat akun dummy untuk super_admin emailnya arjun0nur@gmail.com passwordnya 'admin'
- Pertahankan clean arsitektur, buat folder yang rapi dan mudah di baca, jangan asal meneruh code sembarangan

## 10. Expected Result

Implementasi dianggap selesai apabila:

- User dapat login dan memperoleh User Access Token.
- Client dapat memperoleh Service Access Token.
- User Authorization menggunakan Role, Permission, dan Data Access Scope.
- Client Authorization menggunakan OAuth Scope dan Data Access Scope opsional.
- User dan Client tidak saling menyamar sebagai principal.
- Resource Server dapat memvalidasi dan membatasi akses secara konsisten.
- Sistem mendukung penambahan microservice baru tanpa mengubah fundamental authorization architecture.

---

**Architecture Decision**

UMMU menggunakan centralized identity management dan distributed authorization enforcement.

Perbedaan utamanya:

- User: Role + Permission + Data Access Scope.
- Client: OAuth Scope + Data Access Scope (opsional).

OAuth Scope dan Permission dapat memiliki nama yang sama, tetapi tidak boleh dianggap sebagai konsep yang identik.

Dokumen ini menjadi pedoman implementasi seluruh microservices UMMU.