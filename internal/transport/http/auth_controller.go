package http

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"

	"auth_service/internal/domain/dto"
	"auth_service/internal/security"
	"auth_service/internal/services"
	middleware "auth_service/internal/transport/http/middleware_http"
	"auth_service/internal/usecases"
	"auth_service/pkg"
	loggerpkg "auth_service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthController struct {
	authUsecase   usecases.AuthUsecaseInterface
	security      security.SecurityService
	googleOIDC    services.GoogleOIDCService
	logger        loggerpkg.Logger
	secureCookies bool
}

var _ = []RegisterRequest{}

func NewAuthController(
	authUsecase usecases.AuthUsecaseInterface,
	securityService security.SecurityService,
	googleOIDC services.GoogleOIDCService,
	logger loggerpkg.Logger,
	secureCookies bool,
) *AuthController {
	return &AuthController{
		authUsecase:   authUsecase,
		security:      securityService,
		googleOIDC:    googleOIDC,
		logger:        logger,
		secureCookies: secureCookies,
	}
}

func (c *AuthController) Route(rg *gin.RouterGroup) {
	authGroup := rg.Group("/auth")
	authGroup.POST("/register", c.Register)
	authGroup.POST("/login", c.Login)
	authGroup.POST("/refresh", c.Refresh)
	authGroup.POST("/introspect", c.Introspect)

	protected := authGroup.Group("")
	protected.Use(middleware.AuthMiddleware(c.security, c.logger))
	protected.GET("/me", c.Me)
	protected.POST("/check-permission", c.CheckPermission)
	protected.GET("/permissions", c.GetPermissions)
	protected.POST("/logout", c.Logout)
	protected.POST("/authorize", c.Authorize)
}

func (c *AuthController) GoogleRoute(rg *gin.Engine) {
	rg.GET("/auth/google", c.GoogleLogin)
	rg.GET("/auth/google/callback", c.GoogleCallback)
}

func (c *AuthController) GoogleLogin(ctx *gin.Context) {
	if c.googleOIDC == nil {
		JSONError(ctx, http.StatusServiceUnavailable, "Google login is not configured")
		return
	}

	state, err := newOAuthState()
	if err != nil {
		c.logger.Error(middleware.GetCorrelationID(ctx), "", "anonymous", "failed to generate Google OAuth state: "+err.Error())
		JSONError(ctx, http.StatusInternalServerError, "unable to start Google login")
		return
	}

	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie("google_oauth_state", state, 300, "/auth/google", "", c.secureCookies, true)
	ctx.Redirect(http.StatusFound, c.googleOIDC.AuthorizationURL(state))
}

func (c *AuthController) GoogleCallback(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	state, cookieErr := ctx.Cookie("google_oauth_state")
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie("google_oauth_state", "", -1, "/auth/google", "", c.secureCookies, true)
	if cookieErr != nil || state == "" || ctx.Query("state") == "" ||
		subtle.ConstantTimeCompare([]byte(state), []byte(ctx.Query("state"))) != 1 {
		c.logger.Warn(correlationID, "", "anonymous", "Google login rejected because OAuth state did not match")
		JSONError(ctx, http.StatusBadRequest, "invalid Google OAuth state")
		return
	}
	if ctx.Query("error") != "" {
		c.logger.Warn(correlationID, "", "anonymous", "Google login was not authorized")
		JSONError(ctx, http.StatusUnauthorized, "Google login was not authorized")
		return
	}
	code := ctx.Query("code")
	if code == "" {
		JSONError(ctx, http.StatusBadRequest, "missing Google authorization code")
		return
	}
	if c.googleOIDC == nil {
		JSONError(ctx, http.StatusServiceUnavailable, "Google login is not configured")
		return
	}

	identity, err := c.googleOIDC.VerifyCallback(ctx.Request.Context(), code, state)
	if err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "Google identity verification failed: "+err.Error())
		JSONError(ctx, http.StatusUnauthorized, "Google authentication failed")
		return
	}
	response, err := c.authUsecase.LoginWithGoogle(ctx.Request.Context(), identity)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "Google login failed: "+err.Error())
		JSONError(ctx, http.StatusUnauthorized, "Google authentication failed")
		return
	}
	JSONOK(ctx, "login success", response)
}

