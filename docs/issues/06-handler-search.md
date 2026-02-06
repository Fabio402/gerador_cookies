# #6 - Handler POST /search (Compatibilidade)

**Etiqueta:** `Improvement`
**Prioridade:** Média
**Dependência:** #4, #5
**Epic:** [Refatorar API HTTP](00-epic-refatorar-api-http.md)

---

## Descrição

Implementar o endpoint `POST /search` que mantém compatibilidade com o comportamento atual, suportando os modos SbSd, ABCK e Both (ambos).

---

## Comportamento

| Flag | Comportamento |
|------|---------------|
| `sbSd: true` | Executa apenas fluxo SbSd (equivale a `/sbsd`) |
| `both: true` | Executa SbSd primeiro, depois ABCK |
| Nenhum | Executa apenas fluxo ABCK (equivale a `/abck`) |

---

## Request

### Endpoint
```
POST /search
Content-Type: application/json
```

### Body (compatível com código atual)

```json
{
  "url": "www.nike.com.br",
  "akamaiUrl": "/149e9513.../ips.js",
  "proxy": "http://user:pass@proxy:port",
  "randomUserAgent": "chrome_144",
  "userAgent": "Mozilla/5.0...",
  "secChUa": "\"Chromium\";v=\"144\"...",
  "language": "pt-BR",
  "akamaiProvider": "jevi",
  "sbSdProvider": "jevi",
  "useN4S": false,
  "sbSd": false,
  "both": false,
  "lowSecurity": false,
  "useScript": false,
  "forceUpdateDynamics": false,
  "generateReport": false
}
```

### Campos Específicos de /search

| Campo | Tipo | Default | Descrição |
|-------|------|---------|-----------|
| `sbSd` | bool | false | Ativar modo SbSd |
| `both` | bool | false | Executar SbSd + ABCK |
| `useN4S` | bool | false | **Deprecated** - usar `akamaiProvider: "n4s"` |

---

## Response

### Formato (compatível com código atual)

```json
{
  "cookie": "_abck=ABC123~0~XYZ; bm_sz=123ABC; bm_s=DEF456",
  "telemetry": "a=ABC123&&&e=MTIzQUJD...&&&sensor_data=c2Vuc29y..."
}
```

### Novo Formato (opcional via header)

Se o client enviar `Accept: application/json; version=2`, retornar o novo formato:

```json
{
  "success": true,
  "cookies": {
    "full_string": "...",
    "items": [...]
  },
  "telemetry": {
    "abck_token": "...",
    "bm_sz_encoded": "...",
    "sensor_data_encoded": "..."
  },
  "session": {
    "provider": "jevi",
    "profile": "chrome_144"
  }
}
```

---

## Implementação

### `internal/handler/search.go`

