package quota

import (
	"fmt"
	"sync"
	"time"
)

// Rule describes a quota constraint with optional scoping fields.
type Rule struct {
	TenantID       string
	ToolID         string
	AgentID        string
	Env            string
	LimitPerMinute *int
	LimitPerDay    *int
	Weight         int
}

// Request captures the attributes used to match a quota rule.
type Request struct {
	TenantID string
	ToolID   string
	AgentID  string
	Env      string
}

// Decision reports the outcome of a quota evaluation.
type Decision struct {
	Allowed         bool
	Reason          string
	RemainingMinute *int
	RemainingDay    *int
}

type window struct {
	minuteStart time.Time
	minuteCount int
	dayStart    time.Time
	dayCount    int
}

// Limiter keeps per-key counters with simple fixed windows.
type Limiter struct {
	mu      sync.Mutex
	windows map[string]*window
}

// NewLimiter constructs a Limiter.
func NewLimiter() *Limiter {
	return &Limiter{windows: make(map[string]*window)}
}

// Evaluate picks the first matching rule (ordered by weight externally) and applies counters.
func (l *Limiter) Evaluate(now time.Time, req Request, rules []Rule) Decision {
	rule, ok := matchRule(req, rules)
	if !ok {
		return Decision{Allowed: true, Reason: "no_rule"}
	}

	key := fmt.Sprintf("%s|%s|%s|%s", req.TenantID, req.ToolID, req.AgentID, req.Env)

	l.mu.Lock()
	w, exists := l.windows[key]
	if !exists {
		w = &window{minuteStart: now, dayStart: now}
		l.windows[key] = w
	}

	// reset windows if expired
	if now.Sub(w.minuteStart) >= time.Minute {
		w.minuteStart = now
		w.minuteCount = 0
	}
	if now.Sub(w.dayStart) >= 24*time.Hour {
		w.dayStart = now
		w.dayCount = 0
	}

	// evaluate limits
	if rule.LimitPerMinute != nil {
		if w.minuteCount >= *rule.LimitPerMinute {
			remaining := 0
			l.mu.Unlock()
			return Decision{Allowed: false, Reason: "minute_quota_exceeded", RemainingMinute: &remaining}
		}
	}
	if rule.LimitPerDay != nil {
		if w.dayCount >= *rule.LimitPerDay {
			remaining := 0
			l.mu.Unlock()
			return Decision{Allowed: false, Reason: "daily_quota_exceeded", RemainingDay: &remaining}
		}
	}

	// increment counts
	if rule.LimitPerMinute != nil {
		w.minuteCount++
	}
	if rule.LimitPerDay != nil {
		w.dayCount++
	}

	// compute remaining after increments
	var remainingMinute *int
	var remainingDay *int
	if rule.LimitPerMinute != nil {
		val := *rule.LimitPerMinute - w.minuteCount
		if val < 0 {
			val = 0
		}
		remainingMinute = &val
	}
	if rule.LimitPerDay != nil {
		val := *rule.LimitPerDay - w.dayCount
		if val < 0 {
			val = 0
		}
		remainingDay = &val
	}

	l.mu.Unlock()

	return Decision{Allowed: true, Reason: "ok", RemainingMinute: remainingMinute, RemainingDay: remainingDay}
}

func matchRule(req Request, rules []Rule) (Rule, bool) {
	for _, r := range rules {
		if r.TenantID != "" && r.TenantID != req.TenantID {
			continue
		}
		if r.ToolID != "" && r.ToolID != req.ToolID {
			continue
		}
		if r.AgentID != "" && r.AgentID != req.AgentID {
			continue
		}
		if r.Env != "" && r.Env != req.Env {
			continue
		}
		return r, true
	}
	return Rule{}, false
}
