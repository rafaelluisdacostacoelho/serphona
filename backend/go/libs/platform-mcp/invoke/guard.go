package invoke

import (
	"context"
	"errors"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// GuardConfig configures payload and timeout enforcement before invoking an Executor.
type GuardConfig struct {
	MaxBodyBytes   int64         // rejects requests with input larger than this; 0 disables
	DefaultTimeout time.Duration // applied when request has no timeout; 0 disables
	MaxTimeout     time.Duration // caps request-provided timeout; 0 disables
}

// GuardExecutor enforces payload and timeout limits before delegating to an Executor.
type GuardExecutor struct {
	inner Executor
	cfg   GuardConfig
}

// NewGuardExecutor wraps an Executor with payload/timeout guards.
func NewGuardExecutor(inner Executor, cfg GuardConfig) *GuardExecutor {
	return &GuardExecutor{inner: inner, cfg: cfg}
}

// Invoke validates size/time limits then forwards to the inner executor.
func (g *GuardExecutor) Invoke(ctx context.Context, req protocol.InvocationRequest) (<-chan protocol.InvocationEvent, error) {
	if g.cfg.MaxBodyBytes > 0 && int64(len(req.Input)) > g.cfg.MaxBodyBytes {
		return nil, errors.New("payload exceeds max body bytes")
	}

	effectiveTimeout := req.Timeout
	if effectiveTimeout == 0 && g.cfg.DefaultTimeout > 0 {
		effectiveTimeout = g.cfg.DefaultTimeout
	}
	if g.cfg.MaxTimeout > 0 && effectiveTimeout > g.cfg.MaxTimeout {
		effectiveTimeout = g.cfg.MaxTimeout
	}

	if effectiveTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, effectiveTimeout)
		// Avoid leaking context; caller owns returned channel, so cancel on ctx.Done.
		go func() {
			<-ctx.Done()
			cancel()
		}()
	}

	return g.inner.Invoke(ctx, req)
}
