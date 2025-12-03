// Package asterisk provides Asterisk ARI client implementations.
package asterisk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ARIClient manages connection to Asterisk REST Interface.
type ARIClient struct {
	baseURL    string
	username   string
	password   string
	appName    string
	logger     *zap.Logger
	httpClient *http.Client

	// WebSocket connection for events
	conn           *websocket.Conn
	reconnectDelay time.Duration
	maxReconnects  int
}

// NewARIClient creates a new Asterisk ARI client.
func NewARIClient(baseURL, username, password, appName string, logger *zap.Logger) *ARIClient {
	return &ARIClient{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		username: username,
		password: password,
		appName:  appName,
		logger:   logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		reconnectDelay: 5 * time.Second,
		maxReconnects:  10,
	}
}

// Connect establishes WebSocket connection to Asterisk ARI for events.
func (c *ARIClient) Connect(ctx context.Context) error {
	// Build WebSocket URL: ws://asterisk:8088/ari/events?app=appName&api_key=username:password
	wsURL := strings.Replace(c.baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = fmt.Sprintf("%s/events?app=%s&api_key=%s:%s",
		wsURL, c.appName, c.username, c.password)

	c.logger.Info("connecting to Asterisk ARI WebSocket",
		zap.String("app", c.appName),
	)

	// Establish WebSocket connection
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		if resp != nil {
			c.logger.Error("websocket handshake failed",
				zap.Int("status", resp.StatusCode),
				zap.Error(err),
			)
		}
		return fmt.Errorf("failed to connect to ARI WebSocket: %w", err)
	}

	c.conn = conn
	c.logger.Info("connected to Asterisk ARI WebSocket")

	return nil
}

// Close closes the ARI connection.
func (c *ARIClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// AnswerChannel answers an incoming channel.
func (c *ARIClient) AnswerChannel(ctx context.Context, channelID string) error {
	url := fmt.Sprintf("%s/channels/%s/answer", c.baseURL, channelID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to answer channel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("answer failed with status: %d", resp.StatusCode)
	}

	c.logger.Info("channel answered", zap.String("channel_id", channelID))
	return nil
}

// PlaybackStart starts audio playback on a channel.
func (c *ARIClient) PlaybackStart(ctx context.Context, channelID string, media string) (string, error) {
	url := fmt.Sprintf("%s/channels/%s/play?media=%s", c.baseURL, channelID, media)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to start playback: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("playback failed with status: %d", resp.StatusCode)
	}

	c.logger.Info("playback started",
		zap.String("channel_id", channelID),
		zap.String("media", media),
	)

	// Return a generated playback ID (in real ARI, this comes from response)
	return fmt.Sprintf("playback-%d", time.Now().Unix()), nil
}

// HangupChannel hangs up a channel.
func (c *ARIClient) HangupChannel(ctx context.Context, channelID string) error {
	url := fmt.Sprintf("%s/channels/%s", c.baseURL, channelID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to hangup channel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hangup failed with status: %d", resp.StatusCode)
	}

	c.logger.Info("channel hung up", zap.String("channel_id", channelID))
	return nil
}

// CreateBridge creates a new mixing bridge.
func (c *ARIClient) CreateBridge(ctx context.Context, bridgeType string) (string, error) {
	url := fmt.Sprintf("%s/bridges?type=%s", c.baseURL, bridgeType)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create bridge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create bridge failed with status: %d", resp.StatusCode)
	}

	c.logger.Info("bridge created", zap.String("type", bridgeType))

	// Return a generated bridge ID (in real ARI, parse from response)
	return fmt.Sprintf("bridge-%d", time.Now().Unix()), nil
}

// AddChannelToBridge adds a channel to a bridge.
func (c *ARIClient) AddChannelToBridge(ctx context.Context, bridgeID, channelID string) error {
	url := fmt.Sprintf("%s/bridges/%s/addChannel?channel=%s", c.baseURL, bridgeID, channelID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add channel to bridge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("add channel failed with status: %d", resp.StatusCode)
	}

	c.logger.Info("channel added to bridge",
		zap.String("bridge_id", bridgeID),
		zap.String("channel_id", channelID),
	)
	return nil
}

// ARIEvent represents an event from Asterisk ARI.
type ARIEvent struct {
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Channel   *ARIChannel            `json:"channel,omitempty"`
	Data      map[string]interface{} `json:"-"`
}

// ARIChannel represents a channel in ARI events.
type ARIChannel struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Caller struct {
		Number string `json:"number"`
		Name   string `json:"name"`
	} `json:"caller"`
	Connected struct {
		Number string `json:"number"`
		Name   string `json:"name"`
	} `json:"connected"`
}

// ListenForEvents listens for ARI events with automatic reconnection.
func (c *ARIClient) ListenForEvents(ctx context.Context, handler func(*ARIEvent) error) error {
	c.logger.Info("starting ARI event listener")

	reconnectAttempts := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Ensure connection
		if c.conn == nil {
			if err := c.Connect(ctx); err != nil {
				c.logger.Error("failed to connect to ARI", zap.Error(err))
				reconnectAttempts++

				if reconnectAttempts >= c.maxReconnects {
					return fmt.Errorf("max reconnection attempts reached: %d", c.maxReconnects)
				}

				c.logger.Info("retrying connection",
					zap.Int("attempt", reconnectAttempts),
					zap.Duration("delay", c.reconnectDelay),
				)

				select {
				case <-time.After(c.reconnectDelay):
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			reconnectAttempts = 0
		}

		// Read event from WebSocket
		var event ARIEvent
		err := c.conn.ReadJSON(&event)
		if err != nil {
			c.logger.Error("failed to read event", zap.Error(err))

			// Close connection to trigger reconnect
			c.conn.Close()
			c.conn = nil

			continue
		}

		// Reset reconnect counter on successful read
		reconnectAttempts = 0

		// Handle event
		if err := handler(&event); err != nil {
			c.logger.Error("event handler error",
				zap.String("event_type", event.Type),
				zap.Error(err),
			)
			// Continue processing other events even if handler fails
		}
	}
}

// GetChannelInfo retrieves channel information.
func (c *ARIClient) GetChannelInfo(ctx context.Context, channelID string) (*ARIChannel, error) {
	url := fmt.Sprintf("%s/channels/%s", c.baseURL, channelID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("channel not found: %s", channelID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get channel failed with status: %d", resp.StatusCode)
	}

	var channel ARIChannel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		return nil, fmt.Errorf("failed to decode channel info: %w", err)
	}

	return &channel, nil
}
