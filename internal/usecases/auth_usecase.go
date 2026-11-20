package usecases

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"auth_service/internal/domain"
	"auth_service/internal/domain/dto"
	"auth_service/internal/security"
	"auth_service/pkg"
	loggerpkg "auth_service/pkg/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrUserIDMismatch = errors.New("user id does not match token subject")
	ErrNotAuthorized  = errors.New("not authorized")
)

type authUsecase struct {
	userUsecase       UserUsecaseInterface
	clientUsecase     ClientUsecaseInterface
	roleUsecase       RoleUsecaseInterface
	permissionUsecase PermissionUsecaseInterface
	tokenUsecase      TokenUsecaseInterface
	security          security.SecurityService
	logger            loggerpkg.Logger
	appName           string
	accessTokenTTLMin int
	refreshTokenTTLHr int
}

func NewAuthUsecase(
	userUsecase UserUsecaseInterface,
	clientUsecase ClientUsecaseInterface,
	roleUsecase RoleUsecaseInterface,
	permissionUsecase PermissionUsecaseInterface,
	tokenUsecase TokenUsecaseInterface,
	securityService security.SecurityService,
	logger loggerpkg.Logger,
	appName string,
	accessTokenTTLMinutes int,
	refreshTokenTTLHours int,
) AuthUsecaseInterface {
	return &authUsecase{
		userUsecase:       userUsecase,
		clientUsecase:     clientUsecase,
		roleUsecase:       roleUsecase,
		permissionUsecase: permissionUsecase,
		tokenUsecase:      tokenUsecase,
		security:          securityService,
		logger:            logger,
		appName:           appName,
		accessTokenTTLMin: accessTokenTTLMinutes,
		refreshTokenTTLHr: refreshTokenTTLHours,
	}
}

func (uc *authUsecase) Register(ctx context.Context, req dto.UserRegisterRequest) (*domain.User, error) {
	return uc.userUsecase.Register(ctx, req)
}

func (uc *authUsecase) Login(ctx context.Context, req dto.UserLoginRequest) (*dto.AuthLoginResponse, error) {
	user, err := uc.userUsecase.Login(ctx, req)
	if err != nil {
		return nil, err
	}

	var clientID *uuid.UUID
	if req.ClientID != "" {
		clientUUID, err := uuid.Parse(req.ClientID)
		if err != nil {
			return nil, errors.New("invalid client")
		}
		client, err := uc.clientUsecase.GetByID(ctx, clientUUID)
		if err != nil {
			return nil, err
		}
		if client == nil {
			return nil, errors.New("invalid client")
		}
		clientID = &client.ID
	}

	return uc.createLoginResponse(ctx, user, clientID)
}

func (uc *authUsecase) LoginWithGoogle(ctx context.Context, identity domain.GoogleIdentity) (*dto.AuthLoginResponse, error) {
	user, err := uc.userUsecase.FindOrCreateGoogleUser(ctx, identity)
	if err != nil {
		return nil, err
	}
	return uc.createLoginResponse(ctx, user, nil)
}

func (uc *authUsecase) createLoginResponse(ctx context.Context, user *domain.User, clientID *uuid.UUID) (*dto.AuthLoginResponse, error) {
	roles, permissions, accessContexts, err := uc.profileTokenClaims(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	accessToken, err := uc.security.GenerateToken(pkg.Claims{
		PrincipalType:  "user",
		Roles:          roles,
		Permissions:    permissions,
		AccessContexts: accessContexts,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   uc.appName,
			Subject:  user.ID.String(),
			Audience: jwt.ClaimStrings{"auth-service", "profile-service"},
		},
	}, uc.accessTokenTTLMin)
	if err != nil {
		return nil, err
	}

	refreshToken := uuid.NewString()
	tokenRecord := &domain.Token{
		UserID:           user.ID,
		ClientID:         clientID,
		RefreshTokenHash: pkg.HashOpaqueToken(refreshToken),
		ExpiresAt:        time.Now().Add(time.Duration(uc.refreshTokenTTLHr) * time.Hour),
	}

	if err := uc.tokenUsecase.Create(ctx, tokenRecord); err != nil {
		return nil, err
	}

	return &dto.AuthLoginResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    uc.accessTokenTTLMin * 60,
		RefreshToken: refreshToken,
		User: dto.UserResponse{
			ID:       user.ID.String(),
			Username: user.Username,
			Email:    user.Email,
			Status:   user.Status,
		},
	}, nil
}

