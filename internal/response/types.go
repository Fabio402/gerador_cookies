package response

import (
	"encoding/json"
	"net/http"
)

// CookieItem representa um cookie individual
type CookieItem struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
}

// Cookies contém todos os cookies gerados
type Cookies struct {
	FullString string       `json:"full_string"`
	Items      []CookieItem `json:"items"`
}

// Telemetry contém dados de telemetria
type Telemetry struct {
	AbckToken         string `json:"abck_token,omitempty"`
	BmSzEncoded       string `json:"bm_sz_encoded,omitempty"`
	BmSEncoded        string `json:"bm_s_encoded,omitempty"`
	SensorDataEncoded string `json:"sensor_data_encoded,omitempty"`
}

// Session contém metadados da sessão
type Session struct {
	Provider string `json:"provider"`
	Profile  string `json:"profile"`
	Attempts int    `json:"attempts,omitempty"`
}

// ErrorContext contém contexto adicional do erro
type ErrorContext struct {
	Attempt     int   `json:"attempt,omitempty"`
	MaxAttempts int   `json:"max_attempts,omitempty"`
	ElapsedMs   int64 `json:"elapsed_ms,omitempty"`
}

// ErrorDetail contém detalhes do erro
type ErrorDetail struct {
	Step        string        `json:"step"`
	StepNumber  int           `json:"step_number"`
	Description string        `json:"description"`
	Provider    string        `json:"provider,omitempty"`
	Domain      string        `json:"domain,omitempty"`
	HTTPStatus  int           `json:"http_status,omitempty"`
	RawError    string        `json:"raw_error"`
	Retryable   bool          `json:"retryable"`
	Context     *ErrorContext `json:"context,omitempty"`
}

// Debug contém informações de debug
type Debug struct {
	ReportPath string `json:"report_path,omitempty"`
}

// SuccessResponse é a resposta de sucesso padrão
type SuccessResponse struct {
	Success   bool       `json:"success"`
	Cookies   *Cookies   `json:"cookies"`
	Telemetry *Telemetry `json:"telemetry"`
	Session   *Session   `json:"session"`
}

// ErrorResponse é a resposta de erro padrão
type ErrorResponse struct {
	Success        bool         `json:"success"`
	Error          *ErrorDetail `json:"error"`
	PartialCookies *Cookies     `json:"partial_cookies,omitempty"`
	Debug          *Debug       `json:"debug,omitempty"`
}

// JSON helpers

func WriteSuccess(w http.ResponseWriter, resp *SuccessResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func WriteError(w http.ResponseWriter, statusCode int, resp *ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	if resp.Error != nil && resp.Error.Retryable {
		w.Header().Set("Retry-After", "1")
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}
