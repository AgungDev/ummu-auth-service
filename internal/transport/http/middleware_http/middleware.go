package middleware_http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"auth_service/internal/security"
	"auth_service/pkg"
	loggerpkg "auth_service/pkg/logger"
	"github.com/gin-gonic/gin"
)

const ContextKeyCorrelationID = "correlation_id"

func RequestLoggerMiddleware(log loggerpkg.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		correlationID := ctx.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = log.NewCorrelationID()
		}
		ctx.Set(ContextKeyCorrelationID, correlationID)
		ctx.Writer.Header().Set("X-Correlation-ID", correlationID)
		ctx.Request = ctx.Request.WithContext(
			context.WithValue(ctx.Request.Context(), ContextKeyCorrelationID, correlationID),
		)

		start := time.Now()
		ctx.Next()
		duration := time.Since(start)

		log.Info(
			correlationID,
			"",
			"",
			"request completed",
			slog.String("method", ctx.Request.Method),
			slog.String("path", ctx.FullPath()),
			slog.Int("status", ctx.Writer.Status()),
			slog.String("duration", duration.String()),
		)
	}
}

func GetCorrelationID(ctx *gin.Context) string {
	if cid, ok := ctx.Get(ContextKeyCorrelationID); ok {
		if str, ok := cid.(string); ok {
			return str
		}
		return fmt.Sprint(cid)
	}
	return ctx.GetHeader("X-Correlation-ID")
}

func AuthMiddleware(securityService security.SecurityService, logger loggerpkg.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		correlationID := GetCorrelationID(ctx)
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn(correlationID, "", "anonymous", "missing authorization header")
			ctx.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "authorization header is required", "error": "authorization header is required"})
			return
		}

		parts := strings.Fields(authHeader)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			logger.Warn(correlationID, "", "anonymous", "invalid authorization header format")
			ctx.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "authorization header must be Bearer token", "error": "invalid authorization format"})
			return
		}

		claims, err := securityService.ValidateJWT(parts[1])
		if err != nil {
			logger.Warn(correlationID, "", "anonymous", "invalid bearer token: "+err.Error())
			ctx.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "invalid or expired token", "error": "invalid or expired token"})
			return
		}

		ctx.Set("jwt_claims", claims)
		ctx.Set("user_subject", claims.Subject)
		ctx.Set("principal_type", claims.PrincipalType)
		ctx.Next()
	}
}

func RequirePrincipalType(principalType string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, ok := claimsFromContext(ctx)
		if !ok || claims.PrincipalType != principalType {
			ctx.JSON(http.StatusForbidden, gin.H{"success": false, "message": "principal is not allowed"})
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

func RequireAudience(audience string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, ok := claimsFromContext(ctx)
		if !ok {
			ctx.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "verified token claims are missing"})
			ctx.Abort()
			return
		}
		for _, current := range claims.Audience {
			if current == audience {
				ctx.Next()
				return
			}
		}
		ctx.JSON(http.StatusForbidden, gin.H{"success": false, "message": "token audience is not allowed"})
		ctx.Abort()
	}
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claims, ok := claimsFromContext(ctx)
		if !ok || claims.PrincipalType != "user" {
			ctx.JSON(http.StatusForbidden, gin.H{"success": false, "message": "user principal is required"})
			ctx.Abort()
			return
		}
		for _, accessContext := range claims.AccessContexts {
			if accessContext.Scope.Type != "global" {
				continue
			}
			for _, granted := range accessContext.Permissions {
				if granted == permission {
					ctx.Next()
					return
				}
			}
		}
		ctx.JSON(http.StatusForbidden, gin.H{"success": false, "message": "permission denied"})
		ctx.Abort()
	}
}

func claimsFromContext(ctx *gin.Context) (*pkg.Claims, bool) {
	raw, ok := ctx.Get("jwt_claims")
	if !ok {
		return nil, false
	}
	claims, ok := raw.(*pkg.Claims)
	return claims, ok
}
