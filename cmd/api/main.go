package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/whoAngeel/wms-lite/internal/movement"
	"github.com/whoAngeel/wms-lite/internal/platform"
	"github.com/whoAngeel/wms-lite/internal/product"
)

func main() {
	// 1 load configuration
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	cfg, err := platform.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	log.Printf("WMS Lite starting on mode %s\n", cfg.Server.Env)

	// 2 connect to database
	db, err := platform.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	productRepo := product.NewRepository(db)
	productService := *product.NewService(productRepo)
	productHandler := product.NewHandler(&productService)

	movementRepo := movement.NewRepository(db)
	movementService := *movement.NewService(movementRepo, db)
	movementHandler := movement.NewHandler(&movementService)

	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	corsConfig := cors.Config{
		AllowOrigins: []string{
			"http://localhost:3001", "http://localhost:5137",
		},
		AllowMethods: []string{
			"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization",
		},
		ExposeHeaders: []string{
			"Content-Length",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	if cfg.Server.Env == "development" {
		corsConfig.AllowOrigins = []string{"*"}
	}

	router.Use(cors.New(corsConfig))

	setupRoutes(router, productHandler, movementHandler)

	srv := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB,
	}
	go func() {
		log.Printf("Server is running on http://localhost:%s", cfg.Server.Port)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error running server: %v", err)
		}
	}()

	// 3 graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // block until signal received

	log.Printf("Shutting down server...")

	// context con timeout para shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}
	db.Close()
	log.Println("Server stopped")
}

func setupRoutes(router *gin.Engine,
	productHandler *product.Handler,
	movementHandler *movement.Handler,
) {
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
		movements := v1.Group("/movements")
		{
			movements.POST("", movementHandler.Create)
			movements.GET("", movementHandler.List)
			movements.GET("/:id", movementHandler.GetByID)
			movements.GET("/product/:id", movementHandler.ListByProductID)
		}
	}
}