func newOAuthState() (string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate secure OAuth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body RegisterRequest true "Register request body"
// @Success 201 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/auth/register [post]
func (c *AuthController) Register(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "auth register request received")

	var req dto.UserRegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "auth register validation failed: "+err.Error())
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	user, err := c.authUsecase.Register(ctx.Request.Context(), req)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "auth register failed: "+err.Error())
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	resp := dto.UserResponse{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		FullName: user.FullName,
		Status:   user.Status,
	}

	JSONCreated(ctx, "user registered", resp)
}

// Login godoc
// @Summary Login user
// @Description Authenticate user and return JWT access token
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body LoginRequest true "Login request body"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "auth login request received")

	var req dto.UserLoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "auth login validation failed: "+err.Error())
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := c.authUsecase.Login(ctx.Request.Context(), req)
	if err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "auth login failed: "+err.Error())
		JSONError(ctx, http.StatusUnauthorized, err.Error())
		return
	}

	JSONOK(ctx, "login success", resp)
}

// Refresh godoc
// @Summary Refresh access token
// @Description Refresh JWT access token using refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (c *AuthController) Refresh(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "auth refresh request received")

	var req dto.RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "refresh validation failed: "+err.Error())
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := c.authUsecase.Refresh(ctx.Request.Context(), req)
	if err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "refresh failed: "+err.Error())
		JSONError(ctx, http.StatusUnauthorized, err.Error())
		return
	}

	JSONOK(ctx, "token refreshed", resp)
}

// Logout godoc
// @Summary Logout current user
// @Description Revoke current user's session and refresh token state
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} MessageResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/auth/logout [post]
func (c *AuthController) Logout(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "auth logout request received")

	claimsRaw, ok := ctx.Get("jwt_claims")
	if !ok {
		JSONError(ctx, http.StatusUnauthorized, "missing jwt claims")
		return
	}

	claims, ok := claimsRaw.(*pkg.Claims)
	if !ok {
		JSONError(ctx, http.StatusUnauthorized, "invalid jwt claims")
		return
	}

	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		JSONError(ctx, http.StatusUnauthorized, "invalid user in token")
		return
	}

	if err := c.authUsecase.Logout(ctx.Request.Context(), userID); err != nil {
		c.logger.Error(correlationID, "", "anonymous", "logout failed: "+err.Error())
		JSONError(ctx, http.StatusInternalServerError, "unable to logout")
		return
	}

	JSONOK(ctx, "logout success", nil)
}

// Me godoc
// @Summary Get current user profile
// @Description Retrieve profile information for authenticated user
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/auth/me [get]
func (c *AuthController) Me(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "auth me request received")

	claimsRaw, ok := ctx.Get("jwt_claims")
	if !ok {
		JSONError(ctx, http.StatusUnauthorized, "missing jwt claims")
		return
	}

	claims, ok := claimsRaw.(*pkg.Claims)
	if !ok {
		JSONError(ctx, http.StatusUnauthorized, "invalid jwt claims")
		return
	}

	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		JSONError(ctx, http.StatusUnauthorized, "invalid user in token")
		return
	}

	resp, err := c.authUsecase.GetUserProfile(ctx.Request.Context(), userID)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "failed to get user profile: "+err.Error())
		JSONError(ctx, http.StatusInternalServerError, "unable to retrieve user profile")
		return
	}
	if resp == nil {
		JSONError(ctx, http.StatusNotFound, "user not found")
		return
	}

	JSONOK(ctx, "user profile retrieved", resp)
}

