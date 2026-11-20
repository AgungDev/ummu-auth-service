# DOKUMEN RANCANGAN: ROLE & PERMISSION SIAK MODERN UMMU

**Rancangan model otorisasi berbasis RBAC, Scope, dan ABAC**

| Atribut | Keterangan |
| :--- | :--- |
| **Institusi** | Universitas Muhammadiyah Maluku Utara (UMMU) |
| **Sistem** | Modernisasi Sistem Informasi Akademik (SIAK) |
| **Dokumen** | Spesifikasi awal roles dan permissions |
| **Versi / Tanggal** | 1.0 / 16 November 2026 |

## Informasi Dokumen

| Elemen | Keterangan |
| :--- | :--- |
| **Tujuan** | Menjadi dokumen pendukung laporan modernisasi SIAK yang menjelaskan rancangan pengelolaan role, permission, dan pembatasan akses. |
| **Ruang lingkup** | Rancangan awal authorization untuk layanan SIAK modern berbasis microservices, termasuk pengguna manusia dan identitas layanan. |
| **Status** | Rancangan konseptual. Daftar role dan permission perlu divalidasi terhadap proses bisnis, struktur organisasi, dan kebijakan UMMU. |
| **Prinsip** | Least privilege, separation of duties, explicit authorization, auditability, dan pemisahan autentikasi dari data domain. |

---

## 1. Pendahuluan

Modernisasi SIAK memerlukan mekanisme pengelolaan hak akses yang konsisten, terukur, dan dapat dikembangkan mengikuti pertumbuhan layanan. Model akses yang hanya mengandalkan kategori umum seperti administrator dan pengguna berisiko menghasilkan kewenangan yang terlalu luas, sulit diaudit, serta tidak sesuai dengan pembagian tugas akademik dan administratif.

Dokumen ini mengusulkan model authorization berbasis **Role-Based Access Control (RBAC)**, dilengkapi pembatasan **scope** dan aturan berbasis atribut (**Attribute-Based Access Control / ABAC**). Role menentukan kelompok kewenangan, permission menentukan tindakan yang diizinkan, sedangkan scope dan atribut membatasi objek atau data yang dapat diakses.

---

## 2. Tujuan dan Prinsip Desain

* Menyediakan katalog role dan permission yang konsisten untuk seluruh domain SIAK.
* Membatasi akses berdasarkan tanggung jawab, unit organisasi, penugasan, dan kepemilikan data.
* Memisahkan identitas pengguna dari data profil mahasiswa, dosen, pegawai, dan transaksi.
* Mendukung komunikasi antarmicroservice dengan identitas layanan dan hak akses khusus.
* Memastikan keputusan akses dapat ditelusuri melalui audit log.

### 2.1 Prinsip yang digunakan

| Prinsip | Penerapan |
| :--- | :--- |
| **Least privilege** | Setiap pengguna atau service hanya memperoleh hak minimum yang dibutuhkan. |
| **Separation of duties** | Tindakan kritis, misalnya membuat dan memverifikasi transaksi, dapat dipisahkan ke peran berbeda. |
| **Default deny** | Akses ditolak jika tidak ada permission atau kebijakan yang secara eksplisit mengizinkan. |
| **Scope restriction** | Permission tidak otomatis memberi akses ke seluruh data; cakupan data tetap dibatasi. |
| **Auditability** | Pemberian, perubahan, dan pencabutan hak akses dicatat. |
| **Domain ownership** | Microservice pemilik data bertanggung jawab atas validasi aturan akses domain. |

---

## 3. Model Authorization yang Diusulkan

Model yang direkomendasikan adalah **Hybrid RBAC + ABAC** dengan pengelolaan katalog kebijakan secara terpusat dan penegakan kebijakan pada masing-masing microservice.

| Komponen | Fungsi | Contoh |
| :--- | :--- | :--- |
| **Subject** | Identitas yang melakukan permintaan. | Pengguna, dosen, mahasiswa, service account. |
| **Role** | Mengelompokkan tanggung jawab. | `academic_operator`, `lecturer`, `student`. |
| **Permission** | Menyatakan tindakan terhadap resource. | `student:read`, `academic.grade:update`. |
| **Scope** | Membatasi cakupan data. | `global`, `faculty`, `department`, `unit`, `self`, `assigned`. |
| **Attribute / Policy** | Memeriksa kondisi dan relasi bisnis. | Dosen hanya mengubah nilai kelas yang diampu. |
| **Enforcement** | Memutuskan dan menerapkan izin pada request. | API Gateway dan microservice terkait. |

