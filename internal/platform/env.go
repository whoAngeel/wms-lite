package platform

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// config contiene toda la configuracion de la app
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	Auth     AuthConfig
	Cache    CacheConfig
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Name            string
	Password        string
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifeTime time.Duration
}

type ServerConfig struct {
	Port           string
	Env            string
	AllowedOrigins []string
}

type AuthConfig struct {
	JWTSecret string
}

type CacheConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// loadConfig carga las variables de entorno
func LoadConfig() (*Config, error) {
	// cargar .env en desarrollo
	if os.Getenv("ENV") != "production" {
		if err := godotenv.Load(); err != nil {
			fmt.Println("File .env not found")
		}
	}

	// parsear configuracion de la base de datos
	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5434"))
	if err != nil {
		return nil, fmt.Errorf("invalid database port: %v", err)
	}

	maxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	maxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "5"))

	connMaxLifeTime, err := time.ParseDuration(getEnv("DB_CONN_MAX_LIFE_TIME", "5m"))
	if err != nil {
		connMaxLifeTime = 5 * time.Minute
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is not set")
	}

	config := &Config{
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            dbPort,
			User:            getEnv("DB_USER", "postgres"),
			Name:            getEnv("DB_NAME", "wms_db"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			URL:             getEnv("DATABASE_URL", ""),
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxLifeTime: connMaxLifeTime,
		},
		Server: ServerConfig{
			Port:           getEnv("API_PORT", "4002"),
			Env:            getEnv("ENV", "development"),
			AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "*"), ","),
		},
		Auth: AuthConfig{
			JWTSecret: jwtSecret,
		},
		Cache: CacheConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
	}
	return config, nil

}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}

	return defaultValue
}
