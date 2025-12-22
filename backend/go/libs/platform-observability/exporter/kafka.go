package exporter

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/serphona/backend/go/libs/platform-observability/config"
)

// KafkaExporter sends observability events to Kafka.
type KafkaExporter struct {
	writer *kafka.Writer
}

// NewKafka creates a new Kafka exporter when enabled.
func NewKafka(cfg *config.Config) (*KafkaExporter, error) {
	if !cfg.KafkaEnabled || len(cfg.KafkaBrokers) == 0 {
		return nil, nil
	}

	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		ClientID:  cfg.KafkaClientID,
	}

	if cfg.KafkaTLSEnabled {
		dialer.TLS = &tls.Config{InsecureSkipVerify: cfg.KafkaTLSInsecure}
	}

	if cfg.KafkaSASLUser != "" && cfg.KafkaSASLPass != "" {
		switch cfg.KafkaSASLMechanism {
		case "", "plain", "PLAIN":
			dialer.SASLMechanism = plain.Mechanism{Username: cfg.KafkaSASLUser, Password: cfg.KafkaSASLPass}
		default:
			return nil, errors.New("unsupported SASL mechanism")
		}
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBrokers...),
		Topic:        cfg.KafkaTopic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		RequiredAcks: kafka.RequireOne,
		Transport: &kafka.Transport{
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, address)
			},
		},
	}

	return &KafkaExporter{writer: w}, nil
}

// Export serializes the event to JSON and publishes it.
func (k *KafkaExporter) Export(ctx context.Context, event any) error {
	if k == nil || k.writer == nil {
		return nil
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return k.writer.WriteMessages(ctx, kafka.Message{Value: payload})
}

// Close closes the exporter.
func (k *KafkaExporter) Close(ctx context.Context) error {
	if k == nil || k.writer == nil {
		return nil
	}
	return k.writer.Close()
}