### 3.1 Alur keputusan akses

1. Pengguna atau service melakukan autentikasi dan memperoleh token yang valid.
2. API Gateway memeriksa validitas token, audience, masa berlaku, dan kebijakan akses awal.
3. Microservice memeriksa permission yang dibutuhkan oleh endpoint atau operasi.
4. Microservice memeriksa scope, kepemilikan data, relasi penugasan, dan aturan bisnis.
5. Permintaan dijalankan hanya jika seluruh pemeriksaan yang diwajibkan terpenuhi; keputusan penting dicatat dalam audit log.

> **Catatan:** API Gateway bukan satu-satunya lapisan keamanan. Microservice tetap wajib menegakkan otorisasi karena akses internal, kesalahan konfigurasi gateway, atau pemanggilan antarlayanan tidak boleh melewati pemeriksaan domain.

---

## 4. Katalog Roles

Role berikut merupakan baseline untuk tahap awal. Penetapan final harus disesuaikan dengan nomenklatur jabatan dan struktur organisasi resmi UMMU.

### 4.1 System / Platform Roles

| Kode role | Nama | Deskripsi |
| :--- | :--- | :--- |
| `super_admin` | Super Administrator | Mengelola kebijakan dan konfigurasi platform lintas domain. Penggunaannya dibatasi untuk keadaan khusus. |
| `system_admin` | System Administrator | Mengelola konfigurasi teknis dan akun operasional sesuai kewenangan. |
| `security_admin` | Security Administrator | Mengelola kebijakan keamanan, akses, dan peninjauan kontrol. |
| `auditor` | Auditor | Membaca audit log dan informasi kepatuhan tanpa mengubah data operasional. |
| `developer` | Developer | Akses teknis non-produksi atau operasional terbatas sesuai prosedur; bukan akses bisnis otomatis. |

### 4.2 Academic Roles

| Kode role | Nama | Deskripsi |
| :--- | :--- | :--- |
| `student` | Mahasiswa | Mengakses informasi akademik dan layanan milik sendiri. |
| `lecturer` | Dosen | Mengakses proses akademik sesuai kelas, mata kuliah, dan penugasan. |
| `academic_advisor` | Dosen Pembimbing Akademik | Mengakses data mahasiswa yang berada dalam penugasan bimbingannya. |
| `academic_operator` | Operator Akademik | Menjalankan operasional akademik sesuai fakultas, program studi, atau unit. |
| `academic_admin` | Administrator Akademik | Mengelola administrasi akademik sesuai kewenangan. |
| `academic_manager` | Manajer Akademik | Memantau dan mengawasi proses akademik sesuai lingkup kewenangan. |

### 4.3 Finance & Administration Roles

| Kode role | Nama | Deskripsi |
| :--- | :--- | :--- |
| `finance_operator` | Operator Keuangan | Mengelola aktivitas transaksi dan tagihan sesuai kewenangan. |
| `finance_admin` | Administrator Keuangan | Mengelola administrasi dan konfigurasi domain keuangan. |
| `payment_operator` | Operator Pembayaran | Memeriksa dan memverifikasi pembayaran sesuai prosedur. |
| `admission_operator` | Operator Penerimaan | Mengelola proses penerimaan mahasiswa baru. |
| `administrative_operator` | Operator Administrasi | Mengelola proses administrasi umum. |
| `library_operator` | Operator Perpustakaan | Mengelola operasional layanan perpustakaan. |

### 4.4 Service Roles

| Kode role | Nama | Deskripsi |
| :--- | :--- | :--- |
| `service_account` | Service Account | Identitas non-manusia untuk komunikasi antarmicroservice dengan hak minimum. |
| `integration_client` | Integration Client | Identitas integrasi sistem eksternal dengan akses yang dibatasi. |
| `reporting_service` | Reporting Service | Akses baca terkontrol ke data yang diperlukan untuk pelaporan. |

> **Perhatian:** Role service harus dipisahkan dari role manusia. Service account tidak boleh menggunakan `super_admin` atau role operasional manusia. Setiap client sebaiknya memiliki identitas, kredensial, audience, dan permission tersendiri.

