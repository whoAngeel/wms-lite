package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type Middleware struct {
	service *Service
	logger  zerolog.Logger
}

func NewMiddleware(service *Service, logger zerolog.Logger) *Middleware {
	moduleLogger := logger.With().Str("middleware", "auth").Logger()
	return &Middleware{
		service: service,
		logger:  moduleLogger,
	}
}

func (m *Middleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		var token string
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			cookieToken, err := c.Cookie("access_token")
			if err == nil {
				token = cookieToken
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header or cookie required"})
			c.Abort()
			return
		}

		claims, err := m.service.ValidateAccessToken(token)
		if err != nil {
			m.logger.Warn().Err(err).Str("ip", c.ClientIP()).Msg("Invalid token attempt")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func (m *Middleware) RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleInterface, exists := c.Get(("role"))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		userRole := roleInterface.(string)

		for _, alloallowedRole := range allowedRoles {
			if userRole == alloallowedRole {
				c.Next()
				return
			}
		}

		m.logger.Warn().
			Str("user_role", userRole).
			Strs("allowed_roles", allowedRoles).
			Str("path", c.Request.URL.Path).
			Msg("Insufficient permissions")
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}
