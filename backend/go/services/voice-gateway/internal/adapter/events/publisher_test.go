package events

import (
	"context"
	"fmt"
	"testing"

	"github.com/IBM/sarama/mocks"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/domain/call"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func setupTestPublisher(t *testing.T) *kafkaPublisher {
	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndSucceed()
	return &kafkaPublisher{
		producer:    mockProducer,
		topicPrefix: "test",
		logger:      zap.NewNop(),
	}
}

func TestPublishCallStarted(t *testing.T) {
	publisher := setupTestPublisher(t)
	ctx := context.Background()
	c := &call.Call{}

	err := publisher.PublishCallStarted(ctx, c)
	assert.NoError(t, err)
}

func TestPublishCallAnswered(t *testing.T) {
	publisher := setupTestPublisher(t)
	ctx := context.Background()
	c := &call.Call{}

	err := publisher.PublishCallAnswered(ctx, c)
	assert.NoError(t, err)
}

func TestPublishCallTransferred(t *testing.T) {
	publisher := setupTestPublisher(t)
	ctx := context.Background()
	c := &call.Call{}

	err := publisher.PublishCallTransferred(ctx, c)
	assert.NoError(t, err)
}

func TestPublishCallEnded(t *testing.T) {
	publisher := setupTestPublisher(t)
	ctx := context.Background()
	c := &call.Call{}

	err := publisher.PublishCallEnded(ctx, c)
	assert.NoError(t, err)
}

func TestPublishEventSuccess(t *testing.T) {
	producer := mocks.NewSyncProducer(t, nil)
	producer.ExpectSendMessageAndSucceed()

	publisher := &kafkaPublisher{producer: producer, topicPrefix: "test", logger: zap.NewNop()}

	err := publisher.publishEvent(context.Background(), "custom.event", "key", map[string]string{"k": "v"})
	assert.NoError(t, err)
}

func TestPublishEventFailureWithDLQ(t *testing.T) {
	producer := mocks.NewSyncProducer(t, nil)
	producer.ExpectSendMessageAndFail(fmt.Errorf("primary send failed"))
	producer.ExpectSendMessageAndSucceed()

	publisher := &kafkaPublisher{producer: producer, topicPrefix: "test", logger: zap.NewNop()}

	err := publisher.publishEvent(context.Background(), "custom.event", "key", map[string]string{"k": "v"})
	assert.Error(t, err)
}
