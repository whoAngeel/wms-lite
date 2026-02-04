package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
)

type Service struct {
	repo      *Repository
	db        *sqlx.DB
	logger    zerolog.Logger
	jwtSecret []byte
}

func NewService(repo *Repository, db *sqlx.DB, logger zerolog.Logger, jwtSecret string) *Service {
	moduleLogger := logger.With().Str("layer", "auth-service").Logger()
	return &Service{
		repo:      repo,
		db:        db,
		logger:    moduleLogger,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {
	// validar que el email no exista
	existingUser, _ := s.repo.GetUserByEmail(ctx, req.Email)
	if existingUser != nil {
		return nil, fmt.Errorf("email already registered")
	}

	// validar role (user o admin)
	role := req.Role
	if role == "" {
		role = "user"
	}

	if role != "user" && role != "admin" && role != "readonly" {
		return nil, fmt.Errorf("invalid role: must be user, admin, or readonly")
	}

	// crear usuario (el repository hashea el password)
	user, err := s.repo.CreateUser(ctx, req.Email, req.Password, role)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create user")
		return nil, fmt.Errorf("failed to create user")
	}

	s.logger.Info().Int("user_id", user.ID).Str("email", user.Email).Str("role", user.Role).Msg("User register successfully")

	response := user.ToUserResponse()
	return &response, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest, deviceName, ipAddress, userAgent string) (*AuthResponse, error) {
	// buscar usuario por email
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Warn().Str("email", req.Email).Msg("Login attemp with non-existent email")
		return nil, fmt.Errorf("invalid credentials")
	}

	// verificar que el usuario este activo
	if !user.IsActive {
		s.logger.Warn().Int("user_id", user.ID).Msg("Login attempt for inactive user")
		return nil, fmt.Errorf("account is inactive")
	}

	// verificar password
	err = s.repo.VerifyPassword(user.PasswordHash, req.Password)
	if err != nil {
		s.logger.Warn().Str("email", req.Email).Msg("Login attempt with wrong password")
		return nil, fmt.Errorf("invalid credentials")
	}

	// generar access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to generate access token")
		return nil, fmt.Errorf("failed to generate access token")
	}

	// crear sesion (refresh token)
	expiresAt := time.Now().Add(RefreshTokenDuration)
	session, err := s.repo.CreateSession(ctx, user.ID, deviceName, ipAddress, userAgent, expiresAt)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create session")
		return nil, fmt.Errorf("failed to create session")
	}

	s.logger.Info().Int("user_id", user.ID).Str("email", user.Email).Int("session_id", session.ID).Str("ip_address", ipAddress).Msg("Login successfully")
	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: session.RefreshToken,
		ExpiresAt:    int64(AccessTokenDuration.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// buscar sesion por refresh token
	session, err := s.repo.FindSessionByToken(ctx, refreshToken)
	if err != nil {
		s.logger.Warn().
			Str("refresh_token", refreshToken[:20]+"...").
			Msg("Refresh attempt with invalid token")
		return nil, fmt.Errorf("invalid refresh token")
	}

	// verificar que no este revocado
	if session.IsRevoked {
		s.logger.Warn().
			Int("session_id", session.ID).
			Str("token_family", session.TokenFamily).
			Msg("Attempt to use revoked token - possible token reuse attack")
		err := s.repo.RevokeTokenFamily(ctx, session.TokenFamily)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to revoke token family")
		}
		return nil, fmt.Errorf("token has been revoked")
	}

	// verificar que no haya expirado
	if time.Now().After(session.ExpiresAt) {
		s.logger.Warn().
			Int("session_id", session.ID).
			Msg("Refresh attempt with expired token")
		return nil, fmt.Errorf("refresh token has expired")
	}

	// buscar usuario
	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		s.logger.Error().Err(err).Int("user_id", session.UserID).Msg("User not found")
		return nil, fmt.Errorf("user not found")
	}

	// verificar que el usuario siga activo
	if !user.IsActive {
		s.logger.Warn().Int("user_id", user.ID).Msg("Refresh attempt for inactive user")
		return nil, fmt.Errorf("account is inactive")
	}

	// generar nuevo access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to generate access token")
		return nil, fmt.Errorf("failed to generate token")
	}

	// Rotar refresh token (crear nuevo, revocar viejo)
	newExpiresAt := time.Now().Add(RefreshTokenDuration)
	newSession, err := s.repo.RotateRefreshToken(ctx, session, newExpiresAt)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to rotate refresh token")
		return nil, fmt.Errorf("failed to rotate token")
	}

	s.logger.Info().
		Int("user_id", user.ID).
		Int("old_session_id", session.ID).
		Int("new_session_id", newSession.ID).
		Msg("Token refreshed successfully")

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newSession.RefreshToken,
		ExpiresAt:    int64(AccessTokenDuration.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	// buscar session
	session, err := s.repo.FindSessionByToken(ctx, refreshToken)
	if err != nil {
		// no hacer nada si el token no existe (ya fue eliminado)
		return nil
	}

	// revocar session
	err = s.repo.RevokeSession(ctx, session.ID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to revoke token")
		return err
	}

	s.logger.Info().Int("session_id", session.ID).Int("user_id", session.UserID).Msg("User logged out successfully")
	return nil
}

func (s *Service) LogoutEverywhere(ctx context.Context, userID int) error {
	err := s.repo.RevokeAllUserSessions(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Int("user_id", userID).Msg("Failed to revoke all session")
		return err
	}

	s.logger.Info().Int("user_id", userID).Msg("User logged out from everywhere successfully")
	return nil
}

// listar sesiones activas
func (s *Service) GetActiveSessions(ctx context.Context, userID int, currentRefreshToken string) (*SessionListResponse, error) {
	sessions, err := s.repo.GetUserSessions(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get user sessions")
		return nil, fmt.Errorf("failed to get user sessions")
	}

	sessionResponses := make([]SessionResponse, 0, len(sessions))
	for _, session := range sessions {
		isCurrent := session.RefreshToken == currentRefreshToken
		sessionResponses = append(sessionResponses, session.ToSessionResponse(isCurrent))
	}

	return &SessionListResponse{
		Sessions: sessionResponses,
		Total:    len(sessionResponses),
	}, nil
}

func (s *Service) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	// parser
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verificar que el algoritmo sea HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	// extraer claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		jwtClaims := &JWTClaims{
			UserID: int(claims["user_id"].(float64)),
			Email:  claims["email"].(string),
			Role:   claims["role"].(string),
		}
		return jwtClaims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *Service) generateAccessToken(user *User) (string, error) {
	// 1. Crear claims (payload del JWT)
	now := time.Now()
	claims := jwt.MapClaims{
		// Claims estándar
		"exp": now.Add(AccessTokenDuration).Unix(), // Expiración
		"iat": now.Unix(),                          // Issued at
		"iss": "wms-lite",                          // Issuer

		// Claims personalizados
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
	}

	// 2. Crear token con algoritmo HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 3. Firmar con secret key
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
