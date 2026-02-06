# #5 - Handler POST /abck

**Etiqueta:** `Feature`
**Prioridade:** Alta
**Dependência:** #1, #2, #3
**Epic:** [Refatorar API HTTP](00-epic-refatorar-api-http.md)

---

## Descrição

Implementar o endpoint `POST /abck` que executa o fluxo completo de geração de sensor ABCK, retornando todos os cookies gerados e tratando erros de forma detalhada.

---

## Fluxo Completo

```
┌─────────────────────────────────────────────────────────────────┐
│  STEP 1: Criar Scraper                                          │
│  └─ Erro: "scraper_init"                                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 2: GetAntiBotScriptURL                                    │
│  └─ Busca homepage, extrai URL do script anti-bot               │
│  └─ Cookies coletados: _abck (inicial), bm_sz                   │
│  └─ Erro: "script_url_extraction"                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 3: GetAntiBotScript (ou usar cache)                       │
│  └─ Se cache válido: SeedAbckScriptCookies                      │
│  └─ Se não: baixa script completo (retorna base64)              │
│  └─ Erro: "script_fetch"                                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 4: GenerateSession (loop até SensorPostLimit)             │
│  └─ 4.1: Chamar provider API (jevi/n4s/roolink)                 │
│  │       └─ Erro: "provider_call"                               │
│  └─ 4.2: POST sensor para Akamai                                │
│  │       └─ Erro: "sensor_post"                                 │
│  └─ 4.3: Validar cookie _abck (contém ~0~)                      │
│          └─ Erro: "cookie_validation" (se todas tentativas)     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 5: Coletar e retornar todos os cookies                    │
│  └─ _abck (com ~0~), bm_sz, outros                              │
└─────────────────────────────────────────────────────────────────┘
```

---

## Request

### Endpoint
```
POST /abck
Content-Type: application/json
```

### Body

```json
{
  "url": "www.nike.com.br",
  "akamaiUrl": "/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/ips.js",
  "proxy": "http://user:pass@proxy.example.com:8080",
  "randomUserAgent": "chrome_144",
  "userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36",
  "secChUa": "\"Not(A:Brand\";v=\"8\", \"Chromium\";v=\"144\", \"Google Chrome\";v=\"144\"",
  "language": "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7",
  "akamaiProvider": "jevi",
  "sensorPostLimit": 8,
  "lowSecurity": false,
  "useScript": false,
  "forceUpdateDynamics": false,
  "generateReport": false
}
```

### Campos

| Campo | Tipo | Obrigatório | Default | Descrição |
|-------|------|-------------|---------|-----------|
| `url` | string | Sim | - | Domínio alvo (sem https://) |
| `akamaiUrl` | string | Não | Auto-detect | URL do script anti-bot |
| `proxy` | string | Não | - | URL do proxy |
| `randomUserAgent` | string | Não | chrome_144 | Perfil TLS |
| `userAgent` | string | Não | Chrome 144 | User-Agent customizado |
| `secChUa` | string | Não | Chrome 144 | Sec-CH-UA header |
| `language` | string | Não | en-US | Accept-Language header |
| `akamaiProvider` | string | Não | jevi | Provider (jevi/n4s/roolink) |
| `sensorPostLimit` | int | Não | 8 | Máximo de tentativas |
| `lowSecurity` | bool | Não | false | Validação menos rigorosa |
| `useScript` | bool | Não | false | Incluir script nas requests |
| `forceUpdateDynamics` | bool | Não | false | Ignorar cache |
| `generateReport` | bool | Não | false | Gerar relatório de debug |

---

## Response

### Sucesso (200 OK)

```json
{
  "success": true,
  "cookies": {
    "full_string": "_abck=ABC123~0~XYZ789; bm_sz=123ABC456",
    "items": [
      {
        "name": "_abck",
        "value": "ABC123~0~XYZ789...",
        "domain": ".nike.com.br"
      },
      {
        "name": "bm_sz",
        "value": "123ABC456DEF...",
        "domain": ".nike.com.br"
      }
    ]
  },
  "telemetry": {
    "abck_token": "ABC123",
    "bm_sz_encoded": "MTIzQUJDNDU2REVG...",
    "sensor_data_encoded": "c2Vuc29yX2RhdGFfaGVyZQ=="
  },
  "session": {
    "provider": "jevi",
    "profile": "chrome_144",
    "attempts": 2
  }
}
```

### Erro (4xx/5xx)

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
      {"name": "_abck", "value": "initial...", "domain": ".nike.com.br"},
      {"name": "bm_sz", "value": "xxx", "domain": ".nike.com.br"}
    ]
  },
  "debug": {
    "report_path": "/tmp/getsensor-report-1234567890.txt"
  }
}
```

---

## Implementação

### `internal/handler/abck.go`

```go
package handler