---

## 5. Katalog Permission

Format kode permission menggunakan pola `resource:action`. Gunakan nama resource dan action yang konsisten, spesifik, serta tidak bergantung pada nama endpoint atau framework.

### 5.1 User & Identity

| Permission | Keterangan |
| :--- | :--- |
| `user:read` | Melihat data akun pengguna yang diizinkan. |
| `user:create` | Membuat akun pengguna. |
| `user:update` | Memperbarui atribut akun yang diizinkan. |
| `user:delete` | Menonaktifkan atau menghapus akun sesuai kebijakan retensi. |
| `user:assign_role` | Memberikan role kepada pengguna. |
| `user:revoke_role` | Mencabut role dari pengguna. |
| `user:read_profile` | Melihat profil pengguna. |
| `user:update_profile` | Memperbarui profil pengguna. |

### 5.2 Student Management

| Permission | Keterangan |
| :--- | :--- |
| `student:read` | Melihat data mahasiswa sesuai scope. |
| `student:create` | Membuat data mahasiswa. |
| `student:update` | Memperbarui data mahasiswa. |
| `student:delete` | Menghapus atau menonaktifkan data sesuai kebijakan. |
| `student:read_academic` | Melihat riwayat akademik mahasiswa. |
| `student:read_personal` | Melihat data pribadi mahasiswa dengan pembatasan khusus. |
| `student:update_status` | Mengubah status akademik mahasiswa. |
| `student:export` | Mengekspor data mahasiswa sesuai kewenangan. |

### 5.3 Academic Management

| Permission | Keterangan |
| :--- | :--- |
| `academic.course:read` | Melihat data mata kuliah. |
| `academic.course:manage` | Mengelola data mata kuliah. |
| `academic.schedule:read` | Melihat jadwal akademik. |
| `academic.schedule:manage` | Mengelola jadwal akademik. |
| `academic.krs:read` | Melihat KRS. |
| `academic.krs:create` | Mengajukan atau mengisi KRS sesuai aturan. |
| `academic.krs:approve` | Menyetujui KRS sesuai kewenangan. |
| `academic.grade:read` | Melihat nilai. |
| `academic.grade:create` | Menginput nilai. |
| `academic.grade:update` | Memperbarui nilai dalam periode yang diizinkan. |
| `academic.grade:publish` | Mempublikasikan nilai. |
| `academic.transcript:read` | Melihat transkrip akademik. |
| `academic.transcript:export` | Mengekspor transkrip akademik. |

### 5.4 Lecturer Management

| Permission | Keterangan |
| :--- | :--- |
| `lecturer:read` | Melihat data dosen. |
| `lecturer:manage` | Mengelola data dosen. |
| `lecturer.assignment:read` | Melihat penugasan dosen. |
| `lecturer.assignment:manage` | Mengelola penugasan dosen. |
| `lecturer.attendance:manage` | Mengelola presensi perkuliahan sesuai penugasan. |

### 5.5 Finance & Payment

| Permission | Keterangan |
| :--- | :--- |
| `finance.billing:read` | Melihat tagihan. |
| `finance.billing:create` | Membuat tagihan. |
| `finance.billing:update` | Memperbarui tagihan sesuai aturan. |
| `finance.transaction:read` | Melihat transaksi. |
| `finance.transaction:create` | Membuat transaksi atau pencatatan transaksi. |
| `finance.transaction:verify` | Memverifikasi transaksi. |
| `finance.transaction:refund` | Memproses pengembalian dana sesuai otorisasi. |
| `finance.report:read` | Melihat laporan keuangan. |
| `finance.report:export` | Mengekspor laporan keuangan. |

### 5.6 Administration

| Permission | Keterangan |
| :--- | :--- |
| `administration.document:read` | Melihat dokumen administrasi. |
| `administration.document:create` | Membuat dokumen administrasi. |
| `administration.document:update` | Memperbarui dokumen administrasi. |
| `administration.document:approve` | Menyetujui dokumen sesuai kewenangan. |
| `administration.report:read` | Melihat laporan administrasi. |
| `administration.report:export` | Mengekspor laporan administrasi. |

### 5.7 System & Security

