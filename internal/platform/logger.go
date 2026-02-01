package platform

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func InitLogger(env string) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	var output io.Writer = os.Stdout

	if env == "development" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	}

	var logLevel zerolog.Level
	switch env {
	case "production":
		logLevel = zerolog.InfoLevel
	case "development":
		logLevel = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	// agregar timestamp/ file:line de donde se llamo al log
	logger := zerolog.New(output).With().Timestamp().Logger()

	log.Logger = logger

	return logger
}