import (
    "encoding/json"
    "net/http"

    "gerador_cookies/internal/config"
    "gerador_cookies/internal/errors"
    "gerador_cookies/internal/response"
    "gerador_cookies/internal/service"
)

type AbckRequest struct {
    URL                 string `json:"url"`
    AkamaiURL           string `json:"akamaiUrl"`
    Proxy               string `json:"proxy"`
    RandomUA            string `json:"randomUserAgent"`
    UserAgent           string `json:"userAgent"`
    SecChUa             string `json:"secChUa"`
    Language            string `json:"language"`
    AkamaiProvider      string `json:"akamaiProvider"`
    SensorPostLimit     int    `json:"sensorPostLimit"`
    LowSecurity         bool   `json:"lowSecurity"`
    UseScript           bool   `json:"useScript"`
    ForceUpdateDynamics bool   `json:"forceUpdateDynamics"`
    GenerateReport      bool   `json:"generateReport"`
}

type AbckHandler struct {
    config  *config.Config
    service *service.SolverService
}

func NewAbckHandler(cfg *config.Config, svc *service.SolverService) *AbckHandler {
    return &AbckHandler{
        config:  cfg,
        service: svc,
    }
}

func (h *AbckHandler) Handle(w http.ResponseWriter, r *http.Request) {
    // 1. Decode request
    var req AbckRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.WriteError(w, http.StatusBadRequest, &response.ErrorResponse{
            Success: false,
            Error: &response.ErrorDetail{
                Step:        "request_decode",
                StepNumber:  0,
                Description: "Falha ao decodificar request JSON",
                RawError:    err.Error(),
                Retryable:   false,
            },
        })
        return
    }

    // 2. Validar campos obrigatórios
    if req.URL == "" {
        response.WriteError(w, http.StatusBadRequest, &response.ErrorResponse{
            Success: false,
            Error: &response.ErrorDetail{
                Step:        "request_validation",
                StepNumber:  0,
                Description: "Campo 'url' é obrigatório",
                RawError:    "missing required field: url",
                Retryable:   false,
            },
        })
        return
    }

    // 3. Aplicar defaults
    h.applyDefaults(&req)

    // 4. Executar fluxo ABCK
    result, err := h.service.GenerateAbck(r.Context(), &service.AbckInput{
        Domain:              req.URL,
        AkamaiURL:           req.AkamaiURL,
        Proxy:               req.Proxy,
        ProfileType:         req.RandomUA,
        UserAgent:           req.UserAgent,
        SecChUa:             req.SecChUa,
        Language:            req.Language,
        AkamaiProvider:      req.AkamaiProvider,
        SensorPostLimit:     req.SensorPostLimit,
        LowSecurity:         req.LowSecurity,
        UseScript:           req.UseScript,
        ForceUpdateDynamics: req.ForceUpdateDynamics,
        GenerateReport:      req.GenerateReport,
    })

    // 5. Tratar erro
    if err != nil {
        if solverErr, ok := err.(*errors.SolverError); ok {
            errResp := solverErr.ToErrorResponse()

            if result != nil && result.PartialCookies != nil {
                errors.WithPartialCookies(errResp, result.PartialCookies)
            }
            if req.GenerateReport && result != nil && result.ReportPath != "" {
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
                Description: "Erro interno do servidor",
                RawError:    err.Error(),
                Retryable:   false,
            },
        })
        return
    }

    // 6. Sucesso
    response.WriteSuccess(w, &response.SuccessResponse{
        Success:   true,
        Cookies:   result.Cookies,
        Telemetry: result.Telemetry,
        Session:   result.Session,
    })
}

