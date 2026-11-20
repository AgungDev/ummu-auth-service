package http

import (
	"errors"
	"net/http"

	"auth_service/internal/domain"
	"auth_service/internal/domain/dto"
	middleware "auth_service/internal/transport/http/middleware_http"
	"auth_service/internal/usecases"
	loggerpkg "auth_service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type clientController struct {
	clientUsecase usecases.ClientUsecaseInterface
	logger        loggerpkg.Logger
}

// NewClientController creates a new client controller instance
func NewClientController(clientUsecase usecases.ClientUsecaseInterface, logger loggerpkg.Logger) *clientController {
	return &clientController{clientUsecase: clientUsecase, logger: logger}
}

// Route registers client-related routes
func (c *clientController) Route(rg *gin.RouterGroup) {
	clientGroup := rg.Group("/clients")
	{
		clientGroup.POST("", middleware.RequirePermission("system.client:manage"), c.Create)
		clientGroup.GET("/:id", middleware.RequirePermission("system.client:manage"), c.GetByID)
		clientGroup.GET("", middleware.RequirePermission("system.client:manage"), c.GetAll)
		clientGroup.PUT("/:id", middleware.RequirePermission("system.client:manage"), c.Update)
		clientGroup.DELETE("/:id", middleware.RequirePermission("system.client:manage"), c.Delete)
	}
}

// Create creates a new client
func (c *clientController) Create(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "client create request received")
	var req dto.ClientRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "client create validation failed: "+err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := c.clientUsecase.Create(ctx.Request.Context(), req)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "client create failed: "+err.Error())
		if errors.Is(err, domain.ErrConflict) {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp := clientResponse(client)

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Client created successfully",
		"data":    resp,
	})
}

// GetByID retrieves a client by ID
func (c *clientController) GetByID(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "client get request received")
	clientID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
		return
	}

	client, err := c.clientUsecase.GetByID(ctx.Request.Context(), clientID)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "client lookup failed: "+err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve client"})
		return
	}
	if client == nil {
		c.logger.Warn(correlationID, "", "anonymous", "client not found")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
		return
	}

	resp := clientResponse(client)

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Client retrieved successfully",
		"data":    resp,
	})
}

// GetAll retrieves all clients
func (c *clientController) GetAll(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "client list request received")
	clients, err := c.clientUsecase.GetAll(ctx.Request.Context())
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "client list failed: "+err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responses := make([]dto.ClientResponse, 0, len(clients))
	for _, client := range clients {
		responses = append(responses, clientResponse(&client))
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Clients retrieved successfully",
		"data":    responses,
	})
}

// Update updates a client
func (c *clientController) Update(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "client update request received")
	clientID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
		return
	}

	var req dto.ClientRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.clientUsecase.Update(ctx.Request.Context(), clientID, req); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "client update failed: "+err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
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
		"message": "Client updated successfully",
	})
}

// Delete removes a client
func (c *clientController) Delete(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "client delete request received")
	clientID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
		return
	}

	if err := c.clientUsecase.Delete(ctx.Request.Context(), clientID); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "client delete failed: "+err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "client not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Client deleted successfully",
	})
}

func clientResponse(client *domain.Client) dto.ClientResponse {
	response := dto.ClientResponse{
		ID:             client.ID.String(),
		Name:           client.Name,
		IsConfidential: client.IsConfidential,
		Status:         client.Status,
		RedirectURIs:   make([]string, 0, len(client.RedirectURIs)),
		Grants:         make([]string, 0, len(client.Grants)),
		Scopes:         make([]string, 0, len(client.Scopes)),
		Audiences:      make([]string, 0, len(client.Audiences)),
		AccessContexts: make([]dto.ClientAccessContextRequest, 0, len(client.AccessContexts)),
	}
	for _, redirectURI := range client.RedirectURIs {
		response.RedirectURIs = append(response.RedirectURIs, redirectURI.RedirectURI)
	}
	for _, grant := range client.Grants {
		response.Grants = append(response.Grants, grant.GrantType)
	}
	for _, scope := range client.Scopes {
		response.Scopes = append(response.Scopes, scope.Scope)
	}
	for _, audience := range client.Audiences {
		response.Audiences = append(response.Audiences, audience.Audience)
	}
	for _, accessContext := range client.AccessContexts {
		response.AccessContexts = append(response.AccessContexts, dto.ClientAccessContextRequest{
			ScopeType: accessContext.ScopeType,
			ScopeID:   accessContext.ScopeID,
		})
	}
	return response
}
