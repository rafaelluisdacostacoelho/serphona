package resilience

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests")
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name            string
	maxFailures     uint32
	timeout         time.Duration
	halfOpenTimeout time.Duration

	mu              sync.RWMutex
	state           State
	failures        uint32
	successes       uint32
	lastFailureTime time.Time
	lastStateChange time.Time
}

// Config holds circuit breaker configuration
type Config struct {
	Name            string
	MaxFailures     uint32        // number of failures before opening
	Timeout         time.Duration // how long to wait in open state
	HalfOpenTimeout time.Duration // how long to wait in half-open state
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config Config) *CircuitBreaker {
	if config.MaxFailures == 0 {
		config.MaxFailures = 5
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.HalfOpenTimeout == 0 {
		config.HalfOpenTimeout = 30 * time.Second
	}

	return &CircuitBreaker{
		name:            config.Name,
		maxFailures:     config.MaxFailures,
		timeout:         config.Timeout,
		halfOpenTimeout: config.HalfOpenTimeout,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Check if we can proceed
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.afterRequest(err)

	return err
}

// beforeRequest checks if the request can proceed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		// Allow request
		return nil

	case StateOpen:
		// Check if timeout has elapsed
		if now.Sub(cb.lastStateChange) >= cb.timeout {
			// Transition to half-open
			cb.state = StateHalfOpen
			cb.lastStateChange = now
			cb.successes = 0
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// Allow limited requests in half-open state
		if cb.successes > 0 {
			return ErrTooManyRequests
		}
		return nil

	default:
		return nil
	}
}

// afterRequest records the result of the request
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	if err != nil {
		// Failure
		cb.failures++
		cb.lastFailureTime = now

		switch cb.state {
		case StateClosed:
			if cb.failures >= cb.maxFailures {
				cb.state = StateOpen
				cb.lastStateChange = now
			}

		case StateHalfOpen:
			// Immediately go back to open on any failure
			cb.state = StateOpen
			cb.lastStateChange = now
			cb.failures = 1
		}
	} else {
		// Success
		cb.successes++

		switch cb.state {
		case StateClosed:
			// Reset failure count on success
			if cb.failures > 0 {
				cb.failures = 0
			}

		case StateHalfOpen:
			// Check if we can close the circuit
			if now.Sub(cb.lastStateChange) >= cb.halfOpenTimeout || cb.successes >= 3 {
				cb.state = StateClosed
				cb.lastStateChange = now
				cb.failures = 0
				cb.successes = 0
			}
		}
	}
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Failures returns the current failure count
func (cb *CircuitBreaker) Failures() uint32 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// Name returns the circuit breaker name
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.lastStateChange = time.Now()
}