| Permission | Keterangan |
| :--- | :--- |
| `system.config:read` | Melihat konfigurasi sistem. |
| `system.config:update` | Mengubah konfigurasi sistem. |
| `system.audit:read` | Membaca audit log. |
| `system.role:read` | Melihat katalog role. |
| `system.role:manage` | Mengelola role. |
| `system.permission:read` | Melihat katalog permission. |
| `system.permission:manage` | Mengelola katalog permission. |
| `system.client:manage` | Mengelola OAuth client dan kredensial terkait. |
| `system.access:manage` | Mengelola kebijakan dan penetapan akses. |

---

## 6. Mapping Role terhadap Permission

Matriks berikut adalah contoh baseline, bukan matriks final. Tanda centang menunjukkan permission yang dapat dipertimbangkan untuk role tersebut. Setiap pemberian tetap harus dibatasi oleh scope, masa berlaku, dan aturan bisnis.

| Permission | Student | Lecturer | Academic Operator | Finance Operator |
| :--- | :---: | :---: | :---: | :---: |
| `student:read` | ✓* | ✓† | ✓ | — |
| `student:create` | — | — | ✓ | — |
| `student:update` | — | — | ✓ | — |
| `academic.krs:read` | ✓* | ✓† | ✓ | — |
| `academic.krs:create` | ✓* | — | — | — |
| `academic.krs:approve` | — | — | ✓ | — |
| `academic.grade:read` | ✓* | ✓† | ✓ | — |
| `academic.grade:create` | — | ✓† | — | — |
| `finance.billing:read` | ✓* | — | — | ✓ |
| `finance.transaction:verify` | — | — | — | ✓ |

*\* Mahasiswa hanya untuk data miliknya sendiri (scope `self`).*  
*† Dosen hanya untuk mahasiswa atau kelas yang terkait dengan penugasan yang sah (scope `assigned`).*

---

## 7. Scope dan Aturan Berbasis Atribut

Scope menentukan batas data yang dapat diakses oleh suatu penetapan role. Scope tidak menggantikan pemeriksaan aturan bisnis yang lebih rinci.

| Scope | Makna | Contoh penerapan |
| :--- | :--- | :--- |
| `global` | Cakupan seluruh universitas. | Pengelolaan konfigurasi platform oleh pejabat berwenang. |
| `faculty` | Cakupan fakultas tertentu. | Operator akademik Fakultas Teknik. |
| `department` | Cakupan program studi/departemen. | Operator program studi tertentu. |
| `unit` | Cakupan unit kerja tertentu. | Operator unit administrasi. |
| `self` | Data yang dimiliki subjek sendiri. | Mahasiswa melihat KRS dan tagihannya sendiri. |
| `assigned` | Data berdasarkan penugasan atau relasi aktif. | Dosen melihat kelas yang diampu. |

### 7.1 Contoh penetapan role berscope

Contoh konseptual: seorang operator akademik ditetapkan dengan role `academic_operator` dan scope fakultas `FAC-001`. Ia dapat menjalankan permission `student:read` dan `academic.krs:approve`, tetapi hanya terhadap data yang berada dalam cakupan fakultas tersebut.

### 7.2 Contoh aturan ABAC

* Dosen dapat menginput nilai hanya jika dosen tersebut tercatat sebagai pengampu kelas terkait.
* Mahasiswa hanya dapat membaca data akademik dengan `student_id` yang terhubung ke identitasnya.
* Operator hanya dapat mengakses data unit yang menjadi tanggung jawabnya.
* Perubahan nilai dapat dibatasi oleh periode input nilai dan status publikasi.
* Verifikasi pembayaran dapat mensyaratkan status transaksi tertentu dan pemisahan tugas dari pembuat transaksi.

---

## 8. Rancangan Struktur Data Authorization

Struktur berikut merupakan rancangan konseptual untuk basis data authorization. Nama kolom dan tipe data final perlu disesuaikan saat penyusunan migration PostgreSQL.

| Tabel | Tujuan | Kolom inti yang disarankan |
| :--- | :--- | :--- |
| `users` | Referensi subjek dari Auth Service atau identity provider. | `id`, `external_subject_id`, `status` *(hindari menyimpan profil domain lengkap)*. |
| `roles` | Katalog role. | `id`, `code`, `name`, `description`, `is_system`, `created_at`. |
| `permissions` | Katalog permission. | `id`, `code`, `resource`, `action`, `description`. |
| `role_permissions` | Relasi role dan permission. | `role_id`, `permission_id`. |
| `user_roles` | Penetapan role kepada subjek. | `user_id`, `role_id`, `scope_type`, `scope_id`, `valid_from`, `valid_until`. |
| `user_permission_overrides` | Pengecualian akses khusus bila benar-benar diperlukan. | `user_id`, `permission_id`, `effect`, `scope_type`, `scope_id`, `valid_from`, `valid_until`. |
| `service_clients` | Identitas client antarlayanan atau integrasi. | `client_id`, `client_type`, `status`, `audience`, `credential_reference`. |

