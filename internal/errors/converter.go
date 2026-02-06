package errors

import (
	"gerador_cookies/internal/response"
)

// ToErrorResponse converte SolverError para ErrorResponse
func (e *SolverError) ToErrorResponse() *response.ErrorResponse {
	var ctx *response.ErrorContext
	if e.Attempt > 0 || e.MaxAttempts > 0 {
		ctx = &response.ErrorContext{
			Attempt:     e.Attempt,
			MaxAttempts: e.MaxAttempts,
			ElapsedMs:   e.ElapsedMs(),
		}
	}

	rawErrorMsg := ""
	if e.RawError != nil {
		rawErrorMsg = e.RawError.Error()
	}

	return &response.ErrorResponse{
		Success: false,
		Error: &response.ErrorDetail{
			Step:        string(e.Step),
			StepNumber:  e.StepNumber(),
			Description: e.Description(),
			Provider:    e.Provider,
			Domain:      e.Domain,
			HTTPStatus:  e.HTTPStatus(),
			RawError:    rawErrorMsg,
			Retryable:   e.IsRetryable(),
			Context:     ctx,
		},
	}
}

// WithPartialCookies adiciona cookies parciais à resposta de erro
func WithPartialCookies(resp *response.ErrorResponse, cookies *response.Cookies) *response.ErrorResponse {
	resp.PartialCookies = cookies
	return resp
}

// WithDebug adiciona informações de debug à resposta de erro
func WithDebug(resp *response.ErrorResponse, reportPath string) *response.ErrorResponse {
	if reportPath != "" {
		resp.Debug = &response.Debug{
			ReportPath: reportPath,
		}
	}
	return resp
}
