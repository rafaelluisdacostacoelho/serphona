// Package asterisk provides Asterisk ARI client implementations.
package asterisk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	authclient "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/client"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.uber.org/zap"
)

func applyTenantHeader(ctx context.Context, headers http.Header) http.Header {
	if tenantID, err := middleware.TenantIDFromContext(ctx); err == nil && tenantID != "" {
		return middleware.EnsureTenantHeader(headers, tenantID)
	}

	return headers
}

// ARIClientHTTP manages connection to Asterisk REST Interface using HTTP and WebSocket.
type ARIClientHTTP struct {
	baseURL    string
	username   string
	password   string
	appName    string
	httpClient *http.Client
	logger     *zap.Logger

	// WebSocket URL for future event subscription
	wsURL string
}

// NewARIClientHTTP creates a new Asterisk ARI client using HTTP/WebSocket.
func NewARIClientHTTP(config ARIConfig, logger *zap.Logger) (*ARIClientHTTP, error) {
	client := &ARIClientHTTP{
		baseURL:  config.URL,
		username: config.Username,
		password: config.Password,
		appName:  config.AppName,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: authclient.WithDefaultTransport(nil),
		},
		logger: logger,
	}

	// Parse WebSocket URL
	u, err := url.Parse(config.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	wsScheme := "ws"
	if u.Scheme == "https" {
		wsScheme = "wss"
	}
	client.wsURL = fmt.Sprintf("%s://%s/ari/events?app=%s&api_key=%s:%s",
		wsScheme, u.Host, config.AppName, config.Username, config.Password)

	logger.Info("ARI HTTP client initialized",
		zap.String("app", config.AppName),
		zap.String("url", config.URL),
	)

	return client, nil
}

// Close closes the ARI connection.
func (c *ARIClientHTTP) Close() error {
	c.httpClient.CloseIdleConnections()
	c.logger.Info("ARI HTTP client closed")
	return nil
}

// HealthCheck performs a lightweight ARI info call to verify availability.
func (c *ARIClientHTTP) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/ari/asterisk/info", c.baseURL), nil)
	if err != nil {
		return fmt.Errorf("failed to build health request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ari health request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ari health returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// AnswerChannel answers an incoming channel.
func (c *ARIClientHTTP) AnswerChannel(ctx context.Context, channelID string) error {
	endpoint := fmt.Sprintf("%s/ari/channels/%s/answer", c.baseURL, channelID)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to answer channel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("answer failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("channel answered", zap.String("channel_id", channelID))
	return nil
}

// HangupChannel hangs up a channel.
func (c *ARIClientHTTP) HangupChannel(ctx context.Context, channelID, reason string) error {
	endpoint := fmt.Sprintf("%s/ari/channels/%s?reason=%s", c.baseURL, channelID, reason)

	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to hangup channel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hangup failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("channel hung up",
		zap.String("channel_id", channelID),
		zap.String("reason", reason),
	)
	return nil
}

// PlaybackStart starts audio playback on a channel.
func (c *ARIClientHTTP) PlaybackStart(ctx context.Context, channelID, mediaURI string) (string, error) {
	endpoint := fmt.Sprintf("%s/ari/channels/%s/play?media=%s", c.baseURL, channelID, url.QueryEscape(mediaURI))

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to start playback: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("playback failed with status %d: %s", resp.StatusCode, string(body))
	}

	var playback struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&playback); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("playback started",
		zap.String("channel_id", channelID),
		zap.String("media", mediaURI),
		zap.String("playback_id", playback.ID),
	)

	return playback.ID, nil
}

// StopPlayback stops an active playback.
func (c *ARIClientHTTP) StopPlayback(ctx context.Context, playbackID string) error {
	endpoint := fmt.Sprintf("%s/ari/playbacks/%s", c.baseURL, playbackID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop playback: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop playback failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("playback stopped", zap.String("playback_id", playbackID))
	return nil
}

// CreateBridge creates a new mixing bridge.
func (c *ARIClientHTTP) CreateBridge(ctx context.Context, bridgeType string) (string, error) {
	endpoint := fmt.Sprintf("%s/ari/bridges?type=%s", c.baseURL, bridgeType)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create bridge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create bridge failed with status %d: %s", resp.StatusCode, string(body))
	}

	var bridge struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&bridge); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("bridge created",
		zap.String("bridge_id", bridge.ID),
		zap.String("type", bridgeType),
	)

	return bridge.ID, nil
}