> **Catatan desain:** Tabel `users` dapat diganti atau dilengkapi dengan subject reference yang mengacu ke UUID/subjek dari Auth Service. Authorization Service tidak perlu menjadi penyimpanan profil lengkap mahasiswa, dosen, atau pegawai.

### 8.1 Aturan integritas dan keamanan data

* Gunakan unique constraint untuk `roles.code` dan `permissions.code`.
* Gunakan foreign key dan unique constraint pada pasangan `role_id` / `permission_id`.
* Pastikan satu penetapan role memiliki scope dan masa berlaku yang jelas.
* Simpan riwayat perubahan role, permission, scope, dan client.
* Rahasia client disimpan dalam bentuk hash atau secret manager; jangan menyimpan secret mentah.
* Terapkan transaksi database untuk perubahan role dan permission yang saling terkait.

---

## 9. Integrasi dengan Auth Service, Kong, dan Microservices

| Komponen | Tanggung jawab yang disarankan |
| :--- | :--- |
| **Auth Service** | Autentikasi, penerbitan token, validasi identitas, dan siklus hidup token. Tidak menjadi repositori profil domain lengkap. |
| **Authorization component/service** | Mengelola katalog role, permission, assignment, scope, dan evaluasi kebijakan bila diperlukan. |
| **Kong API Gateway** | Validasi token dan kontrol akses awal di perimeter API; meneruskan identitas dan konteks request secara aman. |
| **Domain microservice** | Menegakkan permission dan aturan data/domain, termasuk ownership dan relasi penugasan. |
| **Audit/logging** | Mencatat perubahan akses dan keputusan penting dengan actor, target, waktu, serta correlation ID. |

### 9.1 Klaim token

Token sebaiknya memuat klaim minimum seperti `sub`, `iss`, `aud`, `exp`, dan informasi teknis lain yang diperlukan. Role atau scope dapat dimasukkan bila ukurannya terkendali dan strategi pencabutan akses sudah ditetapkan. Hindari menanamkan seluruh permission ke access token jika katalog besar atau perubahan akses harus berlaku cepat.

### 9.2 Contoh payload konseptual

```json
{
  "sub": "user-uuid",
  "iss": "https://auth.ummu.ac.id",
  "aud": ["siak-api"],
  "principal_type": "user",
  "roles": ["academic_operator"],
  "permissions": ["student:read", "academic.krs:approve"],
  "access_contexts": [
    {
      "role": "academic_operator",
      "scope": {"type": "faculty", "id": "FAC-001"},
      "permissions": ["student:read", "academic.krs:approve"]
    }
  ],
  "exp": 1794841200,
  "jti": "token-uuid"
}
```

---

## 10. Pengelolaan, Audit, dan Pencabutan Akses

* Pemberian role harus memiliki dasar penugasan atau persetujuan pejabat yang berwenang.
* Role dengan dampak tinggi, seperti `super_admin` dan `security_admin`, harus dibatasi jumlah pemegangnya dan ditinjau berkala.
* Perubahan role, permission, scope, masa berlaku, dan status service client harus dicatat dalam audit log.
* Akses sementara harus memiliki `valid_from` dan `valid_until` serta dicabut otomatis atau melalui proses terjadwal.
* Pencabutan akses perlu mempertimbangkan token yang masih aktif, cache authorization, dan mekanisme revocation/introspection.
* Audit log harus mencatat `actor`, `action`, `target`, `timestamp`, `result`, `reason`, dan `correlation_id` jika tersedia.

---

## 11. Tahapan Implementasi

