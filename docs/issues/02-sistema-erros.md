# #2 - Sistema de erros detalhados

**Etiqueta:** `Improvement`
**Prioridade:** Alta
**Dependência:** #1
**Epic:** [Refatorar API HTTP](00-epic-refatorar-api-http.md)

---

## Descrição

Implementar um sistema de erros que indique exatamente onde e por que cada falha ocorreu durante os fluxos de geração de cookies. Cada erro deve conter informações suficientes para debug sem precisar acessar logs do servidor.

---

## Requisitos

1. Cada erro deve indicar o **step exato** onde falhou
2. Incluir **mensagem original** do erro (`raw_error`)
3. Incluir **contexto** (tentativas, tempo decorrido)
4. Indicar se o erro é **retryable**
5. Retornar **cookies parciais** coletados até o ponto de falha
6. Mapear erros para **HTTP status codes** apropriados

---

## Steps e Códigos de Erro

| Step # | Código | HTTP | Retryable | Descrição |
|--------|--------|------|-----------|-----------|
| 1 | `scraper_init` | 500 | Não | Falha ao criar cliente TLS |
| 2 | `script_url_extraction` | 502 | Sim | Script anti-bot não encontrado no HTML |
| 3 | `script_fetch` | 502 | Sim | Falha ao baixar script |
| 4 | `script_decode` | 400 | Não | Base64 inválido |
| 5 | `bm_so_extraction` | 400 | Não | Cookie bm_so/sbsd_o não encontrado |
| 6 | `provider_call` | 503 | Sim | Provider indisponível |
| 7 | `sensor_post` | 502 | Sim | Akamai rejeitou sensor |
| 8 | `sbsd_generation` | 503 | Sim | Falha ao gerar challenge SbSd |
| 9 | `sbsd_post` | 502 | Sim | Akamai rejeitou challenge |
| 10 | `cookie_validation` | 200 | Sim | Cookie gerado mas inválido |
| 11 | `tls_api_error` | 503 | Sim | TLS-API indisponível |

---

## Implementação

### `internal/errors/errors.go`

```go
package errors

import (
    "fmt"
    "net/http"
    "time"
)

// StepCode representa o código do step onde ocorreu o erro
type StepCode string

const (
    StepScraperInit        StepCode = "scraper_init"
    StepScriptURLExtract   StepCode = "script_url_extraction"
    StepScriptFetch        StepCode = "script_fetch"
    StepScriptDecode       StepCode = "script_decode"
    StepBmSoExtraction     StepCode = "bm_so_extraction"
    StepProviderCall       StepCode = "provider_call"
    StepSensorPost         StepCode = "sensor_post"
    StepSbsdGeneration     StepCode = "sbsd_generation"
    StepSbsdPost           StepCode = "sbsd_post"
    StepCookieValidation   StepCode = "cookie_validation"
    StepTLSAPIError        StepCode = "tls_api_error"
)

// stepInfo contém metadados de cada step
type stepInfo struct {
    Number      int
    Description string
    HTTPStatus  int
    Retryable   bool
}

var stepInfoMap = map[StepCode]stepInfo{
    StepScraperInit:        {1, "Falha ao criar cliente TLS", http.StatusInternalServerError, false},
    StepScriptURLExtract:   {2, "Script anti-bot não encontrado no HTML", http.StatusBadGateway, true},
    StepScriptFetch:        {3, "Falha ao baixar script anti-bot", http.StatusBadGateway, true},
    StepScriptDecode:       {4, "Falha ao decodificar script base64", http.StatusBadRequest, false},
    StepBmSoExtraction:     {5, "Cookie bm_so/sbsd_o não encontrado", http.StatusBadRequest, false},
    StepProviderCall:       {6, "Falha ao chamar API do provider", http.StatusServiceUnavailable, true},
    StepSensorPost:         {7, "Akamai rejeitou sensor data", http.StatusBadGateway, true},
    StepSbsdGeneration:     {8, "Falha ao gerar challenge SbSd", http.StatusServiceUnavailable, true},
    StepSbsdPost:           {9, "Akamai rejeitou challenge SbSd", http.StatusBadGateway, true},
    StepCookieValidation:   {10, "Cookie gerado mas validação falhou", http.StatusOK, true},
    StepTLSAPIError:        {11, "TLS-API indisponível", http.StatusServiceUnavailable, true},
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
    return http.StatusInternalServerError
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
```

### `internal/errors/converter.go`

```go
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
```

---

## Exemplo de Response de Erro

```json
{
  "success": false,
  "error": {
    "step": "provider_call",
    "step_number": 6,
    "description": "Falha ao chamar API do provider",
    "provider": "jevi",
    "domain": "www.nike.com.br",
    "http_status": 503,
    "raw_error": "connection timeout after 10s",
    "retryable": true,
    "context": {
      "attempt": 3,
      "max_attempts": 8,
      "elapsed_ms": 12543
    }
  },
  "partial_cookies": {
    "full_string": "_abck=initial; bm_sz=xxx",
    "items": [
      {"name": "_abck", "value": "initial...", "domain": "www.nike.com.br"},
      {"name": "bm_sz", "value": "xxx", "domain": "www.nike.com.br"}
    ]
  },
  "debug": {
    "report_path": "/tmp/getsensor-report-1234567890.txt"
  }
}
```

---

## Uso nos Handlers

```go
// Exemplo de uso em um handler
func (h *SbsdHandler) Handle(w http.ResponseWriter, r *http.Request) {
    // ... processamento ...

    result, err := h.service.GenerateSbsd(ctx, req)
    if err != nil {
        if solverErr, ok := err.(*errors.SolverError); ok {
            errResp := solverErr.ToErrorResponse()

            // Adicionar cookies parciais se disponíveis
            if result != nil && result.PartialCookies != nil {
                errors.WithPartialCookies(errResp, result.PartialCookies)
            }

            // Adicionar debug se habilitado
            if req.GenerateReport && result != nil {
                errors.WithDebug(errResp, result.ReportPath)
            }

            response.WriteError(w, solverErr.HTTPStatus(), errResp)
            return
        }

        // Erro genérico
        response.WriteError(w, http.StatusInternalServerError, &response.ErrorResponse{
            Success: false,
            Error: &response.ErrorDetail{
                Step:        "unknown",
                StepNumber:  0,
                Description: "Erro interno do servidor",
                RawError:    err.Error(),
                Retryable:   false,
            },
        })
        return
    }

    // Sucesso
    response.WriteSuccess(w, result.ToSuccessResponse())
}
```

---

## Critérios de Aceitação

- [ ] Package `internal/errors` criado
- [ ] Todos os steps mapeados com códigos, descrições e HTTP status
- [ ] Constructors para cada tipo de erro
- [ ] Método `ToErrorResponse()` converte para response JSON
- [ ] Suporte a cookies parciais na resposta de erro
- [ ] Suporte a informações de debug
- [ ] Header `Retry-After: 1` adicionado quando retryable
- [ ] Testes unitários para conversão de erros

---

## Validação

```bash
# Build
go build ./internal/errors/

# Testar
go test ./internal/errors/
```
