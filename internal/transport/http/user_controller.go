package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"auth_service/internal/domain"
	"auth_service/internal/domain/dto"
	middleware "auth_service/internal/transport/http/middleware_http"
	"auth_service/internal/usecases"
	loggerpkg "auth_service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type userController struct {
	userUsecase usecases.UserUsecaseInterface
	logger      loggerpkg.Logger
}

// NewUserController creates a new user controller instance
func NewUserController(userUsecase usecases.UserUsecaseInterface, logger loggerpkg.Logger) *userController {
	return &userController{userUsecase: userUsecase, logger: logger}
}

// Route registers user-related routes
func (c *userController) Route(rg *gin.RouterGroup) {
	userGroup := rg.Group("/users")
	{
		userGroup.POST("", middleware.RequirePermission("user:create"), c.Create)
		userGroup.GET("/:id", middleware.RequirePermission("user:read"), c.GetByID)
		userGroup.GET("", middleware.RequirePermission("user:read"), c.GetAll)
		userGroup.PUT("/:id", middleware.RequirePermission("user:update"), c.Update)
		userGroup.DELETE("/:id", middleware.RequirePermission("user:delete"), c.Delete)
		userGroup.POST("/:id/roles", middleware.RequirePermission("user:assign_role"), c.AssignRoles)
		userGroup.PUT("/:id/roles", middleware.RequirePermission("user:assign_role"), c.ReplaceRoles)
		userGroup.GET("/:id/roles", middleware.RequirePermission("user:read"), c.GetRoles)
		userGroup.DELETE("/:id/roles/:roleId", middleware.RequirePermission("user:revoke_role"), c.RemoveRole)
		userGroup.POST("/:id/permissions", middleware.RequirePermission("system.access:manage"), c.AssignPermissions)
		userGroup.PUT("/:id/permissions", middleware.RequirePermission("system.access:manage"), c.ReplacePermissions)
		userGroup.GET("/:id/permissions", middleware.RequirePermission("user:read"), c.GetPermissions)
		userGroup.DELETE("/:id/permissions/:permissionId", middleware.RequirePermission("system.access:manage"), c.RemovePermission)
	}
}

// GetByID retrieves a user by ID
func (c *userController) GetByID(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "user get request received")
	userID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := c.userUsecase.GetByID(ctx.Request.Context(), userID)
	if err != nil || user == nil {
		c.logger.Warn(correlationID, "", "anonymous", "user not found")
		JSONError(ctx, http.StatusNotFound, "user not found")
		return
	}

	resp := dto.UserResponse{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Status:   user.Status,
	}

	JSONOK(ctx, "user retrieved", resp)
}

// GetAll retrieves all users
func (c *userController) GetAll(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "user list request received")
	users, err := c.userUsecase.GetAll(ctx.Request.Context())
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "user list failed: "+err.Error())
		JSONError(ctx, http.StatusInternalServerError, "failed to retrieve users")
		return
	}

	responses := make([]dto.UserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, dto.UserResponse{
			ID:       user.ID.String(),
			Username: user.Username,
			Email:    user.Email,
			Status:   user.Status,
		})
	}

	JSONOK(ctx, "users retrieved", responses)
}

// Update updates user information
func (c *userController) Update(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "user update request received")
	userID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid user id")
		return
	}

	var req dto.UserUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "user update validation failed: "+err.Error())
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.userUsecase.Update(ctx.Request.Context(), userID, req); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "user update failed: "+err.Error())
		JSONError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	JSONOK(ctx, "user updated", nil)
}

// Delete removes a user
func (c *userController) Delete(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "user delete request received")
	userID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := c.userUsecase.Delete(ctx.Request.Context(), userID); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "user delete failed: "+err.Error())
		JSONError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	JSONOK(ctx, "user deleted", nil)
}

func (c *userController) Create(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "user create request received")

	var req dto.UserRegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "user create validation failed: "+err.Error())
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	user, err := c.userUsecase.Register(ctx.Request.Context(), req)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "user create failed: "+err.Error())
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	resp := dto.UserResponse{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Status:   user.Status,
	}

	JSONCreated(ctx, "user created", resp)
}

func (c *userController) AssignRoles(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "user assign roles request received")

	userID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid user id")
		return
	}

	var req dto.RoleAssignmentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "assign roles validation failed: "+err.Error())
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	assignments, err := parseRoleAssignments(userID, req.RoleIDs, req.Assignments)
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.userUsecase.AssignRoleAssignments(ctx.Request.Context(), userID, assignments); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "assign roles failed: "+err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			JSONError(ctx, http.StatusNotFound, "user or role not found")
			return
		}
		JSONError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	JSONOK(ctx, "roles assigned", nil)
}

func (c *userController) ReplaceRoles(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	userID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid user id")
		return
	}

	var req dto.RoleReplacementRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if (req.RoleIDs == nil) == (req.Assignments == nil) {
		JSONError(ctx, http.StatusBadRequest, "provide either role_ids or assignments")
		return
	}
	var assignments []domain.RoleAssignmentInput
	if req.RoleIDs != nil {
		if len(*req.RoleIDs) > 0 {
			assignments, err = parseRoleAssignments(userID, *req.RoleIDs, nil)
		}
	} else {
		if len(*req.Assignments) > 0 {
			assignments, err = parseRoleAssignments(userID, nil, *req.Assignments)
		}
	}
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.userUsecase.ReplaceRoleAssignments(ctx.Request.Context(), userID, assignments); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "replace user roles failed: "+err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			JSONError(ctx, http.StatusNotFound, "user or role not found")
			return
		}
		JSONError(ctx, http.StatusInternalServerError, "failed to replace user roles")
		return
	}
	JSONOK(ctx, "roles replaced", nil)
}

