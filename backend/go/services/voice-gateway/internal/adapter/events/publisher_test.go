package events

import (
	"context"
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