| Tahap | Fokus pekerjaan | Hasil yang diharapkan |
| :--- | :--- | :--- |
| **Phase 1** | Katalog roles, permissions, `role_permissions`. | Fondasi RBAC dan seed data awal. |
| **Phase 2** | User-role assignment, scope, masa berlaku. | Pembatasan akses berdasarkan organisasi dan penugasan. |
| **Phase 3** | Service client dan service-to-service authorization. | Identitas serta izin antarlayanan yang terpisah. |
| **Phase 4** | ABAC, ownership checks, dan permission override terbatas. | Aturan akses yang mengikuti konteks bisnis. |
| **Phase 5** | Audit, cache, revocation, monitoring, dan evaluasi kebijakan. | Kontrol akses yang dapat diawasi dan dioperasikan. |

---

## 12. Risiko dan Mitigasi

| Risiko | Dampak | Mitigasi |
| :--- | :--- | :--- |
| **Role terlalu luas** | Pengguna memperoleh akses melebihi tanggung jawabnya. | Least privilege, pemisahan tugas, dan review role berkala. |
| **Permission tidak konsisten** | Perbedaan penamaan dan perilaku antarservice. | Katalog permission terpusat dan standar penamaan. |
| **Hanya mengandalkan API Gateway** | Akses internal dapat melewati pemeriksaan domain. | Enforcement wajib pada microservice pemilik resource. |
| **Scope tidak diterapkan** | Data lintas fakultas/unit dapat terbuka. | Scope validation dan pengujian akses lintas unit. |
| **Token/cache usang** | Akses yang dicabut masih dapat digunakan sementara. | Token berumur pendek, revocation/introspection, dan invalidasi cache. |
| **Service account terlalu kuat** | Kompromi satu service berdampak luas. | Client identity terpisah, permission minimum, rotasi secret, dan pembatasan audience. |

---

## 13. Hal yang Perlu Divalidasi

Sebelum rancangan ini ditetapkan sebagai standar produksi, beberapa hal berikut perlu dikonfirmasi bersama pemilik proses bisnis dan pengelola sistem:

1. Daftar unit organisasi resmi UMMU beserta hierarki fakultas, program studi, dan unit kerja.
2. Nomenklatur jabatan dan pembagian tugas aktual pada BAAK, keuangan, penerimaan, dan unit terkait.
3. Alur persetujuan KRS, input/publikasi nilai, pembayaran, refund, serta dokumen administrasi.
4. Data owner untuk setiap domain dan microservice.
5. Kebutuhan akses integrasi eksternal, termasuk LPPK dan sistem lain yang terhubung.
6. Kebijakan retensi data, audit, dan persetujuan akses khusus.

---

## 14. Kesimpulan

Rancangan authorization SIAK Modern UMMU mengusulkan penggunaan **Hybrid RBAC + ABAC** dengan **scope** sebagai pembatas akses data. Role mengelompokkan tanggung jawab, permission mendefinisikan tindakan, sedangkan scope dan atribut memastikan tindakan hanya berlaku pada data serta kondisi yang sah. Pengelolaan katalog dapat dilakukan secara terpusat, sementara penegakan aturan tetap menjadi tanggung jawab microservice pemilik domain. Pendekatan ini mendukung pengembangan layanan secara bertahap tanpa menjadikan Auth Service sebagai penyimpan seluruh identitas domain.

---

## Lampiran A — Standar Penamaan

| Elemen | Format | Contoh |
| :--- | :--- | :--- |
| **Role code** | `snake_case` | `academic_operator` |
| **Permission code** | `resource:action` | `academic.grade:update` |
| **Scope type** | `snake_case` | `faculty`, `department`, `self`, `assigned` |
| **Service client** | nama layanan yang unik | `siak-legacy-api` |
| **Audit action** | kata kerja + objek | `role.assign`, `permission.revoke` |

---

## Lampiran B — Definisi Istilah

| Istilah | Definisi |
| :--- | :--- |
| **Authentication** | Proses memverifikasi identitas pengguna atau service. |
| **Authorization** | Proses menentukan apakah subjek boleh melakukan tindakan tertentu. |
| **RBAC** | Model kontrol akses yang memberikan permission melalui role. |
| **ABAC** | Model kontrol akses yang mempertimbangkan atribut subjek, resource, tindakan, dan konteks. |
| **Scope** | Batas cakupan data atau organisasi untuk suatu penetapan akses. |
| **Service account** | Identitas non-manusia yang digunakan oleh aplikasi atau microservice. |
| **Least privilege** | Prinsip pemberian hak minimum yang diperlukan untuk menjalankan tugas. |