```go
package handler

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"

    "gerador_cookies/internal/config"
    "gerador_cookies/internal/errors"
    "gerador_cookies/internal/response"
    "gerador_cookies/internal/service"
)

// SearchRequest mantém compatibilidade com o request atual
type SearchRequest struct {
    URL                 string  `json:"url"`
    AkamaiURL           string  `json:"akamaiUrl"`
    Proxy               string  `json:"proxy"`
    RandomUA            string  `json:"randomUserAgent"`
    UserAgent           *string `json:"userAgent"`
    SecChUa             *string `json:"secChUa"`
    Language            *string `json:"language"`
    AkamaiProvider      *string `json:"akamaiProvider"`
    SbSdProvider        *string `json:"sbSdProvider"`
    UseN4S              *bool   `json:"useN4S"`      // Deprecated
    SbSd                *bool   `json:"sbSd"`
    Both                *bool   `json:"both"`
    LowSecurity         *bool   `json:"lowSecurity"`
    UseScript           *bool   `json:"useScript"`
    ForceUpdateDynamics *bool   `json:"forceUpdateDynamics"`
    GenerateReport      *bool   `json:"generateReport"`
}

// LegacyResponse mantém compatibilidade com o response atual
type LegacyResponse struct {
    Cookie    string `json:"cookie"`
    Telemetry string `json:"telemetry"`
}

type SearchHandler struct {
    config  *config.Config
    service *service.SolverService
}

func NewSearchHandler(cfg *config.Config, svc *service.SolverService) *SearchHandler {
    return &SearchHandler{
        config:  cfg,
        service: svc,
    }
}

func (h *SearchHandler) Handle(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    // 1. Decode request
    var req SearchRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.writeLegacyError(w)
        return
    }

    // 2. Validar campos obrigatórios
    if req.URL == "" {
        h.writeLegacyError(w)
        return
    }

    // 3. Aplicar defaults e compatibilidade
    h.applyDefaults(&req)

    // 4. Determinar modo
    isSbSd := h.getBool(req.SbSd)
    isBoth := h.getBool(req.Both)

    // 5. Executar fluxo apropriado
    var result *service.SearchOutput
    var err error

    if isBoth {
        // Modo Both: SbSd primeiro, depois ABCK
        result, err = h.service.GenerateSearchBoth(r.Context(), h.toSearchInput(&req))
    } else if isSbSd {
        // Modo SbSd apenas
        result, err = h.service.GenerateSearchSbsd(r.Context(), h.toSearchInput(&req))
    } else {
        // Modo ABCK apenas
        result, err = h.service.GenerateSearchAbck(r.Context(), h.toSearchInput(&req))
    }

    // 6. Tratar erro
    if err != nil {
        if h.getBool(req.GenerateReport) && result != nil && result.ReportPath != "" {
            w.Header().Set("X-Request-Report-Path", result.ReportPath)
        }
        h.writeLegacyError(w)
        return
    }

    // 7. Verificar versão do response
    if h.wantsNewFormat(r) {
        response.WriteSuccess(w, &response.SuccessResponse{
            Success:   true,
            Cookies:   result.Cookies,
            Telemetry: result.Telemetry,
            Session:   result.Session,
        })
        return
    }

    // 8. Response legado
    legacyResp := &LegacyResponse{
        Cookie:    result.Cookies.FullString,
        Telemetry: h.buildLegacyTelemetry(result, isSbSd),
    }

    json.NewEncoder(w).Encode(legacyResp)
}

func (h *SearchHandler) applyDefaults(req *SearchRequest) {
    // RandomUserAgent
    if req.RandomUA == "" {
        req.RandomUA = "chrome_144"
    }

    // UserAgent
    if req.UserAgent == nil {
        ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"
        req.UserAgent = &ua
    }

    // SecChUa
    if req.SecChUa == nil {
        sc := `"Not(A:Brand";v="8", "Chromium";v="144", "Google Chrome";v="144"`
        req.SecChUa = &sc
    }

    // Language
    if req.Language == nil {
        lang := "en-US"
        req.Language = &lang
    }

    // AkamaiProvider (com compatibilidade para useN4S)
    if req.AkamaiProvider == nil {
        provider := "jevi"
        if req.UseN4S != nil && *req.UseN4S {
            provider = "n4s"
        }
        req.AkamaiProvider = &provider
    }

    // Validar provider
    validProviders := map[string]bool{"jevi": true, "n4s": true, "roolink": true}
    if !validProviders[*req.AkamaiProvider] {
        provider := "n4s"
        req.AkamaiProvider = &provider
    }

    // SbSdProvider
    if req.SbSdProvider != nil && !validProviders[*req.SbSdProvider] {
        req.SbSdProvider = nil
    }
}

func (h *SearchHandler) toSearchInput(req *SearchRequest) *service.SearchInput {
    return &service.SearchInput{
        Domain:              req.URL,
        AkamaiURL:           req.AkamaiURL,
        Proxy:               req.Proxy,
        ProfileType:         req.RandomUA,
        UserAgent:           h.getString(req.UserAgent),
        SecChUa:             h.getString(req.SecChUa),
        Language:            h.getString(req.Language),
        AkamaiProvider:      h.getString(req.AkamaiProvider),
        SbSdProvider:        h.getString(req.SbSdProvider),
        LowSecurity:         h.getBool(req.LowSecurity),
        UseScript:           h.getBool(req.UseScript),
        ForceUpdateDynamics: h.getBool(req.ForceUpdateDynamics),
        GenerateReport:      h.getBool(req.GenerateReport),
    }
}

func (h *SearchHandler) buildLegacyTelemetry(result *service.SearchOutput, isSbSd bool) string {
    tel := result.Telemetry
    if isSbSd {
        return fmt.Sprintf("a=%s&&&e=%s&&&bm_s=%s",
            tel.AbckToken,
            tel.BmSzEncoded,
            tel.BmSEncoded,
        )
    }
    return fmt.Sprintf("a=%s&&&e=%s&&&sensor_data=%s",
        tel.AbckToken,
        tel.BmSzEncoded,
        tel.SensorDataEncoded,
    )
}

func (h *SearchHandler) wantsNewFormat(r *http.Request) bool {
    accept := r.Header.Get("Accept")
    return strings.Contains(accept, "version=2")
}

func (h *SearchHandler) writeLegacyError(w http.ResponseWriter) {
    w.Write([]byte(`{ "error": true }`))
}

func (h *SearchHandler) getString(s *string) string {
    if s == nil {
        return ""
    }
    return *s
}

func (h *SearchHandler) getBool(b *bool) bool {
    if b == nil {
        return false
    }
    return *b
}
```

