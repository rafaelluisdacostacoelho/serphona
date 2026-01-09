//go:build integration
// +build integration

package integration

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	pgDriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func init() {
	// Initialize database connection
	dsn := "host=localhost user=test password=test dbname=test port=55432 sslmode=disable"
	dbConn, err := gorm.Open(pgDriver.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	db = dbConn

	// Setup test data
	SetupTestData(db)
}

func TestTenantPropagation(t *testing.T) {
	topic := "test-tenant-propagation"
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}

	// Ensure the broker uses the correct port
	if broker == "localhost:9093" {
		broker = "localhost:9092"
	}

	// Initialize a valid context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use the context in Kafka operations
	conn, err := kafka.DialContext(ctx, "tcp", broker)
	if err != nil {
		t.Fatalf("failed to connect to Kafka broker: %v", err)
	}
	defer conn.Close()

	t.Log("Successfully connected to Kafka broker")

	// Ensure the topic exists
	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		t.Fatalf("failed to create Kafka topic: %v", err)
	}

	// Initialize Kafka writer with valid broker
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{broker},
		Topic:   topic,
	})
	defer writer.Close()

	// Verify broker connectivity
	conn, err = kafka.DialLeader(ctx, "tcp", broker, topic, 0)
	if err != nil {
		t.Fatalf("failed to connect to Kafka broker leader: %v", err)
	}
	defer conn.Close()

	// Send a test message
	message := kafka.Message{
		Key:   []byte("tenant_id"),
		Value: []byte("test-tenant"),
	}
	err = writer.WriteMessages(ctx, message)
	if err != nil {
		log.Fatalf("failed to write message: %v", err)
	}

	// Create Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   topic,
		GroupID: "test-group",
	})
	defer reader.Close()

	// Read the message
	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		log.Fatalf("failed to read message: %v", err)
	}

	// Validate the message
	assert.Equal(t, "test-tenant", string(msg.Value))
}
