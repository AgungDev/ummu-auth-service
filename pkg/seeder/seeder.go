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
	var count int64
	passwordHash, err := pkg.HashPassword("Password123!")
	if err != nil {
		return err
	}

	users := []domain.User{
		{Username: "admin", Email: "admin@example.com", PasswordHash: passwordHash, FullName: "Admin Sistem", Status: "active"},
		{Username: "student", Email: "student@example.com", PasswordHash: passwordHash, FullName: "Alya Mahasiswa", Status: "active"},
		{Username: "lecturer", Email: "lecturer@example.com", PasswordHash: passwordHash, FullName: "Dr. Rahman Dosen", Status: "active"},
		{Username: "finance", Email: "finance@example.com", PasswordHash: passwordHash, FullName: "Budi Keuangan", Status: "active"},
		{Username: "operator", Email: "operator@example.com", PasswordHash: passwordHash, FullName: "Siti Operator Akademik", Status: "active"},
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

	if err := db.Model(&domain.Client{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		secretHash, err := pkg.HashPassword("secret123")
		if err != nil {
			return err
		}
		client := &domain.Client{
			ClientID:         "default-client",
			Name:             "Default Client",
			ClientSecretHash: secretHash,
			IsConfidential:   true,
		}
		if err := db.Create(client).Error; err != nil {
			return err
		}

		// insert normalized redirect URIs
		redirect := &domain.ClientRedirectURI{
			ClientID:    client.ID,
			RedirectURI: "http://localhost/callback",
		}
		if err := db.Create(redirect).Error; err != nil {
			return err
		}

		// insert grants
		grants := []string{"password", "refresh_token"}
		for _, g := range grants {
			cg := &domain.ClientGrant{
				ClientID:  client.ID,
				GrantType: g,
			}
			if err := db.Create(cg).Error; err != nil {
				return err
			}
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
		{Code: "student.read", Name: "Lihat mahasiswa", Description: "Melihat data mahasiswa sesuai scope"},
		{Code: "student.create", Name: "Buat mahasiswa", Description: "Membuat data mahasiswa"},
		{Code: "student.update", Name: "Ubah mahasiswa", Description: "Memperbarui data mahasiswa"},
		{Code: "academic.krs.read", Name: "Lihat KRS", Description: "Melihat Kartu Rencana Studi"},
		{Code: "academic.krs.create", Name: "Isi KRS", Description: "Mengajukan atau mengisi KRS"},
		{Code: "academic.krs.approve", Name: "Setujui KRS", Description: "Menyetujui KRS sesuai kewenangan"},
		{Code: "academic.grade.read", Name: "Lihat nilai", Description: "Melihat nilai akademik"},
		{Code: "academic.grade.create", Name: "Input nilai", Description: "Menginput nilai untuk kelas penugasan"},
		{Code: "academic.grade.update", Name: "Ubah nilai", Description: "Memperbarui nilai dalam periode yang diizinkan"},
		{Code: "finance.billing.read", Name: "Lihat tagihan", Description: "Melihat tagihan keuangan"},
		{Code: "finance.transaction.read", Name: "Lihat transaksi", Description: "Melihat transaksi keuangan"},
		{Code: "finance.transaction.verify", Name: "Verifikasi transaksi", Description: "Memverifikasi transaksi pembayaran"},
		{Code: "system.role.manage", Name: "Kelola role", Description: "Mengelola katalog role"},
		{Code: "system.permission.manage", Name: "Kelola permission", Description: "Mengelola katalog permission"},
		{Code: "system.audit.read", Name: "Lihat audit", Description: "Membaca audit log"},
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
		"super_admin":       {"system.role.manage", "system.permission.manage", "system.audit.read"},
		"academic_operator": {"student.read", "student.create", "student.update", "academic.krs.read", "academic.krs.approve", "academic.grade.read"},
		"lecturer":          {"student.read", "academic.krs.read", "academic.grade.read", "academic.grade.create", "academic.grade.update"},
		"student":           {"student.read", "academic.krs.read", "academic.krs.create", "academic.grade.read", "finance.billing.read", "finance.transaction.read"},
		"finance_operator":  {"finance.billing.read", "finance.transaction.read", "finance.transaction.verify"},
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

	userRoles := map[string]string{
		"admin":    "super_admin",
		"student":  "student",
		"lecturer": "lecturer",
		"finance":  "finance_operator",
		"operator": "academic_operator",
	}
	for username, roleCode := range userRoles {
		var user domain.User
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
			return err
		}
		if err := db.Exec(
			"INSERT INTO user_roles (user_id, role_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
			user.ID, roleByCode[roleCode].ID,
		).Error; err != nil {
			return err
		}
	}

	return nil
}
