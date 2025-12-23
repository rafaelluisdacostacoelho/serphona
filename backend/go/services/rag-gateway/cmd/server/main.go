package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	eventscfg "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/config"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/publisher"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-rag/pkg/vector/pgvector"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/config"
	embedopenai "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/embedding/openai"
	embedstub "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/embedding/stub"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/server"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/server/handler"
	pgvector_uc "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/usecase/pgvector"
	ucstub "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/rag-gateway/internal/usecase/stub"
	obs "github.com/serphona/backend/go/libs/platform-observability"
	obscfg "github.com/serphona/backend/go/libs/platform-observability/config"
	obsmw "github.com/serphona/backend/go/libs/platform-observability/middleware"
)

func main() {
	log.Println("Starting RAG Gateway service...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	obsConfig := obscfg.LoadFromEnv()
	if obsConfig.ServiceName == "" {
		obsConfig.ServiceName = "rag-gateway"
	}

	observer, obsErr := obs.Init(obsConfig)
	if obsErr != nil {
		log.Printf("observability init failed (continuing without tracing/metrics): %v", obsErr)
	}

	db, err := sql.Open("pgx", cfg.PGVectorURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	store := pgvector.NewStore(db, pgvector.Config{URL: cfg.PGVectorURL, TableName: cfg.PGVectorTable, Dimension: cfg.PGVectorDim})
	if err := store.Ping(context.Background()); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}

	embeddingClient := selectEmbeddingClient(cfg)
	evtPublisher := maybeInitPublisher()

	ingestSvc := pgvector_uc.NewIngestUsecase(store, embeddingClient, cfg.PGVectorDim, evtPublisher)
	querySvc := pgvector_uc.NewQueryUsecase(store, embeddingClient, cfg.TopKDefault, cfg.PGVectorDim)
	namespaceSvc := ucstub.Namespace{}
	ragHandler := handler.NewRAGHandler(ingestSvc, querySvc, namespaceSvc)

	router := server.NewRouter(ragHandler)
	var handler http.Handler = router
	if observer != nil {
		handler = obsmw.HTTP(router, obsConfig.ServiceName)
	}

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("Server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	if observer != nil {
		_ = observer.Shutdown(ctx)
	}

	log.Println("Server exited")
}

func selectEmbeddingClient(cfg config.Config) interface {
	Embed(context.Context, []string) ([][]float32, error)
} {
	switch cfg.EmbeddingProvider {
	case "openai":
		client, err := embedopenai.New(embedopenai.Config{
			APIKey:  cfg.EmbeddingAPIKey,
			Model:   cfg.EmbeddingModel,
			BaseURL: cfg.EmbeddingBaseURL,
		})
		if err != nil {
			log.Fatalf("failed to init openai embedding client: %v", err)
		}
		return client
	default:
		return embedstub.Client{}
	}
}

func maybeInitPublisher() *publisher.Publisher {
	enabled := parseBool(os.Getenv("RAG_EVENTS_ENABLED"))
	if !enabled {
		log.Println("event publisher disabled (RAG_EVENTS_ENABLED not true)")
		return nil
	}

	cfg := eventscfg.LoadFromEnv()
	if cfg.ServiceName == "" {
		cfg.ServiceName = "rag-gateway"
	}

	pub, err := publisher.New(cfg)
	if err != nil {
		log.Printf("failed to initialize event publisher: %v (continuing without events)", err)
		return nil
	}

	return pub
}

func parseBool(v string) bool {
	switch strings.ToLower(v) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
