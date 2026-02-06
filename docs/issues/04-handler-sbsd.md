# #4 - Handler POST /sbsd

**Etiqueta:** `Feature`
**Prioridade:** Alta
**Dependência:** #1, #2, #3
**Epic:** [Refatorar API HTTP](00-epic-refatorar-api-http.md)

---

## Descrição

Implementar o endpoint `POST /sbsd` que executa o fluxo completo de geração de challenge SbSd, retornando todos os cookies gerados e tratando erros de forma detalhada.

---

## Fluxo Completo

```
┌─────────────────────────────────────────────────────────────────┐
│  STEP 1: Criar Scraper (com SbSd: true)                         │
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
│  STEP 3: GetAntiBotScript                                       │
│  └─ Baixa o script anti-bot (retorna base64)                    │
│  └─ Erro: "script_fetch"                                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 4: Decodificar script (base64 → raw JS)                   │
│  └─ Erro: "script_decode"                                       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 5: Extrair cookie bm_so ou sbsd_o                         │
│  └─ Procura nos cookies já coletados                            │
│  └─ Erro: "bm_so_extraction"                                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 6: GenerateSbSdChallenge (chamar provider)                │
│  └─ Envia: script raw, bmSo, userAgent, etc.                    │
│  └─ Recebe: challenge data                                      │
│  └─ Erro: "sbsd_generation"                                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 7: PostSbSdChallenge (enviar para Akamai)                 │
│  └─ POST challenge data para sensor endpoint                    │
│  └─ Cookies atualizados: _abck (válido), bm_s                   │
│  └─ Erro: "sbsd_post"                                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 8: Coletar e retornar todos os cookies                    │
│  └─ _abck, bm_sz, bm_s, outros                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Request

### Endpoint
```
POST /sbsd
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
  "sbSdProvider": "jevi",
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
| `sbSdProvider` | string | Não | akamaiProvider | Provider específico para SbSd |
| `generateReport` | bool | Não | false | Gerar relatório de debug |

### Perfis TLS Suportados

| Valor | Descrição |
|-------|-----------|
| `chrome_144` | Chrome 144 (padrão) |
| `chrome_142` | Chrome 142 |
| `ios_standard` | iOS Standard |
| `ios_secondary` | iOS Secondary |
| `ios_26` | iOS 26 |
| `ios_18` | iOS 18 Standard |
| `safari_ios_18_5` | Safari iOS 18.5 |
| `firefox_135` | Firefox 135 |

---

## Response

### Sucesso (200 OK)

```json
{
  "success": true,
  "cookies": {
    "full_string": "_abck=ABC123~0~XYZ; bm_sz=123ABC; bm_s=DEF456",
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
      },
      {
        "name": "bm_s",
        "value": "DEF456GHI789...",
        "domain": ".nike.com.br"
      }
    ]
  },
  "telemetry": {
    "abck_token": "ABC123",
    "bm_sz_encoded": "MTIzQUJDNDU2REVG...",
    "bm_s_encoded": "REVGNDM2R0hJNzg5..."
  },
  "session": {
    "provider": "jevi",
    "profile": "chrome_144"
  }
}
```

### Erro (4xx/5xx)

```json
{
  "success": false,
  "error": {
    "step": "bm_so_extraction",
    "step_number": 5,
    "description": "Cookie bm_so/sbsd_o não encontrado",
    "provider": "jevi",
    "domain": "www.nike.com.br",
    "http_status": 400,
    "raw_error": "cookie bm_so ou sbsd_o não encontrado",
    "retryable": false,
    "context": null
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

### `internal/handler/sbsd.go`

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

type SbsdRequest struct {
    URL            string `json:"url"`
    AkamaiURL      string `json:"akamaiUrl"`
    Proxy          string `json:"proxy"`
    RandomUA       string `json:"randomUserAgent"`
    UserAgent      string `json:"userAgent"`
    SecChUa        string `json:"secChUa"`
    Language       string `json:"language"`
    AkamaiProvider string `json:"akamaiProvider"`
    SbSdProvider   string `json:"sbSdProvider"`
    GenerateReport bool   `json:"generateReport"`
}

type SbsdHandler struct {
    config  *config.Config
    service *service.SolverService
}

func NewSbsdHandler(cfg *config.Config, svc *service.SolverService) *SbsdHandler {
    return &SbsdHandler{
        config:  cfg,
        service: svc,
    }
}

func (h *SbsdHandler) Handle(w http.ResponseWriter, r *http.Request) {
    // 1. Decode request
    var req SbsdRequest
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

    // 4. Executar fluxo SbSd
    result, err := h.service.GenerateSbsd(r.Context(), &service.SbsdInput{
        Domain:         req.URL,
        AkamaiURL:      req.AkamaiURL,
        Proxy:          req.Proxy,
        ProfileType:    req.RandomUA,
        UserAgent:      req.UserAgent,
        SecChUa:        req.SecChUa,
        Language:       req.Language,
        AkamaiProvider: req.AkamaiProvider,
        SbSdProvider:   req.SbSdProvider,
        GenerateReport: req.GenerateReport,
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

func (h *SbsdHandler) applyDefaults(req *SbsdRequest) {
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
    if req.SbSdProvider == "" {
        req.SbSdProvider = req.AkamaiProvider
    }
}
```

### `internal/service/sbsd.go`

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

    "github.com/bogdanfinn/tls-client/profiles"
)

type SbsdInput struct {
    Domain         string
    AkamaiURL      string
    Proxy          string
    ProfileType    string
    UserAgent      string
    SecChUa        string
    Language       string
    AkamaiProvider string
    SbSdProvider   string
    GenerateReport bool
}

