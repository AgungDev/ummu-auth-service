package usecases

import (
	"context"
	"fmt"

	"auth_service/internal/domain"
	"auth_service/internal/domain/dto"
	"auth_service/internal/repositories"
	"github.com/google/uuid"
)

type roleUsecase struct {
	roleRepo repositories.RoleRepositoryInterface
}

// NewRoleUsecase creates a new role usecase instance
func NewRoleUsecase(roleRepo repositories.RoleRepositoryInterface) RoleUsecaseInterface {
	return &roleUsecase{roleRepo: roleRepo}
}

// Create creates a new role
func (uc *roleUsecase) Create(ctx context.Context, req dto.RoleRequest) (*domain.Role, error) {
	for _, lookup := range []struct {
		value string
		find  func(context.Context, string) (*domain.Role, error)
		label string
	}{
		{req.Code, uc.roleRepo.FindByCode, "role code"},
		{req.Name, uc.roleRepo.FindByName, "role name"},
	} {
		existingRole, err := lookup.find(ctx, lookup.value)
		if err != nil {
			return nil, err
		}
		if existingRole != nil {
			return nil, fmt.Errorf("%w: %s already exists", domain.ErrConflict, lookup.label)
		}
	}

	role := &domain.Role{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := uc.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

// GetByID retrieves a role by ID
func (uc *roleUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	return uc.roleRepo.FindByID(ctx, id)
}

// GetByName retrieves a role by name
func (uc *roleUsecase) GetByName(ctx context.Context, name string) (*domain.Role, error) {
	return uc.roleRepo.FindByName(ctx, name)
}

// GetAll retrieves all roles
func (uc *roleUsecase) GetAll(ctx context.Context) ([]domain.Role, error) {
	return uc.roleRepo.FindAll(ctx)
}

// Update updates role information
func (uc *roleUsecase) Update(ctx context.Context, id uuid.UUID, req dto.RoleRequest) error {
	role, err := uc.roleRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return domain.ErrNotFound
	}

	for _, lookup := range []struct {
		value string
		find  func(context.Context, string) (*domain.Role, error)
		label string
	}{
		{req.Code, uc.roleRepo.FindByCode, "role code"},
		{req.Name, uc.roleRepo.FindByName, "role name"},
	} {
		existingRole, err := lookup.find(ctx, lookup.value)
		if err != nil {
			return err
		}
		if existingRole != nil && existingRole.ID != id {
			return fmt.Errorf("%w: %s already exists", domain.ErrConflict, lookup.label)
		}
	}

	role = &domain.Role{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	}
	return uc.roleRepo.Update(ctx, id, role)
}

// Delete removes a role
func (uc *roleUsecase) Delete(ctx context.Context, id uuid.UUID) error {
	return uc.roleRepo.Delete(ctx, id)
}

// AssignPermissions assigns permissions to a role
func (uc *roleUsecase) AssignPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	return uc.roleRepo.AssignPermissions(ctx, roleID, permissionIDs)
}

func (uc *roleUsecase) ReplacePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	return uc.roleRepo.ReplacePermissions(ctx, roleID, permissionIDs)
}

// GetPermissionsByRoleID retrieves permissions assigned to a role
func (uc *roleUsecase) GetPermissionsByRoleID(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	return uc.roleRepo.GetPermissionsByRoleID(ctx, roleID)
}

// RemovePermission removes a permission from a role
func (uc *roleUsecase) RemovePermission(ctx context.Context, roleID uuid.UUID, permissionID uuid.UUID) error {
	return uc.roleRepo.RemovePermission(ctx, roleID, permissionID)
}
