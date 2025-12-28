package invoke

// StackConfig defines how to compose the executor stack.
type StackConfig struct {
	Guard          *GuardConfig
	Resilient      *ResilientConfig
	RateLimiter    RateLimiter
	Observer       Observer
	MetricsSink    MetricsSink
	AuditSink      AuditSink
	LabelEnrichers []LabelEnricher
	ExtraObservers []Observer
	MaxOutput      int64
	EnableCancel   bool
}

// BuildExecutor wires the standard stack in order:
// Guard -> Resilient -> Cancelable (optional) -> OutputLimit -> Observed -> base.
func BuildExecutor(base Executor, cfg StackConfig) Executor {
	exec := base
	if cfg.Guard != nil {
		exec = NewGuardExecutor(exec, *cfg.Guard)
	}
	if cfg.RateLimiter != nil {
		exec = NewRateLimitExecutor(exec, cfg.RateLimiter, 0)
	}
	if cfg.Resilient != nil {
		exec = NewResilientExecutor(exec, *cfg.Resilient)
	}
	if cfg.EnableCancel {
		exec = NewCancelableExecutor(exec)
	}
	if cfg.MaxOutput > 0 {
		exec = NewOutputLimitExecutor(exec, cfg.MaxOutput)
	}
	observers := []Observer{}
	if cfg.MetricsSink != nil {
		observers = append(observers, NewMetricsObserver(cfg.MetricsSink, cfg.LabelEnrichers...))
	}
	if cfg.AuditSink != nil {
		observers = append(observers, NewAuditObserver(cfg.AuditSink))
	}
	if cfg.Observer != nil {
		observers = append(observers, cfg.Observer)
	}
	if len(cfg.ExtraObservers) > 0 {
		observers = append(observers, cfg.ExtraObservers...)
	}
	if len(observers) > 0 {
		exec = NewObservedExecutor(exec, NewMultiObserver(observers...))
	}
	return exec
}
