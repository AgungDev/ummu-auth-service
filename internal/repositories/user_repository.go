package repositories

import (
	"auth_service/internal/domain"
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository instance
func NewUserRepository(db *gorm.DB) UserRepositoryInterface {
	return &userRepository{db: db}
}

// Create inserts a new user into the database
func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// FindByID finds a user by ID
func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindByUsername finds a user by username
func (r *userRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail finds a user by email
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByGoogleSub(ctx context.Context, subject string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("google_sub = ?", subject).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) SetGoogleSub(ctx context.Context, userID uuid.UUID, subject string) error {
	result := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ? AND (google_sub IS NULL OR google_sub = '')", userID).
		Update("google_sub", subject)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("Google identity is already linked to an account")
	}
	return nil
}

// FindAll retrieves all users
func (r *userRepository) FindAll(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	if err := r.db.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Update updates an existing user
func (r *userRepository) Update(ctx context.Context, id uuid.UUID, user *domain.User) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Updates(user).Error
}

// Delete deletes a user by ID
func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.User{}, id).Error
}

// AssignRoles assigns roles to a user
func (r *userRepository) AssignRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	assignments := make([]domain.RoleAssignmentInput, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		assignments = append(assignments, domain.RoleAssignmentInput{RoleID: roleID, ScopeType: "none"})
	}
	return r.AssignRoleAssignments(ctx, userID, assignments)
}

func (r *userRepository) ReplaceRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	assignments := make([]domain.RoleAssignmentInput, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		assignments = append(assignments, domain.RoleAssignmentInput{RoleID: roleID, ScopeType: "none"})
	}
	return r.ReplaceRoleAssignments(ctx, userID, assignments)
}

func (r *userRepository) AssignRoleAssignments(ctx context.Context, userID uuid.UUID, assignments []domain.RoleAssignmentInput) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureRecordExists(tx, &domain.User{}, userID); err != nil {
			return err
		}
		roleIDs := make([]uuid.UUID, 0, len(assignments))
		for _, assignment := range assignments {
			roleIDs = append(roleIDs, assignment.RoleID)
		}
		if err := ensureRecordsExist(tx, &domain.Role{}, roleIDs); err != nil {
			return err
		}
		for _, assignment := range assignments {
			if err := tx.Exec(
				`INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
				VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING`,
				userID,
				assignment.RoleID,
				assignment.ScopeType,
				assignment.ScopeID,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *userRepository) ReplaceRoleAssignments(ctx context.Context, userID uuid.UUID, assignments []domain.RoleAssignmentInput) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureRecordExists(tx, &domain.User{}, userID); err != nil {
			return err
		}
		roleIDs := make([]uuid.UUID, 0, len(assignments))
		for _, assignment := range assignments {
			roleIDs = append(roleIDs, assignment.RoleID)
		}
		if err := ensureRecordsExist(tx, &domain.Role{}, roleIDs); err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM user_roles WHERE user_id = ?", userID).Error; err != nil {
			return err
		}
		for _, assignment := range assignments {
			if err := tx.Exec(
				`INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
				VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING`,
				userID,
				assignment.RoleID,
				assignment.ScopeType,
				assignment.ScopeID,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetRolesByUserID retrieves roles assigned to a user
func (r *userRepository) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	if err := ensureRecordExists(r.db.WithContext(ctx), &domain.User{}, userID); err != nil {
		return nil, err
	}
	roles := make([]domain.Role, 0)
	err := r.db.WithContext(ctx).Raw(
		`SELECT DISTINCT r.*
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = ?
		ORDER BY r.code`,
		userID,
	).Scan(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *userRepository) GetRoleAssignmentsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.UserRoleAssignment, error) {
	if err := ensureRecordExists(r.db.WithContext(ctx), &domain.User{}, userID); err != nil {
		return nil, err
	}
	var rows []struct {
		ID          uuid.UUID
		Code        string
		Name        string
		Description string
		ScopeType   string
		ScopeID     string
	}
	err := r.db.WithContext(ctx).Raw(
		`SELECT r.id, r.code, r.name, r.description, ur.scope_type, ur.scope_id
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = ?
		ORDER BY r.code, ur.scope_type, ur.scope_id`,
		userID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	assignments := make([]domain.UserRoleAssignment, 0, len(rows))
	for _, row := range rows {
		assignments = append(assignments, domain.UserRoleAssignment{
			Role: domain.Role{
				ID:          row.ID,
				Code:        row.Code,
				Name:        row.Name,
				Description: row.Description,
			},
			ScopeType: row.ScopeType,
			ScopeID:   row.ScopeID,
		})
	}
	return assignments, nil
}

// RemoveRole removes a role assignment from a user
func (r *userRepository) RemoveRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) error {
	result := r.db.WithContext(ctx).Exec(
		"DELETE FROM user_roles WHERE user_id = ? AND role_id = ?",
		userID,
		roleID,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// AssignPermissions assigns direct permissions to a user
func (r *userRepository) AssignPermissions(ctx context.Context, userID uuid.UUID, permissionIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureRecordExists(tx, &domain.User{}, userID); err != nil {
			return err
		}
		if err := ensureRecordsExist(tx, &domain.Permission{}, permissionIDs); err != nil {
			return err
		}
		for _, permissionID := range permissionIDs {
			if err := tx.Exec(
				"INSERT INTO user_permissions (user_id, permission_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
				userID,
				permissionID,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *userRepository) ReplacePermissions(ctx context.Context, userID uuid.UUID, permissionIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureRecordExists(tx, &domain.User{}, userID); err != nil {
			return err
		}
		if err := ensureRecordsExist(tx, &domain.Permission{}, permissionIDs); err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM user_permissions WHERE user_id = ?", userID).Error; err != nil {
			return err
		}
		for _, permissionID := range permissionIDs {
			if err := tx.Exec(
				"INSERT INTO user_permissions (user_id, permission_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
				userID,
				permissionID,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetPermissions retrieves direct permissions assigned to a user
func (r *userRepository) GetPermissions(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error) {
	if err := ensureRecordExists(r.db.WithContext(ctx), &domain.User{}, userID); err != nil {
		return nil, err
	}
	permissions := make([]domain.Permission, 0)
	err := r.db.WithContext(ctx).Raw(
		`SELECT p.*
		FROM permissions p
		JOIN user_permissions up ON up.permission_id = p.id
		WHERE up.user_id = ?`,
		userID,
	).Scan(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

// RemovePermission removes a direct permission from a user
func (r *userRepository) RemovePermission(ctx context.Context, userID uuid.UUID, permissionID uuid.UUID) error {
	result := r.db.WithContext(ctx).Exec(
		"DELETE FROM user_permissions WHERE user_id = ? AND permission_id = ?",
		userID,
		permissionID,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
