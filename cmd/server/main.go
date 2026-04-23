package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shaurya2807/url-shortener/configs"
	"github.com/shaurya2807/url-shortener/internal/cache"
	"github.com/shaurya2807/url-shortener/internal/handler"
	"github.com/shaurya2807/url-shortener/internal/repository"
	"github.com/shaurya2807/url-shortener/internal/service"
	"github.com/shaurya2807/url-shortener/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	cfg, err := configs.Load()
	if err != nil {
		panic(fmt.Sprintf("load config: %v", err))
	}

	log, err := logger.New(cfg.AppEnv)
	if err != nil {
		panic(fmt.Sprintf("init logger: %v", err))
	}
	defer log.Sync()

	db, err := pgxpool.New(context.Background(), cfg.DB.DSN())
	if err != nil {
		log.Fatal("connect to db", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatal("ping db", zap.Error(err))
	}
	log.Info("database connected")

	repo := repository.NewURLRepository(db)
	redisCache := cache.New(cfg.Redis.Host, cfg.Redis.Port)
	svc := service.NewURLService(repo, redisCache, cfg.BaseURL, log)
	h := handler.NewURLHandler(svc, log)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/shorten", h.Shorten)
	r.GET("/:code", h.Redirect)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("server starting", zap.Int("port", cfg.ServerPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}
	log.Info("server stopped")
}
