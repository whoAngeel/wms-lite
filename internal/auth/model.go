package auth

import "time"

type User struct {
	ID           int       `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	FullName     string    `db:"full_name" json:"full_name"`
	Role         string    `db:"role" json:"role"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type Session struct {
	ID            int       `db:"id" json:"id"`
	UserID        int       `db:"user_id" json:"user_id"`
	RefreshToken  string    `db:"refresh_token" json:"refresh_token"`
	TokenFamily   string    `db:"token_family" json:"token_family"`
	IsRevoked     bool      `db:"is_revoked" json:"is_revoked"`
	DeviceName    *string   `db:"device_name" json:"device_name,omitempty"` // puntero para permitir null
	IPAddress     *string   `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent     *string   `db:"user_agent" json:"user_agent,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	LastUsedAt    time.Time `db:"last_used_at" json:"last_used_at"`
	ExpiresAt     time.Time `db:"expires_at" json:"expires_at"`
	ParentTokenID *int      `db:"parent_token_id" json:"parent_token_id,omitempty"`
}

// DTOs (request and responses)
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	FullName string `json:"full_name" binding:"required"`
	Role     string `json:"role"` // opcional user por defecto
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"` // segundos
	TokenType    string `json:"token_type"` // Bearer
}

type UserResponse struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionResponse struct {
	ID         int       `json:"id"`
	DeviceName string    `json:"device_name"`
	IPAddress  string    `json:"ip_address"`
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IsCurrent  bool      `json:"is_current"` // true si es la sesion actuan
}

type SessionListResponse struct {
	Sessions []SessionResponse `json:"sessions"`
	Total    int               `json:"total"`
}

/// JWT Claims (payload del access token)
type JWTClaims struct { // son para los datos que van dentro del token
	UserID   int    `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

// HELPERS

func (u *User) ToUserResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
	}
}

func (s *Session) ToSessionResponse(isCurrent bool) SessionResponse {
	// Manejar punteros nullables
	deviceName := ""
	if s.DeviceName != nil {
		deviceName = *s.DeviceName
	}

	ipAddress := ""
	if s.IPAddress != nil {
		ipAddress = *s.IPAddress
	}
	return SessionResponse{
		ID:         s.ID,
		DeviceName: deviceName,
		IPAddress:  ipAddress,
		LastUsedAt: s.LastUsedAt,
		ExpiresAt:  s.ExpiresAt,
		IsCurrent:  isCurrent,
	}
}
