package platform

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger configura el logger global de la aplicación
// Retorna el logger configurado para que puedas usarlo en otros archivos
func InitLogger(env string) zerolog.Logger {
	// 1. Configurar el formato de tiempo
	zerolog.TimeFieldFormat = time.RFC3339 // ISO 8601: "2006-01-02T15:04:05Z07:00"

	// 2. Configurar el output (dónde se escriben los logs)
	var output io.Writer = os.Stdout // Por defecto escribir en consola

	// 3. En desarrollo, usar formato humano bonito con colores
	if env == "development" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    false, // Activar colores en consola
		}
	}

	// 4. Configurar nivel de logging según el entorno
	var logLevel zerolog.Level
	switch env {
	case "production":
		logLevel = zerolog.InfoLevel // Solo INFO, WARN, ERROR, FATAL
	case "development":
		logLevel = zerolog.DebugLevel // Todo: DEBUG, INFO, WARN, ERROR, FATAL
	default:
		logLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	// 5. Crear el logger
	logger := zerolog.New(output).
		With().
		Timestamp(). // Agregar timestamp automáticamente a cada log
		Caller().    // Agregar archivo:línea de donde se llamó el log
		Logger()

	// 6. Establecerlo como logger global (opcional pero útil)
	log.Logger = logger

	return logger
}