### `internal/service/search.go`

```go
package service

import (
    "context"
)

type SearchInput struct {
    Domain              string
    AkamaiURL           string
    Proxy               string
    ProfileType         string
    UserAgent           string
    SecChUa             string
    Language            string
    AkamaiProvider      string
    SbSdProvider        string
    LowSecurity         bool
    UseScript           bool
    ForceUpdateDynamics bool
    GenerateReport      bool
}

type SearchOutput struct {
    Cookies        *response.Cookies
    Telemetry      *response.Telemetry
    Session        *response.Session
    ReportPath     string
}

// GenerateSearchSbsd executa apenas o fluxo SbSd
func (s *SolverService) GenerateSearchSbsd(ctx context.Context, input *SearchInput) (*SearchOutput, error) {
    result, err := s.GenerateSbsd(ctx, &SbsdInput{
        Domain:         input.Domain,
        AkamaiURL:      input.AkamaiURL,
        Proxy:          input.Proxy,
        ProfileType:    input.ProfileType,
        UserAgent:      input.UserAgent,
        SecChUa:        input.SecChUa,
        Language:       input.Language,
        AkamaiProvider: input.AkamaiProvider,
        SbSdProvider:   input.SbSdProvider,
        GenerateReport: input.GenerateReport,
    })

    if err != nil {
        return &SearchOutput{ReportPath: result.ReportPath}, err
    }

    return &SearchOutput{
        Cookies:    result.Cookies,
        Telemetry:  result.Telemetry,
        Session:    result.Session,
        ReportPath: result.ReportPath,
    }, nil
}

// GenerateSearchAbck executa apenas o fluxo ABCK
func (s *SolverService) GenerateSearchAbck(ctx context.Context, input *SearchInput) (*SearchOutput, error) {
    result, err := s.GenerateAbck(ctx, &AbckInput{
        Domain:              input.Domain,
        AkamaiURL:           input.AkamaiURL,
        Proxy:               input.Proxy,
        ProfileType:         input.ProfileType,
        UserAgent:           input.UserAgent,
        SecChUa:             input.SecChUa,
        Language:            input.Language,
        AkamaiProvider:      input.AkamaiProvider,
        SensorPostLimit:     8,
        LowSecurity:         input.LowSecurity,
        UseScript:           input.UseScript,
        ForceUpdateDynamics: input.ForceUpdateDynamics,
        GenerateReport:      input.GenerateReport,
    })

    if err != nil {
        return &SearchOutput{ReportPath: result.ReportPath}, err
    }

    return &SearchOutput{
        Cookies:    result.Cookies,
        Telemetry:  result.Telemetry,
        Session:    result.Session,
        ReportPath: result.ReportPath,
    }, nil
}

// GenerateSearchBoth executa SbSd primeiro, depois ABCK
func (s *SolverService) GenerateSearchBoth(ctx context.Context, input *SearchInput) (*SearchOutput, error) {
    // 1. Executar SbSd
    sbsdResult, err := s.GenerateSbsd(ctx, &SbsdInput{
        Domain:         input.Domain,
        AkamaiURL:      input.AkamaiURL,
        Proxy:          input.Proxy,
        ProfileType:    input.ProfileType,
        UserAgent:      input.UserAgent,
        SecChUa:        input.SecChUa,
        Language:       input.Language,
        AkamaiProvider: input.AkamaiProvider,
        SbSdProvider:   input.SbSdProvider,
        GenerateReport: input.GenerateReport,
    })

    if err != nil {
        return &SearchOutput{ReportPath: sbsdResult.ReportPath}, err
    }

    // 2. Executar ABCK (reutilizando sessão)
    // TODO: Implementar reutilização de scraper/sessão entre chamadas
    abckResult, err := s.GenerateAbck(ctx, &AbckInput{
        Domain:              input.Domain,
        AkamaiURL:           "", // Auto-detect novamente
        Proxy:               input.Proxy,
        ProfileType:         input.ProfileType,
        UserAgent:           input.UserAgent,
        SecChUa:             input.SecChUa,
        Language:            input.Language,
        AkamaiProvider:      input.AkamaiProvider,
        SensorPostLimit:     8,
        LowSecurity:         input.LowSecurity,
        UseScript:           input.UseScript,
        ForceUpdateDynamics: input.ForceUpdateDynamics,
        GenerateReport:      input.GenerateReport,
    })

    if err != nil {
        // Retornar resultado do SbSd mesmo se ABCK falhar
        return &SearchOutput{
            Cookies:    sbsdResult.Cookies,
            Telemetry:  sbsdResult.Telemetry,
            Session:    sbsdResult.Session,
            ReportPath: sbsdResult.ReportPath,
        }, err
    }

    // 3. Combinar resultados (ABCK tem precedência nos cookies)
    return &SearchOutput{
        Cookies:   abckResult.Cookies,
        Telemetry: abckResult.Telemetry,
        Session: &response.Session{
            Provider: input.AkamaiProvider,
            Profile:  input.ProfileType,
            Attempts: abckResult.Session.Attempts,
        },
        ReportPath: abckResult.ReportPath,
    }, nil
}
```

