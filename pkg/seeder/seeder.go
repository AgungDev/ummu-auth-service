package seeder

import (
	"auth_service/internal/domain"
	"auth_service/pkg"
	loggerpkg "auth_service/pkg/logger"
	"context"
	"gorm.io/gorm"
)

// SeedDummyData inserts initial dummy data if the database is empty.
func SeedDummyData(db *gorm.DB) error {
	ctx := context.WithValue(context.Background(), loggerpkg.ContextKeyLogSource, "startup")
	db = db.WithContext(ctx)
	passwordHash, err := pkg.HashPassword("Password123!")
	if err != nil {
		return err
	}
	adminPasswordHash, err := pkg.HashPassword("admin")
	if err != nil {
		return err
	}

	users := []domain.User{
		{Username: "admin", Email: "admin@example.com", PasswordHash: passwordHash, Status: "active"},
		{Username: "arjun0nur", Email: "arjun0nur@gmail.com", PasswordHash: adminPasswordHash, Status: "active"},
		{Username: "student", Email: "student@example.com", PasswordHash: passwordHash, Status: "active"},
		{Username: "lecturer", Email: "lecturer@example.com", PasswordHash: passwordHash, Status: "active"},
		{Username: "finance", Email: "finance@example.com", PasswordHash: passwordHash, Status: "active"},
		{Username: "operator", Email: "operator@example.com", PasswordHash: passwordHash, Status: "active"},
	}
	for i := range users {
		var existing domain.User
		if err := db.Where("username = ?", users[i].Username).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&users[i]).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			users[i] = existing
		}
	}

	var existingClient domain.Client
	if err := db.Where("name = ?", "SIAK Legacy API").First(&existingClient).Error; err == gorm.ErrRecordNotFound {
		secretHash, err := pkg.HashPassword("secret123")
		if err != nil {
			return err
		}
		client := &domain.Client{
			Name:             "SIAK Legacy API",
			ClientSecretHash: secretHash,
			IsConfidential:   true,
			Status:           "active",
			Grants:           []domain.ClientGrant{{GrantType: "client_credentials"}},
			Scopes:           []domain.ClientScope{{Scope: "student:read"}},
			Audiences:        []domain.ClientAudience{{Audience: "siak-legacy-api"}},
		}
		if err := db.Create(client).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if err := db.Where("client_id = ? AND audience = ?", existingClient.ID, "student-service").
			Delete(&domain.ClientAudience{}).Error; err != nil {
			return err
		}
		audience := domain.ClientAudience{ClientID: existingClient.ID, Audience: "siak-legacy-api"}
		if err := db.Where(&audience).FirstOrCreate(&audience).Error; err != nil {
			return err
		}
	}

	roleDefinitions := []domain.Role{
		{Code: "super_admin", Name: "Super Administrator", Description: "Mengelola kebijakan platform lintas domain"},
		{Code: "academic_operator", Name: "Operator Akademik", Description: "Menjalankan operasional akademik sesuai unit"},
		{Code: "lecturer", Name: "Dosen", Description: "Mengakses proses akademik untuk kelas dan penugasan"},
		{Code: "student", Name: "Mahasiswa", Description: "Mengakses informasi akademik milik sendiri"},
		{Code: "finance_operator", Name: "Operator Keuangan", Description: "Mengelola tagihan dan transaksi keuangan"},
	}
	roleByCode := make(map[string]*domain.Role)
	for i := range roleDefinitions {
		role := &roleDefinitions[i]
		var existing domain.Role
		if err := db.Where("code = ?", role.Code).First(&existing).Error; err == nil {
			roleByCode[role.Code] = &existing
			continue
		} else if err != gorm.ErrRecordNotFound {
			return err
		}
		if err := db.Create(role).Error; err != nil {
			return err
		}
		roleByCode[role.Code] = role
	}

	permissionDefinitions := []domain.Permission{
		{Code: "student:read", Name: "Lihat mahasiswa", Description: "Melihat data mahasiswa sesuai scope"},
		{Code: "student:create", Name: "Buat mahasiswa", Description: "Membuat data mahasiswa"},
		{Code: "student:update", Name: "Ubah mahasiswa", Description: "Memperbarui data mahasiswa"},
		{Code: "academic.krs:read", Name: "Lihat KRS", Description: "Melihat Kartu Rencana Studi"},
		{Code: "academic.krs:create", Name: "Isi KRS", Description: "Mengajukan atau mengisi KRS"},
		{Code: "academic.krs:approve", Name: "Setujui KRS", Description: "Menyetujui KRS sesuai kewenangan"},
		{Code: "academic.grade:read", Name: "Lihat nilai", Description: "Melihat nilai akademik"},
		{Code: "academic.grade:create", Name: "Input nilai", Description: "Menginput nilai untuk kelas penugasan"},
		{Code: "academic.grade:update", Name: "Ubah nilai", Description: "Memperbarui nilai dalam periode yang diizinkan"},
		{Code: "finance.billing:read", Name: "Lihat tagihan", Description: "Melihat tagihan keuangan"},
		{Code: "finance.transaction:read", Name: "Lihat transaksi", Description: "Melihat transaksi keuangan"},
		{Code: "finance.transaction:verify", Name: "Verifikasi transaksi", Description: "Memverifikasi transaksi pembayaran"},
		{Code: "system.role:manage", Name: "Kelola role", Description: "Mengelola katalog role"},
		{Code: "system.role:read", Name: "Lihat role", Description: "Melihat katalog role"},
		{Code: "system.permission:manage", Name: "Kelola permission", Description: "Mengelola katalog permission"},
		{Code: "system.permission:read", Name: "Lihat permission", Description: "Melihat katalog permission"},
		{Code: "system.client:manage", Name: "Kelola client", Description: "Mengelola client OAuth"},
		{Code: "system.access:manage", Name: "Kelola akses", Description: "Mengelola assignment permission langsung"},
		{Code: "system.audit:read", Name: "Lihat audit", Description: "Membaca audit log"},
		{Code: "user:read", Name: "Lihat pengguna", Description: "Melihat akun pengguna"},
		{Code: "user:create", Name: "Buat pengguna", Description: "Membuat akun pengguna"},
		{Code: "user:update", Name: "Ubah pengguna", Description: "Memperbarui akun pengguna"},
		{Code: "user:delete", Name: "Hapus pengguna", Description: "Menonaktifkan atau menghapus akun pengguna"},
		{Code: "user:assign_role", Name: "Tetapkan role", Description: "Menetapkan role kepada pengguna"},
		{Code: "user:revoke_role", Name: "Cabut role", Description: "Mencabut role dari pengguna"},
		{Code: "profile.admin:read", Name: "Lihat profil pengguna", Description: "Melihat data profil minimum untuk keperluan administrasi"},
		{Code: "profile.admin:update", Name: "Ubah profil pengguna", Description: "Memperbarui profil pengguna sesuai kewenangan"},
	}
	permissionByCode := make(map[string]*domain.Permission)
	for i := range permissionDefinitions {
		permission := &permissionDefinitions[i]
		var existing domain.Permission
		if err := db.Where("code = ?", permission.Code).First(&existing).Error; err == nil {
			permissionByCode[permission.Code] = &existing
			continue
		} else if err != gorm.ErrRecordNotFound {
			return err
		}
		if err := db.Create(permission).Error; err != nil {
			return err
		}
		permissionByCode[permission.Code] = permission
	}

	rolePermissions := map[string][]string{
		"super_admin":       {"system.role:manage", "system.role:read", "system.permission:manage", "system.permission:read", "system.client:manage", "system.access:manage", "system.audit:read", "user:read", "user:create", "user:update", "user:delete", "user:assign_role", "user:revoke_role", "profile.admin:read", "profile.admin:update"},
		"academic_operator": {"student:read", "student:create", "student:update", "academic.krs:read", "academic.krs:approve", "academic.grade:read"},
		"lecturer":          {"student:read", "academic.krs:read", "academic.grade:read", "academic.grade:create", "academic.grade:update"},
		"student":           {"student:read", "academic.krs:read", "academic.krs:create", "academic.grade:read", "finance.billing:read", "finance.transaction:read"},
		"finance_operator":  {"finance.billing:read", "finance.transaction:read", "finance.transaction:verify"},
	}
	for roleCode, permissionCodes := range rolePermissions {
		role := roleByCode[roleCode]
		for _, permissionCode := range permissionCodes {
			permission := permissionByCode[permissionCode]
			if err := db.Exec(
				"INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
				role.ID, permission.ID,
			).Error; err != nil {
				return err
			}
		}
	}

	userRoles := map[string]struct {
		role      string
		scopeType string
		scopeID   string
	}{
		"admin":     {role: "super_admin", scopeType: "global"},
		"arjun0nur": {role: "super_admin", scopeType: "global"},
		"student":   {role: "student", scopeType: "self"},
		"lecturer":  {role: "lecturer", scopeType: "none"},
		"finance":   {role: "finance_operator", scopeType: "none"},
		"operator":  {role: "academic_operator", scopeType: "none"},
	}
	for username, assignment := range userRoles {
		var user domain.User
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			return err
		}
		scopeID := assignment.scopeID
		if assignment.scopeType == "self" {
			scopeID = user.ID.String()
		}
		if err := db.Exec(
			`INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
			VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING`,
			user.ID, roleByCode[assignment.role].ID, assignment.scopeType, scopeID,
		).Error; err != nil {
			return err
		}
	}

	return nil
}
