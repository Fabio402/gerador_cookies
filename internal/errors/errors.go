package errors

import (
	"fmt"
	"time"
)

// StepCode representa o código do step onde ocorreu o erro
type StepCode string

const (
	StepScraperInit      StepCode = "scraper_init"
	StepScriptURLExtract StepCode = "script_url_extraction"
	StepScriptFetch      StepCode = "script_fetch"
	StepScriptDecode     StepCode = "script_decode"
	StepBmSoExtraction   StepCode = "bm_so_extraction"
	StepProviderCall     StepCode = "provider_call"
	StepSensorPost       StepCode = "sensor_post"
	StepSbsdGeneration   StepCode = "sbsd_generation"
	StepSbsdPost         StepCode = "sbsd_post"
	StepCookieValidation StepCode = "cookie_validation"
	StepTLSAPIError      StepCode = "tls_api_error"
)

// stepInfo contém metadados de cada step
type stepInfo struct {
	Number      int
	Description string
	HTTPStatus  int
	Retryable   bool
}

var stepInfoMap = map[StepCode]stepInfo{
	StepScraperInit:      {1, "Falha ao criar cliente TLS", 518, false},
	StepScriptURLExtract: {2, "Script anti-bot não encontrado no HTML", 518, true},
	StepScriptFetch:      {3, "Falha ao baixar script anti-bot", 518, true},
	StepScriptDecode:     {4, "Falha ao decodificar script base64", 518, false},
	StepBmSoExtraction:   {5, "Cookie bm_so/sbsd_o não encontrado", 518, false},
	StepProviderCall:     {6, "Falha ao chamar API do provider", 518, true},
	StepSensorPost:       {7, "Akamai rejeitou sensor data", 518, true},
	StepSbsdGeneration:   {8, "Falha ao gerar challenge SbSd", 518, true},
	StepSbsdPost:         {9, "Akamai rejeitou challenge SbSd", 518, true},
	StepCookieValidation: {10, "Cookie gerado mas validação falhou", 518, true},
	StepTLSAPIError:      {11, "TLS-API indisponível", 518, true},
}

// SolverError representa um erro detalhado do solver
type SolverError struct {
	Step        StepCode
	Provider    string
	Domain      string
	RawError    error
	Attempt     int
	MaxAttempts int
	StartTime   time.Time
}

// Error implementa a interface error
func (e *SolverError) Error() string {
	info := stepInfoMap[e.Step]
	return fmt.Sprintf("[%s] %s: %v", e.Step, info.Description, e.RawError)
}

// HTTPStatus retorna o status HTTP apropriado
func (e *SolverError) HTTPStatus() int {
	if info, ok := stepInfoMap[e.Step]; ok {
		return info.HTTPStatus
	}
	return 518
}

// IsRetryable indica se o erro permite retry
func (e *SolverError) IsRetryable() bool {
	if info, ok := stepInfoMap[e.Step]; ok {
		return info.Retryable
	}
	return false
}

// StepNumber retorna o número do step
func (e *SolverError) StepNumber() int {
	if info, ok := stepInfoMap[e.Step]; ok {
		return info.Number
	}
	return 0
}

// Description retorna a descrição do step
func (e *SolverError) Description() string {
	if info, ok := stepInfoMap[e.Step]; ok {
		return info.Description
	}
	return "Erro desconhecido"
}

// ElapsedMs retorna o tempo decorrido em milissegundos
func (e *SolverError) ElapsedMs() int64 {
	return time.Since(e.StartTime).Milliseconds()
}

// Constructors para cada tipo de erro

func NewScraperInitError(err error, domain string) *SolverError {
	return &SolverError{
		Step:      StepScraperInit,
		Domain:    domain,
		RawError:  err,
		StartTime: time.Now(),
	}
}

func NewScriptURLExtractionError(err error, domain string) *SolverError {
	return &SolverError{
		Step:      StepScriptURLExtract,
		Domain:    domain,
		RawError:  err,
		StartTime: time.Now(),
	}
}

func NewScriptFetchError(err error, domain string) *SolverError {
	return &SolverError{
		Step:      StepScriptFetch,
		Domain:    domain,
		RawError:  err,
		StartTime: time.Now(),
	}
}

func NewScriptDecodeError(err error, domain string) *SolverError {
	return &SolverError{
		Step:      StepScriptDecode,
		Domain:    domain,
		RawError:  err,
		StartTime: time.Now(),
	}
}

func NewBmSoExtractionError(domain string) *SolverError {
	return &SolverError{
		Step:      StepBmSoExtraction,
		Domain:    domain,
		RawError:  fmt.Errorf("cookie bm_so ou sbsd_o não encontrado"),
		StartTime: time.Now(),
	}
}

func NewProviderCallError(err error, provider, domain string, attempt, maxAttempts int, startTime time.Time) *SolverError {
	return &SolverError{
		Step:        StepProviderCall,
		Provider:    provider,
		Domain:      domain,
		RawError:    err,
		Attempt:     attempt,
		MaxAttempts: maxAttempts,
		StartTime:   startTime,
	}
}

func NewSensorPostError(err error, provider, domain string, attempt, maxAttempts int, startTime time.Time) *SolverError {
	return &SolverError{
		Step:        StepSensorPost,
		Provider:    provider,
		Domain:      domain,
		RawError:    err,
		Attempt:     attempt,
		MaxAttempts: maxAttempts,
		StartTime:   startTime,
	}
}

func NewSbsdGenerationError(err error, provider, domain string) *SolverError {
	return &SolverError{
		Step:      StepSbsdGeneration,
		Provider:  provider,
		Domain:    domain,
		RawError:  err,
		StartTime: time.Now(),
	}
}

func NewSbsdPostError(err error, provider, domain string) *SolverError {
	return &SolverError{
		Step:      StepSbsdPost,
		Provider:  provider,
		Domain:    domain,
		RawError:  err,
		StartTime: time.Now(),
	}
}

func NewCookieValidationError(domain string, attempt, maxAttempts int, startTime time.Time) *SolverError {
	return &SolverError{
		Step:        StepCookieValidation,
		Domain:      domain,
		RawError:    fmt.Errorf("cookie _abck não contém token ~0~ válido"),
		Attempt:     attempt,
		MaxAttempts: maxAttempts,
		StartTime:   startTime,
	}
}

func NewTLSAPIError(err error, domain string) *SolverError {
	return &SolverError{
		Step:      StepTLSAPIError,
		Domain:    domain,
		RawError:  err,
		StartTime: time.Now(),
	}
}
