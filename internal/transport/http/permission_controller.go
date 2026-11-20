package http

import (
	"errors"

	"auth_service/internal/domain"
	"auth_service/internal/domain/dto"
	middleware "auth_service/internal/transport/http/middleware_http"
	"auth_service/internal/usecases"
	loggerpkg "auth_service/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type permissionController struct {
	permissionUsecase usecases.PermissionUsecaseInterface
	logger            loggerpkg.Logger
}

// NewPermissionController creates a new permission controller instance
func NewPermissionController(permissionUsecase usecases.PermissionUsecaseInterface, logger loggerpkg.Logger) *permissionController {
	return &permissionController{permissionUsecase: permissionUsecase, logger: logger}
}

// Route registers permission-related routes
func (c *permissionController) Route(rg *gin.RouterGroup) {
	permissionGroup := rg.Group("/permissions")
	{
		permissionGroup.POST("", middleware.RequirePermission("system.permission:manage"), c.Create)
		permissionGroup.GET("/:id", middleware.RequirePermission("system.permission:read"), c.GetByID)
		permissionGroup.GET("", middleware.RequirePermission("system.permission:read"), c.GetAll)
		permissionGroup.PUT("/:id", middleware.RequirePermission("system.permission:manage"), c.Update)
		permissionGroup.DELETE("/:id", middleware.RequirePermission("system.permission:manage"), c.Delete)
	}
}

// Create creates a new permission
func (c *permissionController) Create(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "permission create request received")

	var req dto.PermissionRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "permission create validation failed: "+err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	permission, err := c.permissionUsecase.Create(ctx.Request.Context(), req)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "permission create failed: "+err.Error())
		if errors.Is(err, domain.ErrConflict) {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp := dto.PermissionResponse{
		ID:          permission.ID.String(),
		Code:        permission.Code,
		Name:        permission.Name,
		Description: permission.Description,
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Permission created successfully",
		"data":    resp,
	})
}

// GetByID retrieves a permission by ID
func (c *permissionController) GetByID(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "permission get request received")
	permissionID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission id"})
		return
	}

	permission, err := c.permissionUsecase.GetByID(ctx.Request.Context(), permissionID)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "permission lookup failed: "+err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve permission"})
		return
	}
	if permission == nil {
		c.logger.Warn(correlationID, "", "anonymous", "permission not found")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "permission not found"})
		return
	}

	resp := dto.PermissionResponse{
		ID:          permission.ID.String(),
		Code:        permission.Code,
		Name:        permission.Name,
		Description: permission.Description,
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Permission retrieved successfully",
		"data":    resp,
	})
}

// GetAll retrieves all permissions
func (c *permissionController) GetAll(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "permission list request received")
	permissions, err := c.permissionUsecase.GetAll(ctx.Request.Context())
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "permission list failed: "+err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]dto.PermissionResponse, 0, len(permissions))
	for _, permission := range permissions {
		responses = append(responses, dto.PermissionResponse{
			ID:          permission.ID.String(),
			Code:        permission.Code,
			Name:        permission.Name,
			Description: permission.Description,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Permissions retrieved successfully",
		"data":    responses,
	})
}

// Update updates a permission
func (c *permissionController) Update(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "permission update request received")
	permissionID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission id"})
		return
	}

	var req dto.PermissionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.permissionUsecase.Update(ctx.Request.Context(), permissionID, req); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "permission update failed: "+err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "permission not found"})
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Permission updated successfully",
	})
}

// Delete removes a permission
func (c *permissionController) Delete(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "permission delete request received")
	permissionID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission id"})
		return
	}

	if err := c.permissionUsecase.Delete(ctx.Request.Context(), permissionID); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "permission delete failed: "+err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "permission not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Permission deleted successfully",
	})
}
