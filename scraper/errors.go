package scraper

import "fmt"

// ErrorPhase indicates at which phase the error occurred
type ErrorPhase string

const (
	PhaseInit             ErrorPhase = "INIT"              // Initialization/setup
	PhaseHomepage         ErrorPhase = "HOMEPAGE"          // Fetching homepage
	PhaseScriptExtract    ErrorPhase = "SCRIPT_EXTRACT"    // Extracting script URL from HTML
	PhaseScriptFetch      ErrorPhase = "SCRIPT_FETCH"      // Downloading the script
	PhaseProviderCall     ErrorPhase = "PROVIDER_CALL"     // Calling sensor provider API
	PhaseSensorPost       ErrorPhase = "SENSOR_POST"       // Posting sensor to Akamai
	PhaseCookieValidation ErrorPhase = "COOKIE_VALIDATION" // Validating _abck cookie
	PhaseSBSDPost         ErrorPhase = "SBSD_POST"         // Posting SBSD challenge
	PhaseTLSAPI           ErrorPhase = "TLS_API"           // TLS-API communication error
)

// SolverError represents a structured error with context
type SolverError struct {
	Phase      ErrorPhase // Which phase failed
	Step       string     // Specific step description
	Provider   string     // Provider involved (if applicable)
	Domain     string     // Target domain
	StatusCode int        // HTTP status code (if applicable)
	RawError   string     // Original error message
	Retryable  bool       // Whether the operation can be retried
}

// Error implements the error interface
func (e *SolverError) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("[%s] %s (provider=%s): %s", e.Phase, e.Step, e.Provider, e.RawError)
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("[%s] %s (status=%d): %s", e.Phase, e.Step, e.StatusCode, e.RawError)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Phase, e.Step, e.RawError)
}

// NewError creates a new SolverError
func NewError(phase ErrorPhase, step string, err error) *SolverError {
	rawErr := ""
	if err != nil {
		rawErr = err.Error()
	}
	return &SolverError{
		Phase:     phase,
		Step:      step,
		RawError:  rawErr,
		Retryable: isRetryablePhase(phase),
	}
}

// NewErrorWithStatus creates a SolverError with HTTP status code
func NewErrorWithStatus(phase ErrorPhase, step string, statusCode int, err error) *SolverError {
	e := NewError(phase, step, err)
	e.StatusCode = statusCode
	e.Retryable = isRetryableStatus(statusCode)
	return e
}

// NewErrorWithProvider creates a SolverError with provider info
func NewErrorWithProvider(phase ErrorPhase, step string, provider string, err error) *SolverError {
	e := NewError(phase, step, err)
	e.Provider = provider
	return e
}

// WithDomain adds domain context to the error
func (e *SolverError) WithDomain(domain string) *SolverError {
	e.Domain = domain
	return e
}

// WithRetryable sets the retryable flag
func (e *SolverError) WithRetryable(retryable bool) *SolverError {
	e.Retryable = retryable
	return e
}

// isRetryablePhase determines if errors in this phase are generally retryable
func isRetryablePhase(phase ErrorPhase) bool {
	switch phase {
	case PhaseHomepage, PhaseScriptFetch, PhaseProviderCall, PhaseSensorPost, PhaseSBSDPost:
		return true
	case PhaseInit, PhaseScriptExtract, PhaseCookieValidation:
		return false
	default:
		return false
	}
}

// isRetryableStatus determines if an HTTP status code is retryable
func isRetryableStatus(status int) bool {
	switch status {
	case 408, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// IsRetryable checks if the error can be retried
func IsRetryable(err error) bool {
	if se, ok := err.(*SolverError); ok {
		return se.Retryable
	}
	return false
}

// WrapError wraps a generic error into a SolverError
func WrapError(phase ErrorPhase, step string, err error) error {
	if err == nil {
		return nil
	}
	if se, ok := err.(*SolverError); ok {
		return se
	}
	return NewError(phase, step, err)
}
