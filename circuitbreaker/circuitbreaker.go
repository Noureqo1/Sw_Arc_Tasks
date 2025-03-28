package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mutex sync.RWMutex

	state                 State
	failureThreshold      int
	failureCount          int
	resetTimeout          time.Duration
	lastStateChangeTime   time.Time
	halfOpenSuccessNeeded int
	halfOpenSuccessCount  int

	// Configurable settings
	maxRetries     int
	baseRetryDelay time.Duration
	maxRetryDelay  time.Duration
}

type Options struct {
	FailureThreshold      int
	ResetTimeout          time.Duration
	HalfOpenSuccessNeeded int
	MaxRetries            int
	BaseRetryDelay        time.Duration
	MaxRetryDelay         time.Duration
}

func NewCircuitBreaker(options Options) *CircuitBreaker {
	return &CircuitBreaker{
		state:                 StateClosed,
		failureThreshold:      options.FailureThreshold,
		resetTimeout:          options.ResetTimeout,
		halfOpenSuccessNeeded: options.HalfOpenSuccessNeeded,
		maxRetries:            options.MaxRetries,
		baseRetryDelay:        options.BaseRetryDelay,
		maxRetryDelay:         options.MaxRetryDelay,
		lastStateChangeTime:   time.Now(),
	}
}

func (cb *CircuitBreaker) Execute(operation func() error) error {
	if !cb.allowRequest() {
		return errors.New("circuit breaker is open")
	}

	var lastErr error
	for attempt := 0; attempt <= cb.maxRetries; attempt++ {
		err := operation()
		if err == nil {
			cb.recordSuccess()
			return nil
		}

		lastErr = err
		cb.recordFailure()

		if attempt < cb.maxRetries {
			delay := cb.calculateBackoff(attempt)
			time.Sleep(delay)
		}
	}

	return lastErr
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastStateChangeTime) > cb.resetTimeout {
			cb.mutex.RUnlock()
			cb.mutex.Lock()
			cb.toHalfOpen()
			cb.mutex.Unlock()
			cb.mutex.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) calculateBackoff(attempt int) time.Duration {
	backoff := cb.baseRetryDelay * time.Duration(1<<uint(attempt))
	if backoff > cb.maxRetryDelay {
		backoff = cb.maxRetryDelay
	}
	return backoff
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case StateHalfOpen:
		cb.halfOpenSuccessCount++
		if cb.halfOpenSuccessCount >= cb.halfOpenSuccessNeeded {
			cb.toClosed()
		}
	case StateClosed:
		cb.failureCount = 0
	}
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount++

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.failureThreshold {
			cb.toOpen()
		}
	case StateHalfOpen:
		cb.toOpen()
	}
}

func (cb *CircuitBreaker) toOpen() {
	cb.state = StateOpen
	cb.lastStateChangeTime = time.Now()
}

func (cb *CircuitBreaker) toHalfOpen() {
	cb.state = StateHalfOpen
	cb.halfOpenSuccessCount = 0
	cb.lastStateChangeTime = time.Now()
}

func (cb *CircuitBreaker) toClosed() {
	cb.state = StateClosed
	cb.failureCount = 0
	cb.lastStateChangeTime = time.Now()
}

func (cb *CircuitBreaker) GetState() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}
