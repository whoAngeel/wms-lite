package platform

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

func NewDatabase(cfg DatabaseConfig) (*sqlx.DB, error) {
	// datasource name : string de conexion
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
	)
	// open connection
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error conectando a la base de datos: %w", err)
	}

	// configurar pool de conexiones
	db.SetMaxOpenConns(cfg.MaxOpenConns)       // maximo de conexiones abiertas
	db.SetMaxIdleConns(cfg.MaxIdleConns)       //  maximo de conexiones inactivas
	db.SetConnMaxLifetime(cfg.ConnMaxLifeTime) // tiempo de vida de una conexion

	// ping para verificar la conexion
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error making ping to database: %w", err)
	}

	fmt.Println("Database connected successfully")
	return db, nil
}
