package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/whoAngeel/wms-lite/internal/platform"
	"github.com/whoAngeel/wms-lite/internal/product"
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

	productRepo := product.NewRepository(db)
	productService := *product.NewService(productRepo)
	productHandler := product.NewHandler(&productService)

	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	setupRoutes(router, productHandler)

	srv := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB,
	}
	go func() {
		fmt.Printf("Server is running on http://localhost:%s", cfg.Server.Port)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error running server: %v", err)
		}
	}()

	// 3 graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // block until signal received

	fmt.Println("Shutting down server...")

	// context con timeout para shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}
	db.Close()
	fmt.Println("Server stopped")
}

func setupRoutes(router *gin.Engine, productHandler *product.Handler) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "OKitocke",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	v1 := router.Group("/api/v1")

	{
		products := v1.Group("/products")
		{
			products.POST("", productHandler.Create)
			products.GET("", productHandler.GetAll)
			products.GET("/:id", productHandler.GetByID)
			products.GET("/sku/:sku", productHandler.GetBySKU)
			products.PUT("/:id", productHandler.Update)
			products.DELETE("/:id", productHandler.Delete)
		}
	}
}