type SbsdOutput struct {
    Cookies        *response.Cookies
    Telemetry      *response.Telemetry
    Session        *response.Session
    PartialCookies *response.Cookies
    ReportPath     string
}

func (s *SolverService) GenerateSbsd(ctx context.Context, input *SbsdInput) (*SbsdOutput, error) {
    startTime := time.Now()
    output := &SbsdOutput{}

    // STEP 1: Criar Scraper
    profile := s.getProfile(input.ProfileType)
    config := &scraper.Config{
        Domain:         input.Domain,
        SensorUrl:      "",
        Language:       input.Language,
        AkamaiProvider: input.AkamaiProvider,
        SbSdProvider:   input.SbSdProvider,
        SbSd:           true, // Modo SbSd
        UserAgent:      input.UserAgent,
        SecChUa:        input.SecChUa,
        ProfileType:    input.ProfileType,
        GenerateReport: input.GenerateReport,
        // API Keys do config global
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

    // STEP 3: GetAntiBotScript
    scriptB64, err := sc.GetAntiBotScript()
    if err != nil {
        output.PartialCookies = s.collectCookies(sc, input.Domain)
        return output, errors.NewScriptFetchError(err, input.Domain)
    }

    // STEP 4: Decodificar script
    decodedScript, err := base64.StdEncoding.DecodeString(scriptB64)
    if err != nil {
        output.PartialCookies = s.collectCookies(sc, input.Domain)
        return output, errors.NewScriptDecodeError(err, input.Domain)
    }
    rawScript := string(decodedScript)

    // STEP 5: Extrair bm_so ou sbsd_o
    cookies, err := sc.GetCookies(fmt.Sprintf("https://%s", input.Domain))
    if err != nil {
        output.PartialCookies = s.collectCookies(sc, input.Domain)
        return output, errors.NewBmSoExtractionError(input.Domain)
    }

    var bmSo string
    for _, cookie := range cookies {
        if cookie.Name == "bm_so" {
            bmSo = cookie.Value
            break
        } else if cookie.Name == "sbsd_o" && bmSo == "" {
            bmSo = cookie.Value
        }
    }

    if bmSo == "" {
        output.PartialCookies = s.collectCookies(sc, input.Domain)
        return output, errors.NewBmSoExtractionError(input.Domain)
    }

    // STEP 6: GenerateSbSdChallenge
    provider := input.SbSdProvider
    if provider == "" {
        provider = input.AkamaiProvider
    }

    sbsdData, err := sc.GenerateSbSdChallenge(rawScript, bmSo)
    if err != nil {
        output.PartialCookies = s.collectCookies(sc, input.Domain)
        return output, errors.NewSbsdGenerationError(err, provider, input.Domain)
    }

    // STEP 7: PostSbSdChallenge
    err = sc.PostSbSdChallenge(sbsdData)
    if err != nil {
        output.PartialCookies = s.collectCookies(sc, input.Domain)
        return output, errors.NewSbsdPostError(err, provider, input.Domain)
    }

    // STEP 8: Coletar todos os cookies
    finalCookies, err := sc.GetCookies(fmt.Sprintf("https://%s", input.Domain))
    if err != nil {
        output.PartialCookies = s.collectCookies(sc, input.Domain)
        return output, fmt.Errorf("failed to get final cookies: %w", err)
    }

    // Montar response
    output.Cookies = s.cookiesToResponse(finalCookies, input.Domain)
    output.Telemetry = s.buildSbsdTelemetry(finalCookies)
    output.Session = &response.Session{
        Provider: provider,
        Profile:  input.ProfileType,
    }

    return output, nil
}

func (s *SolverService) buildSbsdTelemetry(cookies []*http.Cookie) *response.Telemetry {
    var abck, bmsz, bms string
    for _, c := range cookies {
        if strings.HasPrefix(c.Name, "_abck") {
            abck = c.Value
        } else if strings.HasPrefix(c.Name, "bm_sz") {
            bmsz = c.Value
        } else if c.Name == "bm_s" {
            bms = c.Value
        }
    }

    var abckToken string
    if parts := strings.Split(abck, "~"); len(parts) > 0 {
        abckToken = parts[0]
    }

    return &response.Telemetry{
        AbckToken:   abckToken,
        BmSzEncoded: base64.StdEncoding.EncodeToString([]byte(bmsz)),
        BmSEncoded:  base64.StdEncoding.EncodeToString([]byte(bms)),
    }
}
```

---

## Registro no Router

Atualizar `cmd/server/main.go`:

```go
// Criar service
solverService := service.NewSolverService(cfg)

// Criar handlers
sbsdHandler := handler.NewSbsdHandler(cfg, solverService)

// Registrar rotas
mux.HandleFunc("POST /sbsd", sbsdHandler.Handle)
```

---

## Critérios de Aceitação

- [ ] Endpoint `POST /sbsd` responde corretamente
- [ ] Fluxo completo de 8 steps executado
- [ ] Todos os cookies retornados (`_abck`, `bm_sz`, `bm_s`)
- [ ] Telemetry com `abck_token`, `bm_sz_encoded`, `bm_s_encoded`
- [ ] Erros indicam step exato onde falhou
- [ ] Cookies parciais retornados em caso de erro
- [ ] Debug report path retornado quando `generateReport: true`
- [ ] Todos os perfis TLS funcionando
- [ ] Todos os providers funcionando (jevi, n4s, roolink)

---

## Validação

```bash
# Teste básico
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "proxy": "http://user:pass@proxy:port",
    "akamaiProvider": "jevi"
  }'

# Teste com report
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "proxy": "http://user:pass@proxy:port",
    "akamaiProvider": "jevi",
    "generateReport": true
  }'

# Verificar resposta de erro
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "invalid-domain-that-does-not-exist.com"
  }'
```