// CheckPermission godoc
// @Summary Check user permission
// @Description Evaluate whether the authenticated user has a specific permission
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body PermissionRequest true "Permission check request"
// @Success 200 {object} PermissionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/auth/check-permission [post]
func (c *AuthController) CheckPermission(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "auth check-permission request received")

	var req dto.CheckPermissionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "check permission validation failed: "+err.Error())
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	claimsRaw, ok := ctx.Get("jwt_claims")
	if !ok {
		JSONError(ctx, http.StatusUnauthorized, "missing jwt claims")
		return
	}

	claims, ok := claimsRaw.(*pkg.Claims)
	if !ok {
		JSONError(ctx, http.StatusUnauthorized, "invalid jwt claims")
		return
	}

	allowed, err := c.authUsecase.CheckPermission(ctx.Request.Context(), claims.Sub, req.Permission)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "check permission failed: "+err.Error())
		JSONError(ctx, http.StatusInternalServerError, "unable to evaluate permission")
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"allowed": allowed})
}

// Introspect godoc
// @Summary Verify access token
// @Description Inspect a token and return its active state, roles and permissions
// @Tags Auth
// @Accept json
// @Produce json
// @Param payload body IntrospectRequest true "Token introspection request"
// @Success 200 {object} IntrospectResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/introspect [post]
func (c *AuthController) Introspect(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "token introspect request received")

	var req dto.IntrospectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "introspect validation failed: "+err.Error())
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := c.authUsecase.Introspect(ctx.Request.Context(), req.Token)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "introspect failed: "+err.Error())
		JSONError(ctx, http.StatusUnauthorized, err.Error())
		return
	}

	JSONOK(ctx, "token introspected", resp)
}

// GetPermissions godoc
// @Summary Get authenticated user permissions
// @Description Return list of permissions for the authenticated user
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} PermissionsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/auth/permissions [get]
func (c *AuthController) GetPermissions(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "auth permissions request received")

	claimsRaw, ok := ctx.Get("jwt_claims")
	if !ok {
		JSONError(ctx, http.StatusUnauthorized, "missing jwt claims")
		return
	}

	claims, ok := claimsRaw.(*pkg.Claims)
	if !ok {
		JSONError(ctx, http.StatusUnauthorized, "invalid jwt claims")
		return
	}

	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		JSONError(ctx, http.StatusUnauthorized, "invalid user in token")
		return
	}

	permissions, err := c.authUsecase.GetPermissionsByUserID(ctx.Request.Context(), userID)
	if err != nil {
		c.logger.Error(correlationID, "", "anonymous", "get permissions failed: "+err.Error())
		JSONError(ctx, http.StatusInternalServerError, "unable to get user permissions")
		return
	}

	JSONOK(ctx, "user permissions retrieved", dto.PermissionsResponse{Permissions: permissions})
}

// Authorize godoc
// @Summary Authorize a user permission
// @Description Check whether a specific user is authorized for a permission
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body AuthorizeRequest true "Authorize request body"
// @Success 200 {object} AuthorizeResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/auth/authorize [post]
func (c *AuthController) Authorize(ctx *gin.Context) {
	correlationID := middleware.GetCorrelationID(ctx)
	c.logger.Info(correlationID, "", "anonymous", "authorize request received")

	var req dto.AuthorizeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.Warn(correlationID, "", "anonymous", "authorize validation failed: "+err.Error())
		JSONError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	claimsRaw, ok := ctx.Get("jwt_claims")
	if !ok {
		JSONError(ctx, http.StatusUnauthorized, "missing jwt claims")
		return
	}

	claims, ok := claimsRaw.(*pkg.Claims)
	if !ok {
		JSONError(ctx, http.StatusUnauthorized, "invalid jwt claims")
		return
	}

	resp, err := c.authUsecase.Authorize(ctx.Request.Context(), claims.Sub, claims.Roles, req)
	if err != nil {
		switch err {
		case usecases.ErrUserIDMismatch:
			JSONError(ctx, http.StatusForbidden, err.Error())
			return
		case usecases.ErrNotAuthorized:
			JSONError(ctx, http.StatusForbidden, err.Error())
			return
		default:
			c.logger.Error(correlationID, "", "anonymous", "authorize failed: "+err.Error())
			JSONError(ctx, http.StatusInternalServerError, "authorization failed")
			return
		}
	}

	JSONOK(ctx, "authorization result", resp)
}