func (uc *authUsecase) profileTokenClaims(ctx context.Context, userID uuid.UUID) ([]string, []string, []pkg.AccessContext, error) {
	userRoles, err := uc.userUsecase.GetRoleAssignmentsByUserID(ctx, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load user roles for access token: %w", err)
	}
	rolesSet := make(map[string]struct{}, len(userRoles))
	accessContexts := make([]pkg.AccessContext, 0, len(userRoles))
	for _, assignment := range userRoles {
		rolesSet[assignment.Role.Code] = struct{}{}
		if assignment.ScopeType == "" || assignment.ScopeType == "none" {
			continue
		}
		var scopeID *string
		if assignment.ScopeID != "" {
			id := assignment.ScopeID
			scopeID = &id
		}
		rolePermissions, err := uc.roleUsecase.GetPermissionsByRoleID(ctx, assignment.Role.ID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("load permissions for role %s: %w", assignment.Role.Code, err)
		}
		contextPermissions := make([]string, 0, len(rolePermissions))
		for _, permission := range rolePermissions {
			contextPermissions = append(contextPermissions, permission.Code)
		}
		sort.Strings(contextPermissions)
		accessContexts = append(accessContexts, pkg.AccessContext{
			Role: assignment.Role.Code,
			Scope: pkg.AccessScope{
				Type: assignment.ScopeType,
				ID:   scopeID,
			},
			Permissions: contextPermissions,
		})
	}
	roles := make([]string, 0, len(rolesSet))
	for role := range rolesSet {
		roles = append(roles, role)
	}
	sort.Strings(roles)

	permissions, err := uc.permissionUsecase.GetByUserID(ctx, userID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load user permissions for access token: %w", err)
	}
	permissionCodes := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		if code := strings.TrimSpace(permission.Code); code != "" {
			permissionCodes = append(permissionCodes, code)
		}
	}
	sort.Strings(permissionCodes)
	return roles, permissionCodes, accessContexts, nil
}

func (uc *authUsecase) Refresh(ctx context.Context, req dto.RefreshTokenRequest) (*dto.TokenResponse, error) {
	oldTokenHash := pkg.HashOpaqueToken(req.RefreshToken)
	tokenRecord, err := uc.tokenUsecase.GetByRefreshTokenHash(ctx, oldTokenHash)
	if err != nil {
		return nil, err
	}
	if tokenRecord == nil {
		return nil, errors.New("invalid refresh token")
	}

	user, err := uc.userUsecase.GetByID(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid refresh token")
	}

	roles, permissions, accessContexts, err := uc.profileTokenClaims(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	accessToken, err := uc.security.GenerateToken(pkg.Claims{
		PrincipalType:  "user",
		Roles:          roles,
		Permissions:    permissions,
		AccessContexts: accessContexts,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   uc.appName,
			Subject:  user.ID.String(),
			Audience: jwt.ClaimStrings{"auth-service", "profile-service"},
		},
	}, uc.accessTokenTTLMin)
	if err != nil {
		return nil, err
	}

	newRefreshToken := uuid.NewString()
	if err := uc.tokenUsecase.Rotate(
		ctx,
		tokenRecord.ID,
		oldTokenHash,
		pkg.HashOpaqueToken(newRefreshToken),
		time.Now().Add(time.Duration(uc.refreshTokenTTLHr)*time.Hour),
	); err != nil {
		return nil, err
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    uc.accessTokenTTLMin * 60,
		RefreshToken: newRefreshToken,
	}, nil
}

