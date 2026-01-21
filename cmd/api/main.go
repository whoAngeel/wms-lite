package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/whoAngeel/wms-lite/internal/platform"
)

func main() {
	// 1 load configuration
	cfg, err := platform.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	fmt.Printf("WMS Lite starting on mode %s\n", cfg.Server.Env)

	// 2 connect to database
	db, err := platform.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// 3 graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("Server ready on port %s", cfg.Server.Port)
	fmt.Printf("Press CTRL+C to stop")

	<-quit // block until signal received
	fmt.Println("Shutting down server...")
	db.Close()
	fmt.Println("Database connection closed")
	fmt.Println("Server stopped")
}
