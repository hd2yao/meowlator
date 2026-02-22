package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/dysania/meowlator/services/api/internal/api"
	"github.com/dysania/meowlator/services/api/internal/app"
	"github.com/dysania/meowlator/services/api/internal/config"
	"github.com/dysania/meowlator/services/api/internal/repository"
)

func main() {
	cfg := config.Load()
	var repo app.Repository = repository.NewMemoryRepository()
	if cfg.MySQLDSN != "" {
		mysqlRepo, err := repository.NewMySQLRepository(cfg.MySQLDSN)
		if err != nil {
			log.Fatalf("failed to init mysql repository: %v", err)
		}
		defer func() {
			_ = mysqlRepo.Close()
		}()
		repo = mysqlRepo
		log.Printf("mysql repository enabled")
	} else {
		log.Printf("mysql dsn not set, using in-memory repository")
	}
	inferenceClient := app.NewHTTPInferenceClient(cfg.InferenceServiceURL)
	var copyClient app.CopyClient = app.NewCopyClient(app.CopyClientConfig{Timeout: cfg.CopyTimeout})
	if cfg.RedisAddr != "" {
		cache, err := app.NewRedisCopyCache(cfg.RedisAddr)
		if err != nil {
			log.Printf("redis cache unavailable, skip caching: %v", err)
		} else {
			copyClient = app.NewCachingCopyClient(copyClient, cache, 6*time.Hour)
			log.Printf("redis copy cache enabled")
		}
	}

	svc := app.NewService(
		repo,
		inferenceClient,
		copyClient,
		app.Thresholds{EdgeAccept: cfg.EdgeAcceptThreshold, CloudFallback: cfg.CloudFallbackThreshold},
		cfg.DefaultRetentionDays,
		cfg.ModelVersion,
		cfg.PainRiskEnabled,
		cfg.EdgeDeviceWhitelist,
	)

	h := api.NewHandler(svc, api.HandlerOptions{
		RateLimitPerUserMin: cfg.RateLimitPerUserMin,
		RateLimitPerIPMin:   cfg.RateLimitPerIPMin,
		AdminToken:          cfg.AdminToken,
	})
	mux := http.NewServeMux()
	h.Register(mux)

	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			<-ticker.C
			deleted, err := svc.CleanupExpiredSamples(context.Background())
			if err != nil {
				log.Printf("cleanup expired samples failed: %v", err)
				continue
			}
			if deleted > 0 {
				log.Printf("cleanup expired samples completed: deleted=%d", deleted)
			}
		}
	}()

	log.Printf("api service listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatal(err)
	}
}
