package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type Handler struct {
	service *Service
	logger  zerolog.Logger
}

func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	moduleLogger := logger.With().Str("module", "auth").Logger()
	return &Handler{
		service: service,
		logger:  moduleLogger,
	}
}

func (h *Handler) Register(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid register request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, err := h.service.Register(ctx, req)
	if err != nil {
		if err.Error() == "email already registered" {
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
			return
		}

		h.logger.Error().Err(err).Msg("Failed to register user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to register user",
		})
		return
	}
	h.logger.Info().Int("user_id", user.ID).Str("email", user.Email).Msg("User registered successfully")
	c.JSON(http.StatusCreated, user)
}

func (h *Handler) Login(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid login request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	deviceName := parseDeviceName(c.GetHeader("User-Agent"))
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	authResponse, err := h.service.Login(ctx, req, deviceName, ipAddress, userAgent)
	if err != nil {
		// no revelar si es email o password invalido
		if err.Error() == "invalid credentials" || err.Error() == "account is inactive" {
			h.logger.Warn().Str("email", req.Email).Str("ip", ipAddress).Msg("Failed login attempt")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid credentials",
			})
			return
		}
		h.logger.Error().Err(err).Msg("Login failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Login failed",
		})
		return
	}

	h.logger.Info().
		Str("email", req.Email).
		Str("ip", ipAddress).
		Str("device", deviceName).
		Msg("Login successful")
	c.JSON(http.StatusOK, authResponse)
}

func (h *Handler) Refresh(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid refresh request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	authResponse, err := h.service.Refresh(ctx, req.RefreshToken)
	if err != nil {
		// diferentes mensajes segun error
		if err.Error() == "invalid refresh token" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
			return
		}
		if err.Error() == "refresh token expired" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token expired"})
			return
		}
		if err.Error() == "token has been revoked" {
			h.logger.Warn().Str("ip", c.ClientIP()).Msg("Attempt to refresh revoked token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token has been revoked"})
			return
		}
		h.logger.Error().Err(err).Msg("Refresh failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to refresh token",
		})
		return
	}
	c.JSON(http.StatusOK, authResponse)
}

func (h *Handler) Logout(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn().Err(err).Msg("Invalid refresh request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.service.Logout(ctx, req.RefreshToken)
	if err != nil {
		h.logger.Error().Err(err).Msg("Logout failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) LogoutEverywhere(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDInterface.(int)

	err := h.service.LogoutEverywhere(ctx, userID)
	if err != nil {
		h.logger.Error().Err(err).Int("user_id", userID).Msg("Logout everywhere failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout everywhere"})
		return
	}

	h.logger.Info().Int("user_id", userID).Msg("Logout everywhere successful")
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetSessions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDInterface.(int)

	currentToken := ""
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		currentToken = strings.TrimPrefix(authHeader, "Bearer ")
	}

	sessions, err := h.service.GetActiveSessions(ctx, userID, currentToken)
	if err != nil {
		h.logger.Error().Err(err).Int("user_id", userID).Msg("Failed to get sessions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sessions"})
		return
	}
	c.JSON(http.StatusOK, sessions)
}

func (h *Handler) RevokeSession(c *gin.Context) {
	// ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	// defer cancel()

	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDInterface.(int)

	// TODO implementar en service - verificar que la sesion pertenezca al usuario
	// por ahora asumimos que la sesion es valida

	h.logger.Info().Int("user_id", userID).Msg("Session revoked successfully")
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDInterface.(int)

	user, err := h.service.repo.GetUserByID(ctx, userID)
	if err != nil {
		h.logger.Error().Err(err).Int("user_id", userID).Msg("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}
	response := user.ToUserResponse()
	c.JSON(http.StatusOK, response)
}

func parseDeviceName(userAgent string) string {
	ua := strings.ToLower(userAgent)

	// Detectar mobile
	if strings.Contains(ua, "iphone") {
		return "iPhone"
	}
	if strings.Contains(ua, "ipad") {
		return "iPad"
	}
	if strings.Contains(ua, "android") {
		if strings.Contains(ua, "mobile") {
			return "Android Phone"
		}
		return "Android Tablet"
	}

	// Detectar desktop
	browser := "Unknown Browser"
	if strings.Contains(ua, "chrome") && !strings.Contains(ua, "edge") {
		browser = "Chrome"
	} else if strings.Contains(ua, "firefox") {
		browser = "Firefox"
	} else if strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") {
		browser = "Safari"
	} else if strings.Contains(ua, "edge") {
		browser = "Edge"
	}

	os := "Unknown OS"
	if strings.Contains(ua, "windows") {
		os = "Windows"
	} else if strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os") {
		os = "macOS"
	} else if strings.Contains(ua, "linux") {
		os = "Linux"
	}

	return browser + " / " + os
}
