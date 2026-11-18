-- Dummy seed data for Auth Service
-- Password for all dummy accounts: Password123!
-- Run after migrations/init.sql.

INSERT INTO users (id, username, email, password_hash, full_name, status) VALUES
('11111111-1111-1111-1111-111111111111', 'admin', 'admin@example.com', '$2a$10$hs7AlCQqfYSHtrQddAu3uuCzKy82liEz4Hbj9FgnMTbsek4iWFpuu', 'Admin Sistem', 'active'),
('22222222-2222-2222-2222-111111111111', 'student', 'student@example.com', '$2a$10$hs7AlCQqfYSHtrQddAu3uuCzKy82liEz4Hbj9FgnMTbsek4iWFpuu', 'Alya Mahasiswa', 'active'),
('33333333-3333-3333-3333-111111111111', 'lecturer', 'lecturer@example.com', '$2a$10$hs7AlCQqfYSHtrQddAu3uuCzKy82liEz4Hbj9FgnMTbsek4iWFpuu', 'Dr. Rahman Dosen', 'active'),
('44444444-4444-4444-4444-111111111111', 'finance', 'finance@example.com', '$2a$10$hs7AlCQqfYSHtrQddAu3uuCzKy82liEz4Hbj9FgnMTbsek4iWFpuu', 'Budi Keuangan', 'active'),
('55555555-5555-5555-5555-111111111111', 'operator', 'operator@example.com', '$2a$10$hs7AlCQqfYSHtrQddAu3uuCzKy82liEz4Hbj9FgnMTbsek4iWFpuu', 'Siti Operator Akademik', 'active')
ON CONFLICT (username) DO NOTHING;

INSERT INTO roles (id, code, name, description) VALUES
('10000000-0000-0000-0000-000000000001', 'super_admin', 'Super Administrator', 'Mengelola kebijakan platform lintas domain'),
('10000000-0000-0000-0000-000000000002', 'academic_operator', 'Operator Akademik', 'Menjalankan operasional akademik sesuai unit'),
('10000000-0000-0000-0000-000000000003', 'lecturer', 'Dosen', 'Mengakses proses akademik untuk kelas dan penugasan'),
('10000000-0000-0000-0000-000000000004', 'student', 'Mahasiswa', 'Mengakses informasi akademik milik sendiri'),
('10000000-0000-0000-0000-000000000005', 'finance_operator', 'Operator Keuangan', 'Mengelola tagihan dan transaksi keuangan')
ON CONFLICT (code) DO NOTHING;

INSERT INTO permissions (id, code, name, description) VALUES
('20000000-0000-0000-0000-000000000001', 'student.read', 'Lihat mahasiswa', 'Melihat data mahasiswa sesuai scope'),
('20000000-0000-0000-0000-000000000002', 'student.create', 'Buat mahasiswa', 'Membuat data mahasiswa'),
('20000000-0000-0000-0000-000000000003', 'student.update', 'Ubah mahasiswa', 'Memperbarui data mahasiswa'),
('20000000-0000-0000-0000-000000000004', 'academic.krs.read', 'Lihat KRS', 'Melihat Kartu Rencana Studi'),
('20000000-0000-0000-0000-000000000005', 'academic.krs.create', 'Isi KRS', 'Mengajukan atau mengisi KRS'),
('20000000-0000-0000-0000-000000000006', 'academic.krs.approve', 'Setujui KRS', 'Menyetujui KRS sesuai kewenangan'),
('20000000-0000-0000-0000-000000000007', 'academic.grade.read', 'Lihat nilai', 'Melihat nilai akademik'),
('20000000-0000-0000-0000-000000000008', 'academic.grade.create', 'Input nilai', 'Menginput nilai untuk kelas penugasan'),
('20000000-0000-0000-0000-000000000009', 'academic.grade.update', 'Ubah nilai', 'Memperbarui nilai dalam periode yang diizinkan'),
('20000000-0000-0000-0000-000000000010', 'finance.billing.read', 'Lihat tagihan', 'Melihat tagihan keuangan'),
('20000000-0000-0000-0000-000000000011', 'finance.transaction.read', 'Lihat transaksi', 'Melihat transaksi keuangan'),
('20000000-0000-0000-0000-000000000012', 'finance.transaction.verify', 'Verifikasi transaksi', 'Memverifikasi transaksi pembayaran'),
('20000000-0000-0000-0000-000000000013', 'system.role.manage', 'Kelola role', 'Mengelola katalog role'),
('20000000-0000-0000-0000-000000000014', 'system.permission.manage', 'Kelola permission', 'Mengelola katalog permission'),
('20000000-0000-0000-0000-000000000015', 'system.audit.read', 'Lihat audit', 'Membaca audit log')
ON CONFLICT (code) DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM (VALUES
  ('admin', 'super_admin'),
  ('student', 'student'),
  ('lecturer', 'lecturer'),
  ('finance', 'finance_operator'),
  ('operator', 'academic_operator')
) AS assignments(username, role_code)
JOIN users u ON u.username = assignments.username
JOIN roles r ON r.code = assignments.role_code
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM (VALUES
  ('super_admin', 'system.role.manage'),
  ('super_admin', 'system.permission.manage'),
  ('super_admin', 'system.audit.read'),
  ('academic_operator', 'student.read'),
  ('academic_operator', 'student.create'),
  ('academic_operator', 'student.update'),
  ('academic_operator', 'academic.krs.read'),
  ('academic_operator', 'academic.krs.approve'),
  ('academic_operator', 'academic.grade.read'),
  ('lecturer', 'student.read'),
  ('lecturer', 'academic.krs.read'),
  ('lecturer', 'academic.grade.read'),
  ('lecturer', 'academic.grade.create'),
  ('lecturer', 'academic.grade.update'),
  ('student', 'student.read'),
  ('student', 'academic.krs.read'),
  ('student', 'academic.krs.create'),
  ('student', 'academic.grade.read'),
  ('student', 'finance.billing.read'),
  ('student', 'finance.transaction.read'),
  ('finance_operator', 'finance.billing.read'),
  ('finance_operator', 'finance.transaction.read'),
  ('finance_operator', 'finance.transaction.verify')
) AS mappings(role_code, permission_code)
JOIN roles r ON r.code = mappings.role_code
JOIN permissions p ON p.code = mappings.permission_code
ON CONFLICT DO NOTHING;