func (c *userController) GetRoles(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "user roles request received")

	userID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid user id")
		return
	}

	assignments, err := c.userUsecase.GetRoleAssignmentsByUserID(ctx.Request.Context(), userID)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "get user roles failed: "+err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			JSONError(ctx, http.StatusNotFound, "user not found")
			return
		}
		JSONError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]dto.UserRoleResponse, 0, len(assignments))
	for _, assignment := range assignments {
		resp = append(resp, dto.UserRoleResponse{
			ID:          assignment.Role.ID.String(),
			Code:        assignment.Role.Code,
			Name:        assignment.Role.Name,
			Description: assignment.Role.Description,
			ScopeType:   assignment.ScopeType,
			ScopeID:     assignment.ScopeID,
		})
	}

	JSONOK(ctx, "user roles retrieved", resp)
}

func (c *userController) RemoveRole(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "user remove role request received")

	userID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid user id")
		return
	}

	roleID, err := uuid.Parse(ctx.Param("roleId"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid role id")
		return
	}

	if err := c.userUsecase.RemoveRole(ctx.Request.Context(), userID, roleID); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "remove role failed: "+err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			JSONError(ctx, http.StatusNotFound, "user role assignment not found")
			return
		}
		JSONError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	JSONOK(ctx, "role removed", nil)
}

func (c *userController) AssignPermissions(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "user assign permissions request received")

	userID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid user id")
		return
	}

	var req dto.PermissionAssignmentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "assign permissions validation failed: "+err.Error())
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	permissionIDs := make([]uuid.UUID, 0, len(req.PermissionIDs))
	for _, permissionID := range req.PermissionIDs {
		parsed, err := uuid.Parse(permissionID)
		if err != nil {
			JSONError(ctx, http.StatusBadRequest, "invalid permission id")
			return
		}
		permissionIDs = append(permissionIDs, parsed)
	}

	if err := c.userUsecase.AssignPermissions(ctx.Request.Context(), userID, permissionIDs); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "assign permissions failed: "+err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			JSONError(ctx, http.StatusNotFound, "user or permission not found")
			return
		}
		JSONError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	JSONOK(ctx, "permissions assigned", nil)
}

func (c *userController) ReplacePermissions(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	userID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid user id")
		return
	}

	var req dto.PermissionReplacementRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	permissionIDs, err := parseUUIDs(*req.PermissionIDs, "permission")
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.userUsecase.ReplacePermissions(ctx.Request.Context(), userID, permissionIDs); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "replace user permissions failed: "+err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			JSONError(ctx, http.StatusNotFound, "user or permission not found")
			return
		}
		JSONError(ctx, http.StatusInternalServerError, "failed to replace user permissions")
		return
	}
	JSONOK(ctx, "permissions replaced", nil)
}

func (c *userController) GetPermissions(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "user permissions request received")

	userID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid user id")
		return
	}

	permissions, err := c.userUsecase.GetPermissions(ctx.Request.Context(), userID)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "get user permissions failed: "+err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			JSONError(ctx, http.StatusNotFound, "user not found")
			return
		}
		JSONError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]dto.PermissionResponse, 0, len(permissions))
	for _, permission := range permissions {
		resp = append(resp, dto.PermissionResponse{
			ID:          permission.ID.String(),
			Code:        permission.Code,
			Name:        permission.Name,
			Description: permission.Description,
		})
	}

	JSONOK(ctx, "user permissions retrieved", resp)
}

func (c *userController) RemovePermission(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "user remove permission request received")

	userID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid user id")
		return
	}

	permissionID, err := uuid.Parse(ctx.Param("permissionId"))
	if err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid permission id")
		return
	}

	if err := c.userUsecase.RemovePermission(ctx.Request.Context(), userID, permissionID); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "remove permission failed: "+err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			JSONError(ctx, http.StatusNotFound, "user permission assignment not found")
			return
		}
		JSONError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	JSONOK(ctx, "permission removed", nil)
}

func parseUUIDs(values []string, resource string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("invalid %s id: %q", resource, value)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func parseRoleAssignments(userID uuid.UUID, roleIDs []string, inputs []dto.RoleScopeAssignment) ([]domain.RoleAssignmentInput, error) {
	if len(roleIDs) > 0 && len(inputs) > 0 {
		return nil, errors.New("provide role_ids or assignments, not both")
	}
	assignments := make([]domain.RoleAssignmentInput, 0, len(roleIDs)+len(inputs))
	for _, roleID := range roleIDs {
		inputs = append(inputs, dto.RoleScopeAssignment{RoleID: roleID, ScopeType: "none"})
	}
	if len(inputs) == 0 {
		return nil, errors.New("at least one role assignment is required")
	}
	for _, input := range inputs {
		roleID, err := uuid.Parse(input.RoleID)
		if err != nil {
			return nil, errors.New("invalid role id")
		}
		scopeType := strings.TrimSpace(input.ScopeType)
		if scopeType == "" {
			scopeType = "none"
		}
		scopeID := strings.TrimSpace(input.ScopeID)
		switch scopeType {
		case "none", "global":
			if scopeID != "" {
				return nil, fmt.Errorf("%s scope must not include scope_id", scopeType)
			}
		case "self":
			if scopeID != "" && scopeID != userID.String() {
				return nil, errors.New("self scope_id must match the user")
			}
			scopeID = userID.String()
		case "faculty", "department", "unit", "assigned":
			if scopeID == "" {
				return nil, fmt.Errorf("%s scope requires scope_id", scopeType)
			}
		default:
			return nil, fmt.Errorf("unsupported scope_type %q", scopeType)
		}
		assignments = append(assignments, domain.RoleAssignmentInput{
			RoleID:    roleID,
			ScopeType: scopeType,
			ScopeID:   scopeID,
		})
	}
	return assignments, nil
}