func (h *AbckHandler) applyDefaults(req *AbckRequest) {
    if req.RandomUA == "" {
        req.RandomUA = "chrome_144"
    }
    if req.UserAgent == "" {
        req.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"
    }
    if req.SecChUa == "" {
        req.SecChUa = `"Not(A:Brand";v="8", "Chromium";v="144", "Google Chrome";v="144"`
    }
    if req.Language == "" {
        req.Language = "en-US"
    }
    if req.AkamaiProvider == "" {
        req.AkamaiProvider = "jevi"
    }
    if req.SensorPostLimit <= 0 {
        req.SensorPostLimit = 8
    }
}
```

### `internal/service/abck.go`

```go
package service

import (
    "context"
    "encoding/base64"
    "fmt"
    "net/http"
    "strings"
    "time"

    "gerador_cookies/internal/errors"
    "gerador_cookies/internal/response"
    "gerador_cookies/scraper"
)

type AbckInput struct {
    Domain              string
    AkamaiURL           string
    Proxy               string
    ProfileType         string
    UserAgent           string
    SecChUa             string
    Language            string
    AkamaiProvider      string
    SensorPostLimit     int
    LowSecurity         bool
    UseScript           bool
    ForceUpdateDynamics bool
    GenerateReport      bool
}

type AbckOutput struct {
    Cookies        *response.Cookies
    Telemetry      *response.Telemetry
    Session        *response.Session
    PartialCookies *response.Cookies
    ReportPath     string
}

func (s *SolverService) GenerateAbck(ctx context.Context, input *AbckInput) (*AbckOutput, error) {
    startTime := time.Now()
    output := &AbckOutput{}

    // STEP 1: Criar Scraper
    profile := s.getProfile(input.ProfileType)
    config := &scraper.Config{
        Domain:              input.Domain,
        SensorUrl:           "",
        SensorPostLimit:     input.SensorPostLimit,
        Language:            input.Language,
        LowSecurity:         input.LowSecurity,
        UseScript:           input.UseScript,
        ForceUpdateDynamics: input.ForceUpdateDynamics,
        AkamaiProvider:      input.AkamaiProvider,
        SbSd:                false, // Modo ABCK
        UserAgent:           input.UserAgent,
        SecChUa:             input.SecChUa,
        ProfileType:         input.ProfileType,
        GenerateReport:      input.GenerateReport,
        // API Keys
        JeviAPIKey:    s.config.JeviAPIKey,
        N4SAPIKey:     s.config.N4SAPIKey,
        RoolinkAPIKey: s.config.RoolinkAPIKey,
    }

    sc, err := scraper.NewScraper(input.Proxy, config, profile)
    if err != nil {
        return output, errors.NewScraperInitError(err, input.Domain)
    }
    defer sc.CloseReport()

    if input.GenerateReport {
        output.ReportPath = sc.ReportPath()
    }

    // STEP 2: GetAntiBotScriptURL
    akamaiURL, err := sc.GetAntiBotScriptURL(input.AkamaiURL)
    if err != nil {
        output.PartialCookies = s.collectCookies(sc, input.Domain)
        return output, errors.NewScriptURLExtractionError(err, input.Domain)
    }
    if akamaiURL == "" {
        output.PartialCookies = s.collectCookies(sc, input.Domain)
        return output, errors.NewScriptURLExtractionError(
            fmt.Errorf("script URL not found in page"),
            input.Domain,
        )
    }
    config.SensorUrl = akamaiURL

    // STEP 3: GetAntiBotScript (ou usar cache)
    var script string
    if sc.HasCachedProviderDynamic() {
        // Usar cache - apenas seed cookies
        if err := sc.SeedAbckScriptCookies(); err != nil {
            output.PartialCookies = s.collectCookies(sc, input.Domain)
            return output, errors.NewScriptFetchError(err, input.Domain)
        }
    } else {
        // Baixar script completo
        script, err = sc.GetAntiBotScript()
        if err != nil {
            output.PartialCookies = s.collectCookies(sc, input.Domain)
            return output, errors.NewScriptFetchError(err, input.Domain)
        }
    }

    // STEP 4: GenerateSession (com retries)
    var lastAttempt int
    success, err := sc.GenerateSession(script)

    if err != nil {
        output.PartialCookies = s.collectCookies(sc, input.Domain)

        // Determinar tipo de erro
        errStr := err.Error()
        if strings.Contains(errStr, "provider") || strings.Contains(errStr, "timeout") {
            return output, errors.NewProviderCallError(
                err,
                input.AkamaiProvider,
                input.Domain,
                lastAttempt,
                input.SensorPostLimit,
                startTime,
            )
        }
        return output, errors.NewSensorPostError(
            err,
            input.AkamaiProvider,
            input.Domain,
            lastAttempt,
            input.SensorPostLimit,
            startTime,
        )
    }

    if !success {
        output.PartialCookies = s.collectCookies(sc, input.Domain)
        return output, errors.NewCookieValidationError(
            input.Domain,
            input.SensorPostLimit,
            input.SensorPostLimit,
            startTime,
        )
    }

    // STEP 5: Coletar todos os cookies
    finalCookies, err := sc.GetCookies(fmt.Sprintf("https://%s", input.Domain))
    if err != nil {
        output.PartialCookies = s.collectCookies(sc, input.Domain)
        return output, fmt.Errorf("failed to get final cookies: %w", err)
    }

    // Montar response
    output.Cookies = s.cookiesToResponse(finalCookies, input.Domain)
    output.Telemetry = s.buildAbckTelemetry(finalCookies, sc.SensorDataOnly)
    output.Session = &response.Session{
        Provider: input.AkamaiProvider,
        Profile:  input.ProfileType,
        Attempts: lastAttempt,
    }

    return output, nil
}