---

## Critérios de Aceitação

- [ ] Request body 100% compatível com código atual
- [ ] Response legado `{cookie, telemetry}` por padrão
- [ ] Response novo quando `Accept: application/json; version=2`
- [ ] Flag `useN4S` funciona (deprecated mas compatível)
- [ ] Flag `sbSd: true` executa fluxo SbSd
- [ ] Flag `both: true` executa SbSd + ABCK
- [ ] Header `X-Request-Report-Path` quando `generateReport: true`
- [ ] Erros retornam `{ "error": true }` (compatibilidade)

---

## Validação

```bash
# Teste modo ABCK (padrão)
curl -X POST http://localhost:9999/search \
  -H "Content-Type: application/json" \
  -d '{"url": "www.nike.com.br", "proxy": "..."}'

# Teste modo SbSd
curl -X POST http://localhost:9999/search \
  -H "Content-Type: application/json" \
  -d '{"url": "www.nike.com.br", "proxy": "...", "sbSd": true}'

# Teste modo Both
curl -X POST http://localhost:9999/search \
  -H "Content-Type: application/json" \
  -d '{"url": "www.nike.com.br", "proxy": "...", "both": true}'

# Teste novo formato
curl -X POST http://localhost:9999/search \
  -H "Content-Type: application/json" \
  -H "Accept: application/json; version=2" \
  -d '{"url": "www.nike.com.br", "proxy": "..."}'
```
