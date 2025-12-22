package anomaly

import (
	"testing"
	"time"
)

func TestDetectorSpikeGeneratesAlert(t *testing.T) {
	d := NewDetector(1, 5, 1.5, 3, 60)
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	// steady low counts
	for i := 0; i < 3; i++ {
		ts := base.Add(time.Duration(i) * time.Minute)
		if alert := d.Observe("interaction.agent", "tenant-1", ts); alert != nil {
			t.Fatalf("unexpected alert in baseline: %+v", alert)
		}
	}

	// spike bucket
	spikeTime := base.Add(4 * time.Minute)
	alerted := false
	for i := 0; i < 15; i++ {
		if alert := d.Observe("interaction.agent", "tenant-1", spikeTime); alert != nil {
			alerted = true
			break
		}
	}
	if !alerted {
		t.Fatalf("expected alert on spike")
	}
}

func TestDetectorCooldown(t *testing.T) {
	d := NewDetector(1, 5, 1.0, 3, 120)
	base := time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC)

	// warmup buckets
	for i := 0; i < 3; i++ {
		_ = d.Observe("decision.transfer", "tenant-1", base.Add(time.Duration(i)*time.Minute))
	}

	// first spike triggers alert
	spikeTime := base.Add(4 * time.Minute)
	var firstAlertTime time.Time
	for i := 0; i < 6; i++ {
		if alert := d.Observe("decision.transfer", "tenant-1", spikeTime); alert != nil {
			firstAlertTime = alert.Timestamp
			break
		}
	}
	if firstAlertTime.IsZero() {
		t.Fatalf("expected first alert")
	}

	// second spike within cooldown should NOT alert
	spike2Time := spikeTime.Add(1 * time.Minute)
	for i := 0; i < 6; i++ {
		if alert := d.Observe("decision.transfer", "tenant-1", spike2Time); alert != nil {
			t.Fatalf("expected cooldown to suppress alert, got %+v", alert)
		}
	}

	// after cooldown expires, should alert again
	spike3Time := spikeTime.Add(3 * time.Minute).Add(120 * time.Second)
	alerted := false
	for i := 0; i < 6; i++ {
		if alert := d.Observe("decision.transfer", "tenant-1", spike3Time); alert != nil {
			alerted = true
			break
		}
	}
	if !alerted {
		t.Fatalf("expected alert after cooldown")
	}
}