func (uc *authUsecase) Introspect(ctx context.Context, token string) (*dto.IntrospectResponse, error) {
	claims, err := uc.security.ValidateJWT(token)
	if err != nil {
		return nil, err
	}

	accessContexts := make([]dto.AccessContextResponse, 0, len(claims.AccessContexts))
	for _, accessContext := range claims.AccessContexts {
		accessContexts = append(accessContexts, dto.AccessContextResponse{
			Role:        accessContext.Role,
			Permissions: accessContext.Permissions,
			Scope: dto.AccessScopeResponse{
				Type: accessContext.Scope.Type,
				ID:   accessContext.Scope.ID,
			},
		})
	}

	return &dto.IntrospectResponse{
		Active:         true,
		User:           map[string]string{"sub": claims.Subject, "iss": claims.Issuer},
		PrincipalType:  claims.PrincipalType,
		Subject:        claims.Subject,
		ClientID:       claims.ClientID,
		Scope:          claims.Scope,
		Roles:          claims.Roles,
		Permissions:    claims.Permissions,
		AccessContexts: accessContexts,
	}, nil
}

func (uc *authUsecase) Authorize(ctx context.Context, subject string, req dto.AuthorizeRequest) (*dto.AuthorizeResponse, error) {
	uc.logger.Info("", "", req.UserID, fmt.Sprintf("authorize request: user_id=%s, permission=%s", req.UserID, req.Permission))

	if req.UserID != subject {
		uc.logger.Warn("", "", req.UserID, fmt.Sprintf("user id mismatch: token subject=%s, request user_id=%s", subject, req.UserID))
		return &dto.AuthorizeResponse{
			Authorized: false,
			UserID:     req.UserID,
			Permission: req.Permission,
		}, ErrUserIDMismatch
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		uc.logger.Error("", "", req.UserID, fmt.Sprintf("failed to parse user id: %v", err))
		return &dto.AuthorizeResponse{
			Authorized: false,
			UserID:     req.UserID,
			Permission: req.Permission,
		}, err
	}

	permissions, err := uc.permissionUsecase.GetByUserID(ctx, userID)
	if err != nil {
		uc.logger.Error("", "", req.UserID, fmt.Sprintf("failed to get user permissions: %v", err))
		return &dto.AuthorizeResponse{
			Authorized: false,
			UserID:     req.UserID,
			Permission: req.Permission,
		}, err
	}

	uc.logger.Info("", "", req.UserID, fmt.Sprintf("user permissions retrieved: count=%d", len(permissions)))

	has := false
	for _, perm := range permissions {
		if perm.Code == req.Permission {
			has = true
			break
		}
	}

	if !has {
		uc.logger.Warn("", "", req.UserID, fmt.Sprintf("permission not found: %s", req.Permission))
		return &dto.AuthorizeResponse{
			Authorized: false,
			UserID:     req.UserID,
			Permission: req.Permission,
		}, ErrNotAuthorized
	}

	uc.logger.Info("", "", req.UserID, fmt.Sprintf("authorization granted: permission=%s", req.Permission))
	return &dto.AuthorizeResponse{
		Authorized: true,
		UserID:     req.UserID,
		Permission: req.Permission,
	}, nil
}

func (uc *authUsecase) GetPermissionsByUserID(ctx context.Context, userID uuid.UUID) ([]string, error) {
	uc.logger.Info("", "", userID.String(), "fetching user permissions from database")

	permissions, err := uc.permissionUsecase.GetByUserID(ctx, userID)
	if err != nil {
		uc.logger.Error("", "", userID.String(), fmt.Sprintf("failed to get permissions: %v", err))
		return nil, err
	}

	permissionCodes := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		permissionCodes = append(permissionCodes, permission.Code)
	}

	uc.logger.Info("", "", userID.String(), fmt.Sprintf("permissions retrieved: count=%d", len(permissionCodes)))
	return permissionCodes, nil
}

func (uc *authUsecase) Logout(ctx context.Context, userID uuid.UUID) error {
	return uc.tokenUsecase.DeleteByUserID(ctx, userID)
}

func (uc *authUsecase) GetUserProfile(ctx context.Context, userID uuid.UUID) (*dto.UserResponse, error) {
	user, err := uc.userUsecase.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}
	return &dto.UserResponse{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Status:   user.Status,
	}, nil
}

func (uc *authUsecase) CheckPermission(ctx context.Context, subject string, permission string) (bool, error) {
	userID, err := uuid.Parse(subject)
	if err != nil {
		return false, err
	}

	permissions, err := uc.permissionUsecase.GetByUserID(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, perm := range permissions {
		if perm.Code == permission {
			return true, nil
		}
	}

	return false, nil
}
