package exporter

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/serphona/backend/go/libs/platform-observability/config"
)

// LokiExporter pushes observability events to Loki.
type LokiExporter struct {
    client   *http.Client
    endpoint string
    tenant   string
}

// NewLoki builds a Loki exporter when enabled.
func NewLoki(cfg *config.Config) (*LokiExporter, error) {
    if !cfg.LokiEnabled || cfg.LokiEndpoint == "" {
        return nil, nil
    }

    endpoint := strings.TrimSuffix(cfg.LokiEndpoint, "/") + "/loki/api/v1/push"

    return &LokiExporter{
        client:   &http.Client{Timeout: 5 * time.Second},
        endpoint: endpoint,
        tenant:   cfg.LokiTenant,
    }, nil
}

// Export sends a single event with provided labels.
func (l *LokiExporter) Export(ctx context.Context, labels map[string]string, event any) error {
    if l == nil || l.client == nil {
        return nil
    }

    payload, err := json.Marshal(event)
    if err != nil {
        return err
    }

    reqBody := lokiPush{Streams: []lokiStream{{
        Stream: labels,
        Values: [][]string{{fmt.Sprintf("%d", time.Now().UnixNano()), string(payload)}},
    }}}

    data, err := json.Marshal(reqBody)
    if err != nil {
        return err
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.endpoint, bytes.NewReader(data))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")
    if l.tenant != "" {
        req.Header.Set("X-Scope-OrgID", l.tenant)
    }

    resp, err := l.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= http.StatusMultipleChoices {
        return fmt.Errorf("loki push failed: %s", resp.Status)
    }

    return nil
}

type lokiPush struct {
    Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
    Stream map[string]string `json:"stream"`
    Values [][]string        `json:"values"`
}
