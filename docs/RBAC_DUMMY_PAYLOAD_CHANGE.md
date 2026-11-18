# RBAC Dummy Data dan Perubahan Payload

Dokumen ini mencatat perubahan data seed RBAC dan bentuk payload API. Dokumen
ini membedakan `Before` dengan `After` agar perubahan dapat ditinjau dan diuji.

## 1. Dummy data

Data dummy dibuat untuk lima akun berikut. Password semua akun adalah
`Password123!`.

| Username | Nama | Role | Akun |
|---|---|---|---|
| `admin` | Admin Sistem | `super_admin` | `admin@example.com` |
| `operator` | Siti Operator Akademik | `academic_operator` | `operator@example.com` |
| `lecturer` | Dr. Rahman Dosen | `lecturer` | `lecturer@example.com` |
| `student` | Alya Mahasiswa | `student` | `student@example.com` |
| `finance` | Budi Keuangan | `finance_operator` | `finance@example.com` |

Data diisi oleh seeder aplikasi dan tersedia sebagai SQL alternatif di
[`pkg/data/dummy.sql`](../pkg/data/dummy.sql).

## 2. Sebelum perubahan

Sebelumnya seeder hanya membuat:

* Role `admin`
* Permission `all:access`
* Satu akun `admin`
* Data SQL dummy menggunakan namespace lama, yaitu `student:read`,
  `student:update`, dan `profile:read`

Contoh payload role lama:

```json
{
  "code": "admin",
  "name": "admin",
  "description": "Administrator role"
}
```

Contoh payload permission lama:

```json
{
  "code": "all:access",
  "name": "all:access",
  "description": "Full access permission"
}
```

Contoh assignment lama:

```json
{
  "permission_ids": [
    "55555555-5555-5555-5555-555555555555"
  ]
}
```

## 3. Sesudah perubahan

Seeder sekarang menggunakan role terpisah sesuai domain dan permission dengan
format `resource.action`. Contoh payload tetap kompatibel dengan DTO saat ini.

Contoh membuat role:

```json
{
  "code": "lecturer",
  "name": "Dosen",
  "description": "Mengakses proses akademik untuk kelas dan penugasan"
}
```

Response:

```json
{
  "message": "Role created successfully",
  "data": {
    "id": "10000000-0000-0000-0000-000000000003",
    "code": "lecturer",
    "name": "Dosen",
    "description": "Mengakses proses akademik untuk kelas dan penugasan"
  }
}
```

Contoh membuat permission:

```json
{
  "code": "academic.grade.update",
  "name": "Ubah nilai",
  "description": "Memperbarui nilai dalam periode yang diizinkan"
}
```

Response:

```json
{
  "message": "Permission created successfully",
  "data": {
    "id": "20000000-0000-0000-0000-000000000009",
    "code": "academic.grade.update",
    "name": "Ubah nilai",
    "description": "Memperbarui nilai dalam periode yang diizinkan"
  }
}
```

Contoh assignment role kepada user:

```json
{
  "role_ids": [
    "10000000-0000-0000-0000-000000000003"
  ]
}
```

Contoh assignment permission kepada role:

```json
{
  "permission_ids": [
    "20000000-0000-0000-0000-000000000007",
    "20000000-0000-0000-0000-000000000008",
    "20000000-0000-0000-0000-000000000009"
  ]
}
```

Contoh payload cek permission:

```json
{
  "permission": "academic.grade.update"
}
```

Response:

```json
{
  "allowed": true
}
```

## 4. Mapping role dummy

| Role | Permission utama |
|---|---|
| `super_admin` | `system.role.manage`, `system.permission.manage`, `system.audit.read` |
| `academic_operator` | `student.read`, `student.create`, `student.update`, `academic.krs.read`, `academic.krs.approve`, `academic.grade.read` |
| `lecturer` | `student.read`, `academic.krs.read`, `academic.grade.read`, `academic.grade.create`, `academic.grade.update` |
| `student` | `student.read`, `academic.krs.read`, `academic.krs.create`, `academic.grade.read`, `finance.billing.read`, `finance.transaction.read` |
| `finance_operator` | `finance.billing.read`, `finance.transaction.read`, `finance.transaction.verify` |

## 5. Perubahan kompatibilitas

* Code lama `all:access`, `student:read`, `student:update`, dan `profile:read`
  tidak lagi menjadi baseline data seed.
* Code baru memakai satu format konsisten: `resource.action`.
* Assignments API tetap memakai array UUID pada field `role_ids` dan
  `permission_ids`; tidak ada perubahan nama field pada DTO saat ini.
* Scope dan masa berlaku belum ditambahkan ke tabel `user_roles` pada implementasi
  seed ini. Nilai seperti `faculty`, `department`, `self`, dan `assigned`
  mengikuti rancangan [RBAC](rancangan_roles_permissions_siak_modern_ummu.md)
  dan membutuhkan migration payload/kolom tambahan sebelum digunakan.
