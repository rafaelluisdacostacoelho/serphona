package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	"go.uber.org/zap"

	"tools-manager/internal/config"
	"tools-manager/internal/events"
	"tools-manager/internal/repository"
	"tools-manager/internal/secret"
	"tools-manager/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	configureAuth(cfg)

	logger, err := newLogger(cfg.Server.Env)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync() // flush buffered logs

	pool, err := setupPool(cfg)
	if err != nil {
		logger.Fatal("failed to init db pool", zap.Error(err))
	}
	defer pool.Close()

	repo := repository.New(pool)
	backend := secret.NewMemoryProvider()
	if cfg.Secrets.UseVault {
		vaultBackend, err := secret.NewVaultProvider(cfg.Secrets.VaultAddress, cfg.Secrets.VaultToken, cfg.Secrets.VaultMount, cfg.Secrets.VaultPrefix)
		if err != nil {
			logger.Fatal("failed to init vault backend", zap.Error(err))
		}
		backend = vaultBackend
	}
	secretStore, err := secret.NewStore(cfg.Secrets.EncryptionKey, cfg.Secrets.CacheTTL, backend)
	if err != nil {
		logger.Fatal("failed to init secret store", zap.Error(err))
	}
	notifier := events.NewNotifier(cfg.Integrations.EventsWebhookURL)

	router := server.NewRouter(cfg, logger, repo, secretStore, notifier)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Info("starting tools-manager", zap.String("addr", addr), zap.String("env", cfg.Server.Env))
	if err := router.Engine.Run(addr); err != nil {
		logger.Fatal("server exited", zap.Error(err))
	}
}

func newLogger(env string) (*zap.Logger, error) {
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

func configureAuth(cfg *config.Config) {
	authjwt.SetValidationConfig(authjwt.ValidationConfig{
		AllowedAlgs:     cfg.Auth.AllowedAlgs,
		Issuer:          cfg.Auth.Issuer,
		Audience:        cfg.Auth.Audience,
		ServiceAudience: cfg.Auth.ServiceAudience,
		ClockSkew:       cfg.Auth.ClockSkew,
		MaxTokenBytes:   cfg.Auth.MaxTokenBytes,
		JWKSURL:         cfg.Auth.JWKSURL,
		JWKSCacheTTL:    cfg.Auth.JWKSCacheTTL,
		AllowedKIDs:     cfg.Auth.AllowedKIDs,
		RequiredScopes:  cfg.Auth.RequiredScopes,
	})

	if cfg.Auth.JWTSecret != "" {
		authjwt.SetSecret(cfg.Auth.JWTSecret)
	}
}

func setupPool(cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = cfg.Database.MaxConns
	return pgxpool.NewWithConfig(context.Background(), poolCfg)
}
