package anomaly

import (
    "math"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/serphona/backend/go/libs/platform-observability/types"
)

// Detector identifica anomalias simples via z-score em janelas temporais.
type Detector struct {
    bucketSize      time.Duration
    windowBuckets   int
    zThreshold      float64
    minCount        int
    cooldown        time.Duration

    mu      sync.Mutex
    series  map[string]*series
    lastAlert map[string]time.Time
}

type series struct {
    buckets []bucket
}

type bucket struct {
    start time.Time
    count int
}

// NewDetector cria um detector com janela e limiares configurados.
func NewDetector(bucketMinutes, windowMinutes int, zThreshold float64, minCount int, cooldownSeconds int) *Detector {
    if bucketMinutes <= 0 {
        bucketMinutes = 1
    }
    if windowMinutes <= 0 {
        windowMinutes = 5
    }
    if zThreshold <= 0 {
        zThreshold = 3
    }
    if minCount <= 0 {
        minCount = 5
    }
    if cooldownSeconds <= 0 {
        cooldownSeconds = 300
    }

    return &Detector{
        bucketSize:    time.Duration(bucketMinutes) * time.Minute,
        windowBuckets: int(math.Max(1, float64(windowMinutes)/float64(bucketMinutes))),
        zThreshold:    zThreshold,
        minCount:      minCount,
        cooldown:      time.Duration(cooldownSeconds) * time.Second,
        series:        make(map[string]*series),
        lastAlert:     make(map[string]time.Time),
    }
}

// Observe registra um evento e retorna um alerta se for considerado anômalo.
func (d *Detector) Observe(key, tenant string, ts time.Time) *types.AlertEvent {
    d.mu.Lock()
    defer d.mu.Unlock()

    s := d.series[key]
    if s == nil {
        s = &series{}
        d.series[key] = s
    }

    d.advanceBuckets(s, ts)
    if len(s.buckets) == 0 {
        s.buckets = append(s.buckets, bucket{start: d.bucketStart(ts), count: 0})
    }

    // Incrementar bucket corrente
    s.buckets[len(s.buckets)-1].count++

    // Avaliar z-score
    counts := make([]float64, len(s.buckets))
    for i, b := range s.buckets {
        counts[i] = float64(b.count)
    }

    if len(counts) < 2 { // precisa de histórico
        return nil
    }

    mean, std := stats(counts)
    current := counts[len(counts)-1]
    if current < float64(d.minCount) {
        return nil
    }
    if std == 0 {
        if current <= mean*2 { // sem variação e sem pico
            return nil
        }
    }

    var z float64
    if std > 0 {
        z = (current - mean) / std
    } else {
        z = current - mean
    }

    if z < d.zThreshold {
        return nil
    }

    lastKey := key + ":" + tenant
    if last, ok := d.lastAlert[lastKey]; ok {
        if ts.Sub(last) < d.cooldown {
            return nil
        }
    }
    d.lastAlert[lastKey] = ts

    return &types.AlertEvent{
        AlertID:        uuid.New().String(),
        TenantID:       tenant,
        EventType:      "anomaly.detected",
        Source:         "anomaly-detector",
        Severity:       "warning",
        Message:        "anomaly detected for key " + key,
        Score:          z,
        Current:        int(current),
        Mean:           mean,
        Std:            std,
        WindowMinutes:  d.windowBuckets * int(d.bucketSize/time.Minute),
        Key:            key,
        Timestamp:      ts,
    }
}

func (d *Detector) advanceBuckets(s *series, ts time.Time) {
    if len(s.buckets) == 0 {
        s.buckets = append(s.buckets, bucket{start: d.bucketStart(ts), count: 0})
        return
    }
    currentStart := d.bucketStart(ts)
    lastIdx := len(s.buckets) - 1
    if currentStart.Equal(s.buckets[lastIdx].start) {
        return
    }

    // adicionar buckets intermediários se necessário
    for start := s.buckets[lastIdx].start.Add(d.bucketSize); !start.After(currentStart); start = start.Add(d.bucketSize) {
        s.buckets = append(s.buckets, bucket{start: start, count: 0})
    }

    // manter janela
    if len(s.buckets) > d.windowBuckets {
        s.buckets = s.buckets[len(s.buckets)-d.windowBuckets:]
    }
}

func (d *Detector) bucketStart(ts time.Time) time.Time {
    minutes := ts.Truncate(d.bucketSize)
    return minutes
}

func stats(values []float64) (mean, std float64) {
    if len(values) == 0 {
        return 0, 0
    }
    var sum float64
    for _, v := range values {
        sum += v
    }
    mean = sum / float64(len(values))

    var variance float64
    for _, v := range values {
        variance += (v - mean) * (v - mean)
    }
    variance = variance / float64(len(values))
    std = math.Sqrt(variance)
    return
}
