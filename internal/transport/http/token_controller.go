package http

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"auth_service/internal/domain"
	"auth_service/internal/domain/dto"
	"auth_service/internal/security"
	middleware "auth_service/internal/transport/http/middleware_http"
	"auth_service/internal/usecases"
	"auth_service/pkg"
	loggerpkg "auth_service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenControllerConfig struct {
	Issuer                string
	AllowedAudiences      []string
	AccessTokenTTLMinutes int
}

type tokenController struct {
	clientUsecase usecases.ClientUsecaseInterface
	security      security.SecurityService
	config        TokenControllerConfig
	logger        loggerpkg.Logger
}

var errInvalidScopes = errors.New("invalid scope request")

func NewTokenController(
	clientUsecase usecases.ClientUsecaseInterface,
	securityService security.SecurityService,
	config TokenControllerConfig,
	logger loggerpkg.Logger,
) *tokenController {
	return &tokenController{
		clientUsecase: clientUsecase,
		security:      securityService,
		config:        config,
		logger:        logger,
	}
}

func (c *tokenController) Route(rg *gin.RouterGroup) {
	rg.POST("/oauth/token", c.GetToken)
}

func (c *tokenController) JWKS(ctx *gin.Context) {
	document, err := c.security.PublicJWKS()
	if err != nil {
		c.logger.Error(middleware.GetCorrelationID(ctx), "", "anonymous", "JWKS generation failed: "+err.Error())
		JSONError(ctx, http.StatusInternalServerError, "unable to publish signing keys")
		return
	}
	ctx.Header("Cache-Control", "public, max-age=300")
	ctx.JSON(http.StatusOK, document)
}

func (c *tokenController) GetToken(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	var req dto.TokenRequest
	if err := ctx.ShouldBind(&req); err != nil {
		JSONError(ctx, http.StatusBadRequest, "invalid token request")
		return
	}
	if req.GrantType != "client_credentials" {
		c.logger.Warn(correlationID, "", "anonymous", "unsupported OAuth grant_type: "+req.GrantType)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}

	clientID, clientSecret, ok := clientCredentials(ctx, req.ClientID, req.ClientSecret)
	if !ok {
		ctx.Header("WWW-Authenticate", `Basic realm="oauth/token"`)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	clientUUID, err := uuid.Parse(clientID)
	if err != nil {
		ctx.Header("WWW-Authenticate", `Basic realm="oauth/token"`)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	client, err := c.clientUsecase.GetByID(ctx.Request.Context(), clientUUID)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "client lookup failed: "+err.Error())
		JSONError(ctx, http.StatusInternalServerError, "unable to authenticate client")
		return
	}
	if client == nil || client.Status != "active" || !client.IsConfidential ||
		!c.security.CheckPasswordHash(clientSecret, client.ClientSecretHash) {
		c.logger.Warn(correlationID, "", "anonymous", "invalid OAuth client credentials")
		ctx.Header("WWW-Authenticate", `Basic realm="oauth/token"`)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	if !hasGrant(client.Grants, "client_credentials") {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "unauthorized_client"})
		return
	}
	if !contains(c.config.AllowedAudiences, req.Audience) || !hasAudience(client.Audiences, req.Audience) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid_target"})
		return
	}

	scopes, err := requestedClientScopes(client.Scopes, req.Scope)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid_scope"})
		return
	}
	accessContexts := make([]pkg.AccessContext, 0, len(client.AccessContexts))
	for _, context := range client.AccessContexts {
		var scopeID *string
		if context.ScopeID != "" {
			id := context.ScopeID
			scopeID = &id
		}
		accessContexts = append(accessContexts, pkg.AccessContext{
			Scope: pkg.AccessScope{Type: context.ScopeType, ID: scopeID},
		})
	}
	token, err := c.security.GenerateToken(pkg.Claims{
		PrincipalType:  "client",
		ClientID:       client.ID.String(),
		Scope:          strings.Join(scopes, " "),
		AccessContexts: accessContexts,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   c.config.Issuer,
			Subject:  client.ID.String(),
			Audience: jwt.ClaimStrings{req.Audience},
		},
	}, c.config.AccessTokenTTLMinutes)
	if err != nil {
		c.logger.Error(correlationID, "", client.ID.String(), "client access token generation failed: "+err.Error())
		JSONError(ctx, http.StatusInternalServerError, "token generation failed")
		return
	}

	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")
	ctx.JSON(http.StatusOK, dto.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   c.config.AccessTokenTTLMinutes * 60,
		Scope:       strings.Join(scopes, " "),
	})
}

func clientCredentials(ctx *gin.Context, bodyID, bodySecret string) (string, string, bool) {
	if header := ctx.GetHeader("Authorization"); header != "" {
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Basic") {
			return "", "", false
		}
		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return "", "", false
		}
		clientID, clientSecret, found := strings.Cut(string(decoded), ":")
		if !found || clientID == "" || clientSecret == "" {
			return "", "", false
		}
		clientID, err = url.QueryUnescape(clientID)
		if err != nil {
			return "", "", false
		}
		clientSecret, err = url.QueryUnescape(clientSecret)
		if err != nil {
			return "", "", false
		}
		return clientID, clientSecret, true
	}
	return bodyID, bodySecret, bodyID != "" && bodySecret != ""
}

func requestedClientScopes(granted []domain.ClientScope, requested string) ([]string, error) {
	allowed := make(map[string]struct{}, len(granted))
	for _, item := range granted {
		allowed[item.Scope] = struct{}{}
	}
	scopes := strings.Fields(requested)
	if len(scopes) == 0 {
		return nil, errInvalidScopes
	}
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		if _, ok := allowed[scope]; !ok {
			return nil, errInvalidScopes
		}
		if _, duplicate := seen[scope]; duplicate {
			return nil, errInvalidScopes
		}
		seen[scope] = struct{}{}
	}
	sort.Strings(scopes)
	return scopes, nil
}

func hasGrant(grants []domain.ClientGrant, requested string) bool {
	for _, grant := range grants {
		if grant.GrantType == requested {
			return true
		}
	}
	return false
}

func hasAudience(audiences []domain.ClientAudience, requested string) bool {
	for _, audience := range audiences {
		if audience.Audience == requested {
			return true
		}
	}
	return false
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
