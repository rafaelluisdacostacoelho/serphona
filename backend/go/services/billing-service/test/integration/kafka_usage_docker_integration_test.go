//go:build integration
// +build integration

package kafka

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/adapter/kafka"
	pgrepo "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/adapter/postgres"
	walletapp "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/application/wallet"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/config"
	domain "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/domain/wallet"
)

func TestKafkaUsageEndToEnd_Success(t *testing.T) {
	brokers := testBrokers()
	repo, _, cancel := startUsageConsumer(t, brokers, 200)
	defer cancel()

	producer := mustProducer(t, brokers)
	evt := walletapp.UsageReported{
		TenantID:    uuid.New(),
		Period:      "2024-12",
		Source:      "integration-kafka",
		Calls:       1,
		Minutes:     2,
		Messages:    3,
		APIRequests: 4,
		StorageGB:   1.5,
		Plan:        "starter",
		RequestID:   "req-kafka-docker-success",
	}

	produceUsage(t, producer, evt)

	waitFor(t, 15*time.Second, func() bool {
		w, _ := repo.FindByTenantID(context.Background(), evt.TenantID)
		return w != nil && w.Balance < 200
	})

	wallet, err := repo.FindByTenantID(context.Background(), evt.TenantID)
	if err != nil || wallet == nil {
		t.Fatalf("wallet not persisted: %v", err)
	}

	txs, err := repo.FindTransactionsByWalletID(context.Background(), wallet.ID, 0, 10)
	if err != nil {
		t.Fatalf("find txs: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("expected 1 tx, got %d", len(txs))
	}
}

func TestKafkaUsageEndToEnd_Idempotent(t *testing.T) {
	brokers := testBrokers()
	repo, _, cancel := startUsageConsumer(t, brokers, 100)
	defer cancel()

	producer := mustProducer(t, brokers)
	evt := walletapp.UsageReported{TenantID: uuid.New(), Calls: 1, Plan: "starter", RequestID: "req-kafka-docker-dup"}

	produceUsage(t, producer, evt)
	produceUsage(t, producer, evt)

	waitFor(t, 15*time.Second, func() bool {
		w, _ := repo.FindByTenantID(context.Background(), evt.TenantID)
		return w != nil
	})

	wallet, _ := repo.FindByTenantID(context.Background(), evt.TenantID)
	if wallet.Balance != 95 {
		t.Fatalf("expected balance 95 after duplicate, got %d", wallet.Balance)
	}

	txs, _ := repo.FindTransactionsByWalletID(context.Background(), wallet.ID, 0, 10)
	if len(txs) != 1 {
		t.Fatalf("expected 1 tx after duplicate, got %d", len(txs))
	}

}

func TestKafkaUsageEndToEnd_InsufficientBalance_DLQ(t *testing.T) {
	brokers := testBrokers()
	repo, _, cancel := startUsageConsumer(t, brokers, 0)
	defer cancel()

	producer := mustProducer(t, brokers)
	evt := walletapp.UsageReported{TenantID: uuid.New(), Calls: 5, Plan: "starter", RequestID: "req-kafka-docker-insufficient"}

	produceUsage(t, producer, evt)

	// Expect no debit and a DLQ message
	waitFor(t, 10*time.Second, func() bool {
		w, _ := repo.FindByTenantID(context.Background(), evt.TenantID)
		return w != nil
	})

	wallet, _ := repo.FindByTenantID(context.Background(), evt.TenantID)
	if wallet.Balance != 0 {
		t.Fatalf("expected balance unchanged, got %d", wallet.Balance)
	}

	txs, _ := repo.FindTransactionsByWalletID(context.Background(), wallet.ID, 0, 10)
	if len(txs) != 0 {
		t.Fatalf("expected 0 tx on insufficient, got %d", len(txs))
	}

	dlqMsg := consumeOne(t, brokers, "usage.reported.dlq", 20*time.Second)
	if dlqMsg == nil {
		t.Fatalf("expected DLQ message, got none")
	}

	var payload struct {
		RequestID string `json:"request_id"`
	}
	_ = json.Unmarshal(dlqMsg.Value, &payload)
	if payload.RequestID != evt.RequestID {
		t.Fatalf("dlq request_id mismatch: got %s want %s", payload.RequestID, evt.RequestID)
	}

}

// Helpers

func startUsageConsumer(t *testing.T, brokers []string, initialCredits int64) (*pgrepo.WalletRepository, *walletapp.Service, context.CancelFunc) {
	t.Helper()
	db := newSQLite(t)

	seed := []pgrepo.PricingPlan{{PlanID: "starter", IsDefault: true, CallCents: 5, MinuteCents: 10, MessageCents: 2, APIRequestCents: 1, StorageGBCents: 15}}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed pricing: %v", err)
	}

	pricingCfg := config.PricingConfig{DefaultPlan: "starter", Plans: map[string]config.PlanPricing{
		"starter": {CallCents: 5, MinuteCents: 10, MessageCents: 2, APIRequestCents: 1, StorageGBCents: 15},
	}}
	configPricing := walletapp.NewConfigPricingTable(pricingCfg)
	pricingTable := pgrepo.NewPricingTable(db, configPricing)
	repo := pgrepo.NewWalletRepository(db)
	svc := walletapp.NewService(repo, "USD", initialCredits, pricingTable)

	cfg := config.KafkaConfig{
		Brokers:        brokers,
		GroupID:        "billing-usage-consumer-it-" + uuid.NewString(),
		DLQTopic:       "usage.reported.dlq",
		MaxRetries:     0,
		RetryBackoffMs: 50,
	}

	uc, err := kafka.NewUsageConsumer(cfg, svc, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatalf("create usage consumer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	uc.Start(ctx)

	return repo, svc, func() {
		cancel()
		_ = uc.Close()
	}
}

func newSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(gormsqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.Wallet{}, &domain.WalletTransaction{}, &pgrepo.PricingPlan{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func produceUsage(t *testing.T, producer sarama.SyncProducer, evt walletapp.UsageReported) {
	payload, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal evt: %v", err)
	}
	msg := &sarama.ProducerMessage{Topic: "usage.reported", Value: sarama.ByteEncoder(payload)}
	if _, _, err := producer.SendMessage(msg); err != nil {
		t.Fatalf("produce message: %v", err)
	}
}

func mustProducer(t *testing.T, brokers []string) sarama.SyncProducer {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		t.Fatalf("new producer: %v", err)
	}
	return producer
}

func testBrokers() []string {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9093"
	}
	return []string{brokers}
}

func waitFor(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

func consumeOne(t *testing.T, brokers []string, topic string, timeout time.Duration) *sarama.ConsumerMessage {
	t.Helper()
	consumer, err := sarama.NewConsumer(brokers, nil)
	if err != nil {
		t.Fatalf("new consumer: %v", err)
	}
	defer consumer.Close()

	pc, err := consumer.ConsumePartition(topic, 0, sarama.OffsetOldest)
	if err != nil {
		t.Fatalf("partition consumer: %v", err)
	}
	defer pc.Close()

	select {
	case msg := <-pc.Messages():
		return msg
	case <-time.After(timeout):
		return nil
	}
}
