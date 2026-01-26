package platform

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Driver de PostgreSQL
)

// NewDatabase crea una conexión a PostgreSQL con pool de conexiones
func NewDatabase(cfg DatabaseConfig) (*sqlx.DB, error) {
	// DSN (Data Source Name) - string de conexión
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
	)

	// Abrir conexión con sqlx (envuelve database/sql estándar)
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error conectando a la base de datos: %w", err)
	}

	// Configurar pool de conexiones
	db.SetMaxOpenConns(cfg.MaxOpenConns)       // Máximo de conexiones abiertas
	db.SetMaxIdleConns(cfg.MaxIdleConns)       // Máximo de conexiones inactivas
	db.SetConnMaxLifetime(cfg.ConnMaxLifeTime) // Tiempo de vida de una conexión

	// Ping para verificar conectividad
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error al hacer ping a la base de datos: %w", err)
	}

	fmt.Println("✅ Conexión a PostgreSQL establecida")
	return db, nil
}