func (s *SolverService) buildAbckTelemetry(cookies []*http.Cookie, sensorData string) *response.Telemetry {
    var abck, bmsz string
    for _, c := range cookies {
        if strings.HasPrefix(c.Name, "_abck") {
            abck = c.Value
        } else if strings.HasPrefix(c.Name, "bm_sz") {
            bmsz = c.Value
        }
    }

    var abckToken string
    if parts := strings.Split(abck, "~"); len(parts) > 0 {
        abckToken = parts[0]
    }

    return &response.Telemetry{
        AbckToken:         abckToken,
        BmSzEncoded:       base64.StdEncoding.EncodeToString([]byte(bmsz)),
        SensorDataEncoded: base64.StdEncoding.EncodeToString([]byte(sensorData)),
    }
}
```

---

## Registro no Router

Atualizar `cmd/server/main.go`:

```go
// Criar handlers
abckHandler := handler.NewAbckHandler(cfg, solverService)

// Registrar rotas
mux.HandleFunc("POST /abck", abckHandler.Handle)
```

---

## Diferenças entre /abck e /sbsd

| Aspecto | /abck | /sbsd |
|---------|-------|-------|
| Modo | `SbSd: false` | `SbSd: true` |
| Cache | Pode usar cache | Não usa cache |
| Cookies finais | `_abck`, `bm_sz` | `_abck`, `bm_sz`, `bm_s` |
| Telemetry | `sensor_data_encoded` | `bm_s_encoded` |
| Steps específicos | GenerateSession | GenerateSbSdChallenge + PostSbSdChallenge |
| Retries | Sim (SensorPostLimit) | Não |
| Params extras | `sensorPostLimit`, `lowSecurity`, `useScript` | `sbSdProvider` |

---

## Critérios de Aceitação

- [ ] Endpoint `POST /abck` responde corretamente
- [ ] Fluxo completo de 5 steps executado
- [ ] Cache de provider usado quando disponível
- [ ] Retries até `sensorPostLimit`
- [ ] Todos os cookies retornados (`_abck`, `bm_sz`)
- [ ] Telemetry com `abck_token`, `bm_sz_encoded`, `sensor_data_encoded`
- [ ] Erros indicam step exato e tentativa onde falhou
- [ ] Cookies parciais retornados em caso de erro
- [ ] Campo `attempts` no session indica número de tentativas
- [ ] `lowSecurity` mode funcionando

---

## Validação

```bash
# Teste básico
curl -X POST http://localhost:9999/abck \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "proxy": "http://user:pass@proxy:port",
    "akamaiProvider": "jevi"
  }'

# Teste com limite de retries
curl -X POST http://localhost:9999/abck \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "proxy": "http://user:pass@proxy:port",
    "akamaiProvider": "jevi",
    "sensorPostLimit": 3
  }'

# Teste forçando update de cache
curl -X POST http://localhost:9999/abck \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "proxy": "http://user:pass@proxy:port",
    "forceUpdateDynamics": true
  }'
```