// DestroyBridge destroys a bridge.
func (c *ARIClientHTTP) DestroyBridge(ctx context.Context, bridgeID string) error {
	endpoint := fmt.Sprintf("%s/ari/bridges/%s", c.baseURL, bridgeID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to destroy bridge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("destroy bridge failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("bridge destroyed", zap.String("bridge_id", bridgeID))
	return nil
}

// AddChannelToBridge adds a channel to a bridge.
func (c *ARIClientHTTP) AddChannelToBridge(ctx context.Context, bridgeID, channelID string) error {
	endpoint := fmt.Sprintf("%s/ari/bridges/%s/addChannel?channel=%s", c.baseURL, bridgeID, channelID)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add channel to bridge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add channel failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("channel added to bridge",
		zap.String("bridge_id", bridgeID),
		zap.String("channel_id", channelID),
	)
	return nil
}

// RemoveChannelFromBridge removes a channel from a bridge.
func (c *ARIClientHTTP) RemoveChannelFromBridge(ctx context.Context, bridgeID, channelID string) error {
	endpoint := fmt.Sprintf("%s/ari/bridges/%s/removeChannel?channel=%s", c.baseURL, bridgeID, channelID)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to remove channel from bridge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remove channel failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("channel removed from bridge",
		zap.String("bridge_id", bridgeID),
		zap.String("channel_id", channelID),
	)
	return nil
}

// StartRecording starts recording on a channel.
func (c *ARIClientHTTP) StartRecording(ctx context.Context, channelID, name string, options RecordingOptions) error {
	endpoint := fmt.Sprintf("%s/ari/channels/%s/record?name=%s&format=%s&maxDurationSeconds=%d&maxSilenceSeconds=%d",
		c.baseURL, channelID, name, options.Format, options.MaxDuration, options.MaxSilence)

	if options.Beep {
		endpoint += "&beep=true"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("start recording failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("recording started",
		zap.String("channel_id", channelID),
		zap.String("name", name),
	)
	return nil
}

// StopRecording stops an active recording.
func (c *ARIClientHTTP) StopRecording(ctx context.Context, recordingName string) error {
	endpoint := fmt.Sprintf("%s/ari/recordings/live/%s/stop", c.baseURL, recordingName)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop recording: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop recording failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("recording stopped", zap.String("recording", recordingName))
	return nil
}

// GetChannelInfo retrieves channel information.
func (c *ARIClientHTTP) GetChannelInfo(ctx context.Context, channelID string) (*ChannelInfo, error) {
	endpoint := fmt.Sprintf("%s/ari/channels/%s", c.baseURL, channelID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get channel failed with status %d: %s", resp.StatusCode, string(body))
	}

	var channelData struct {
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
		Creationtime time.Time `json:"creationtime"`
		Language     string    `json:"language"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&channelData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ChannelInfo{
		ID:    channelData.ID,
		Name:  channelData.Name,
		State: channelData.State,
		Caller: CallerInfo{
			Number: channelData.Caller.Number,
			Name:   channelData.Caller.Name,
		},
		Connected: CallerInfo{
			Number: channelData.Connected.Number,
			Name:   channelData.Connected.Name,
		},
		CreationTime: channelData.Creationtime,
		Language:     channelData.Language,
	}, nil
}

// MuteChannel mutes a channel.
func (c *ARIClientHTTP) MuteChannel(ctx context.Context, channelID, direction string) error {
	endpoint := fmt.Sprintf("%s/ari/channels/%s/mute?direction=%s", c.baseURL, channelID, direction)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to mute channel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mute failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("channel muted",
		zap.String("channel_id", channelID),
		zap.String("direction", direction),
	)
	return nil
}

// UnmuteChannel unmutes a channel.
func (c *ARIClientHTTP) UnmuteChannel(ctx context.Context, channelID, direction string) error {
	endpoint := fmt.Sprintf("%s/ari/channels/%s/unmute?direction=%s", c.baseURL, channelID, direction)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to unmute channel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unmute failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("channel unmuted",
		zap.String("channel_id", channelID),
		zap.String("direction", direction),
	)
	return nil
}

// SendDTMF sends DTMF digits to a channel.
func (c *ARIClientHTTP) SendDTMF(ctx context.Context, channelID, dtmf string, opts DTMFOptions) error {
	endpoint := fmt.Sprintf("%s/ari/channels/%s/dtmf?dtmf=%s&duration=%d&between=%d",
		c.baseURL, channelID, dtmf, opts.Duration, opts.Between)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DTMF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send DTMF failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("DTMF sent",
		zap.String("channel_id", channelID),
		zap.String("dtmf", dtmf),
	)
	return nil
}

// GetChannelVariable gets a channel variable.
func (c *ARIClientHTTP) GetChannelVariable(ctx context.Context, channelID, variable string) (string, error) {
	endpoint := fmt.Sprintf("%s/ari/channels/%s/variable?variable=%s", c.baseURL, channelID, variable)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get variable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("get variable failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Value, nil
}

// SetChannelVariable sets a channel variable.
func (c *ARIClientHTTP) SetChannelVariable(ctx context.Context, channelID, variable, value string) error {
	endpoint := fmt.Sprintf("%s/ari/channels/%s/variable?variable=%s&value=%s",
		c.baseURL, channelID, variable, url.QueryEscape(value))

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	// Ensure tenant header is present when context carries tenant id
	req.Header = applyTenantHeader(ctx, req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to set variable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set variable failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("channel variable set",
		zap.String("channel_id", channelID),
		zap.String("variable", variable),
	)
	return nil
}

// StartExternalMedia starts an external media stream (placeholder).
func (c *ARIClientHTTP) StartExternalMedia(ctx context.Context, channelID string, opts ExternalMediaOptions) error {
	c.logger.Info("external media stream setup",
		zap.String("channel_id", channelID),
		zap.String("format", opts.Format),
	)
	return nil
}

// doRequest performs an HTTP request with basic auth.
func (c *ARIClientHTTP) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// ARIConfig holds configuration for ARI client.
type ARIConfig struct {
	URL      string
	Username string
	Password string
	AppName  string
}

// RecordingOptions holds recording configuration.
type RecordingOptions struct {
	Format      string
	MaxDuration int
	MaxSilence  int
	IfExists    string
	Beep        bool
	TerminateOn string
}

// ChannelInfo holds channel information.
type ChannelInfo struct {
	ID           string
	Name         string
	State        string
	Caller       CallerInfo
	Connected    CallerInfo
	CreationTime time.Time
	Language     string
}

// CallerInfo holds caller/connected information.
type CallerInfo struct {
	Number string
	Name   string
}

// DTMFOptions holds DTMF configuration.
type DTMFOptions struct {
	Before   int
	Between  int
	Duration int
	After    int
}

// ExternalMediaOptions holds external media configuration.
type ExternalMediaOptions struct {
	Format     string
	Direction  string
	SampleRate int
}

// ARIEvent represents a generic ARI event for webhook compatibility.
type ARIEvent struct {
	Type      string      `json:"type"`
	Timestamp string      `json:"timestamp"`
	Channel   *ARIChannel `json:"channel,omitempty"`
}

// ARIChannel represents channel data in webhook events.
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

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
