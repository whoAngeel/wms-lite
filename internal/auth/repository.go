package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

func NewRepository(db *sqlx.DB, logger zerolog.Logger) *Repository {
	moduleLogger := logger.With().Str("layer", "auth-repository").Logger()
	return &Repository{
		db:     db,
		logger: moduleLogger,
	}
}

func (r *Repository) CreateUser(ctx context.Context, email, fullName, password, role string) (*User, error) {
	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	query := `
		INSERT INTO users (email, full_name, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, full_name, password_hash, role, is_active, created_at, updated_at
	`

	var user User
	err = r.db.QueryRowContext(ctx, query, email, fullName, passwordHash, role).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, full_name, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetUserByID(ctx context.Context, userID int) (*User, error) {
	query := `
		SELECT id, email, full_name, password_hash, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.FullName,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// compara el password en texto plano con el hash
func (r *Repository) VerifyPassword(hashedPassword, plainPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
}

/// ========== SESIONS ============

// refresh token
func (r *Repository) CreateSession(
	ctx context.Context,
	userID int,
	deviceName, ipAddress, userAgent string,
	expiresAt time.Time,
) (*Session, error) {
	// generar refresh token
	refreshToken := uuid.New().String()

	// generar token family agrupa tokens rotados
	tokenFamily := uuid.New().String()

	// insertar sesion
	query := `
			INSERT INTO sessions (
				user_id, refresh_token, token_family, device_name, ip_address, user_agent, expires_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, user_id, refresh_token, token_family, is_revoked, device_name, ip_address, user_agent, expires_at, created_at, last_used_at, parent_token_id
		`
	var session Session
	err := r.db.QueryRowContext(
		ctx, query,
		userID,
		refreshToken,
		tokenFamily,
		nullString(deviceName),
		nullString(ipAddress),
		nullString(userAgent),
		expiresAt,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshToken,
		&session.TokenFamily,
		&session.IsRevoked,
		&session.DeviceName,
		&session.IPAddress,
		&session.UserAgent,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.LastUsedAt,
		&session.ParentTokenID,
	)

	if err != nil {
		return nil, err
	}
	return &session, nil
}

// busca por refresh token
func (r *Repository) FindSessionByToken(ctx context.Context, refreshToken string) (*Session, error) {
	query := `
		SELECT id, user_id, refresh_token, token_family, is_revoked,
			   device_name, ip_address, user_agent,
			   created_at, last_used_at, expires_at, parent_token_id
		FROM sessions
		WHERE refresh_token = $1
	`

	var session Session
	err := r.db.QueryRowContext(ctx, query, refreshToken).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshToken,
		&session.TokenFamily,
		&session.IsRevoked,
		&session.DeviceName,
		&session.IPAddress,
		&session.UserAgent,
		&session.CreatedAt,
		&session.LastUsedAt,
		&session.ExpiresAt,
		&session.ParentTokenID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// crea un nuevo refresh token y marca el anterior como padre
// esto es critico para detectar reutilizacion de tokens
func (r *Repository) RotateRefreshToken(
	ctx context.Context,
	oldSession *Session,
	expiresAt time.Time,
) (*Session, error) {
	newRefreshToken := uuid.New().String()

	query := `
		INSERT INTO sessions (
			user_id, refresh_token, token_family, device_name, ip_address, user_agent, expires_at, parent_token_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, refresh_token, token_family, is_revoked, device_name, ip_address, user_agent, expires_at, created_at, last_used_at, parent_token_id
	`

	var newSession Session
	err := r.db.QueryRowContext(
		ctx, query,
		oldSession.UserID,
		newRefreshToken,
		oldSession.TokenFamily,
		oldSession.DeviceName,
		oldSession.IPAddress,
		oldSession.UserAgent,
		expiresAt,
		oldSession.ID,
	).Scan(
		&newSession.ID,
		&newSession.UserID,
		&newSession.RefreshToken,
		&newSession.TokenFamily,
		&newSession.IsRevoked,
		&newSession.DeviceName,
		&newSession.IPAddress,
		&newSession.UserAgent,
		&newSession.ExpiresAt,
		&newSession.CreatedAt,
		&newSession.LastUsedAt,
		&newSession.ParentTokenID,
	)

	if err != nil {
		r.logger.Error().Err(err).Int("user_id", oldSession.UserID).Int("old_session_id", oldSession.ID).Msg("failed to rotate refresh token")
		return nil, err
	}

	// marcar el token viejo como revocado
	_, err = r.db.ExecContext(
		ctx, `UPDATE sessions SET is_revoked = true WHERE id = $1`, oldSession.ID,
	)
	if err != nil {
		return nil, err
	}

	return &newSession, nil
}

// actualiza el timestamp de la ultima actividad
func (r *Repository) UpdateSessionLastUsed(ctx context.Context, sessionID int) error {
	query := `
		UPDATE sessions
		SET last_used_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, sessionID)
	return err
}

func (r *Repository) RevokeSession(ctx context.Context, sessionID int) error {
	query := `UPDATE sessions SET is_revoked = true WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, sessionID)
	return err
}

// revoca todas las sesiones de un usuario
// util para `logout everywhere` o cuando se detecta actividad sospechosa
func (r *Repository) RevokeAllUserSessions(ctx context.Context, userID int) error {
	query := `UPDATE sessions SET is_revoked = true WHERE user_id = $1 AND is_revoked = false`

	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// revoca toda una familia de tokens
// se usa cuando se detecta reutilizacion de un token rotado
func (r *Repository) RevokeTokenFamily(ctx context.Context, token_family string) error {
	query := `UPDATE sessions SET is_revoked = true WHERE token_family = $1 AND is_revoked = false`

	_, err := r.db.ExecContext(ctx, query, token_family)
	return err
}

// obtiene todas las sesiones de un usuario
func (r *Repository) GetUserSessions(ctx context.Context, userID int) ([]Session, error) {
	query := `
		SELECT id, user_id, refresh_token, token_family, is_revoked,
			   device_name, ip_address, user_agent,
			   created_at, last_used_at, expires_at, parent_token_id
		FROM sessions
		WHERE user_id = $1 
		  AND is_revoked = false 
		  AND expires_at > CURRENT_TIMESTAMP
		ORDER BY last_used_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.RefreshToken,
			&session.TokenFamily,
			&session.IsRevoked,
			&session.DeviceName,
			&session.IPAddress,
			&session.UserAgent,
			&session.CreatedAt,
			&session.LastUsedAt,
			&session.ExpiresAt,
			&session.ParentTokenID,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// limpia sessions expiradas de la db
func (r *Repository) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

/// ========== HELPERS ==========

func hashPassword(password string) (string, error) {
	// bcrypt.DefaultCost = 10 (2^10 = 1024 rounds)
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// nullString convierte un string en sql.NullString para campos nullable
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
