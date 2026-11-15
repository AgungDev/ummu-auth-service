package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"auth_service/internal/domain"
	"auth_service/internal/domain/dto"
	"auth_service/internal/repositories"
	"auth_service/internal/security"

	"github.com/google/uuid"
)

type userUsecase struct {
	userRepo repositories.UserRepositoryInterface
	security security.SecurityService
}

// NewUserUsecase creates a new user usecase instance
func NewUserUsecase(userRepo repositories.UserRepositoryInterface, securityService security.SecurityService) UserUsecaseInterface {
	return &userUsecase{userRepo: userRepo, security: securityService}
}

// Register handles user registration with validation and password hashing
func (uc *userUsecase) Register(ctx context.Context, req dto.UserRegisterRequest) (*domain.User, error) {
	// Check if username already exists
	existingUser, err := uc.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("username already exists")
	}

	// Check if email already exists
	existingEmail, err := uc.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existingEmail != nil {
		return nil, errors.New("email already exists")
	}

	// Hash password
	hashedPassword, err := uc.security.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// Create user
	user := &domain.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FullName:     req.FullName,
		Status:       "active",
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login validates user credentials
func (uc *userUsecase) Login(ctx context.Context, req dto.UserLoginRequest) (*domain.User, error) {
	// Find user by username
	user, err := uc.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}

	// Check if user exists
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	// Check if user is active
	if user.Status != "active" {
		return nil, errors.New("user is not active")
	}

	// Verify password
	if !uc.security.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

func (uc *userUsecase) FindOrCreateGoogleUser(ctx context.Context, identity domain.GoogleIdentity) (*domain.User, error) {
	if identity.Subject == "" || !identity.EmailVerified {
		return nil, errors.New("Google identity is missing a verified email")
	}
	address, err := mail.ParseAddress(identity.Email)
	if err != nil || address.Address != identity.Email || utf8.RuneCountInString(identity.Email) > 128 {
		return nil, errors.New("Google identity has an invalid email")
	}

	user, err := uc.userRepo.FindByGoogleSub(ctx, identity.Subject)
	if err != nil {
		return nil, err
	}
	if user != nil {
		if user.Status != "active" {
			return nil, errors.New("user is not active")
		}
		return user, nil
	}

	user, err = uc.userRepo.FindByEmail(ctx, identity.Email)
	if err != nil {
		return nil, err
	}
	if user != nil {
		if user.Status != "active" {
			return nil, errors.New("user is not active")
		}
		if user.GoogleSub != nil && *user.GoogleSub != "" {
			return nil, errors.New("account is linked to a different Google identity")
		}
		if err := uc.userRepo.SetGoogleSub(ctx, user.ID, identity.Subject); err != nil {
			linkedUser, lookupErr := uc.userRepo.FindByGoogleSub(ctx, identity.Subject)
			if lookupErr == nil && linkedUser != nil && linkedUser.ID == user.ID {
				return linkedUser, nil
			}
			return nil, err
		}
		user.GoogleSub = &identity.Subject
		return user, nil
	}

	usernameHash := sha256.Sum256([]byte(identity.Subject))
	randomPasswordHash, err := uc.security.HashPassword(uuid.NewString())
	if err != nil {
		return nil, fmt.Errorf("failed to hash generated password: %w", err)
	}
	fullName := strings.TrimSpace(identity.Name)
	if fullName == "" {
		fullName = identity.Email
	}
	if utf8.RuneCountInString(fullName) > 128 {
		runes := []rune(fullName)
		fullName = string(runes[:128])
	}
	user = &domain.User{
		Username:     "google_" + hex.EncodeToString(usernameHash[:12]),
		Email:        identity.Email,
		PasswordHash: randomPasswordHash,
		FullName:     fullName,
		Status:       "active",
		GoogleSub:    &identity.Subject,
	}
	if err := uc.userRepo.Create(ctx, user); err != nil {
		linkedUser, lookupErr := uc.userRepo.FindByGoogleSub(ctx, identity.Subject)
		if lookupErr == nil && linkedUser != nil {
			return linkedUser, nil
		}
		return nil, err
	}
	return user, nil
}

// GetByID retrieves a user by ID
func (uc *userUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return uc.userRepo.FindByID(ctx, id)
}

// GetByUsername retrieves a user by username
func (uc *userUsecase) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return uc.userRepo.FindByUsername(ctx, username)
}

// GetAll retrieves all users
func (uc *userUsecase) GetAll(ctx context.Context) ([]domain.User, error) {
	return uc.userRepo.FindAll(ctx)
}

// Update updates user information
func (uc *userUsecase) Update(ctx context.Context, id uuid.UUID, req dto.UserUpdateRequest) error {
	user := &domain.User{
		FullName: req.FullName,
		Email:    req.Email,
	}
	return uc.userRepo.Update(ctx, id, user)
}

// Delete removes a user
func (uc *userUsecase) Delete(ctx context.Context, id uuid.UUID) error {
	return uc.userRepo.Delete(ctx, id)
}

// AssignRoles assigns roles to a user
func (uc *userUsecase) AssignRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	return uc.userRepo.AssignRoles(ctx, userID, roleIDs)
}

// GetRolesByUserID retrieves roles assigned to a user
func (uc *userUsecase) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	return uc.userRepo.GetRolesByUserID(ctx, userID)
}

// RemoveRole removes a role from a user
func (uc *userUsecase) RemoveRole(ctx context.Context, userID uuid.UUID, roleID uuid.UUID) error {
	return uc.userRepo.RemoveRole(ctx, userID, roleID)
}

// AssignPermissions assigns direct permissions to a user
func (uc *userUsecase) AssignPermissions(ctx context.Context, userID uuid.UUID, permissionIDs []uuid.UUID) error {
	return uc.userRepo.AssignPermissions(ctx, userID, permissionIDs)
}

// GetPermissions retrieves direct permissions assigned to a user
func (uc *userUsecase) GetPermissions(ctx context.Context, userID uuid.UUID) ([]domain.Permission, error) {
	return uc.userRepo.GetPermissions(ctx, userID)
}

// RemovePermission removes a direct permission from a user
func (uc *userUsecase) RemovePermission(ctx context.Context, userID uuid.UUID, permissionID uuid.UUID) error {
	return uc.userRepo.RemovePermission(ctx, userID, permissionID)
}
