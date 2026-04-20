package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"data-hub/cache"
	"data-hub/client"
	"data-hub/config"
	"data-hub/graph"
	"data-hub/graph/generated"
	"data-hub/middleware"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 1. JSON logger — zap production mode outputs structured JSON, ready for Grafana
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// 2. Load config from .env or environment variables
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// 3. Redis cache — no-op if CACHE_ENABLED=false (safe to run without Redis)
	cacheStore := cache.NewCache(cfg)

	// 4. Service clients
	adminClient     := client.NewAdminClient(cfg, logger)
	kycClient       := client.NewKycClient(cfg, logger)
	repaymentClient := client.NewRepaymentClient(cfg, logger)

	// 5. Root resolver with all dependencies injected
	resolver := &graph.Resolver{
		AdminClient:     adminClient,
		KycClient:       kycClient,
		RepaymentClient: repaymentClient,
		Cache:           cacheStore,
		Config:          cfg,
		Logger:          logger,
	}

	// 6. GraphQL server (gqlgen)
	gqlServer := handler.NewDefaultServer(
		generated.NewExecutableSchema(generated.Config{Resolvers: resolver}),
	)

	// 7. Gin router — gin.New() not gin.Default() so we use zap instead of gin's logger
	r := gin.New()
	r.Use(gin.Recovery())                     // recover from panics
	r.Use(middleware.RequestID())             // unique ID on every request
	r.Use(middleware.APIKeyAuth(cfg, logger)) // validate X-Api-Key header

	// 8. Routes
	// GraphQL Playground — open in browser to explore schema and run queries interactively
	playgroundHandler := playground.Handler("Data Hub GraphQL", "/graphql")
	r.GET("/graphql", func(c *gin.Context) {
		playgroundHandler.ServeHTTP(c.Writer, c.Request)
	})

	r.POST("/graphql", func(c *gin.Context) {
		// Embed the request ID into Go context so it flows through resolvers and clients
		ctx := middleware.SetRequestID(
			c.Request.Context(),
			c.GetString(string(middleware.RequestIDKey)),
		)
		c.Request = c.Request.WithContext(ctx)
		gqlServer.ServeHTTP(c.Writer, c.Request)
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 9. Start with graceful shutdown
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}

	go func() {
		logger.Info("server started", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	// Block until Ctrl+C or SIGTERM (Kubernetes sends SIGTERM on pod shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}
	logger.Info("server stopped")
}