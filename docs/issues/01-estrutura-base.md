# #1 - Estrutura base (cmd/, internal/, config)

**Etiqueta:** `Improvement`
**Prioridade:** Alta
**Dependência:** Nenhuma
**Epic:** [Refatorar API HTTP](00-epic-refatorar-api-http.md)

---

## Descrição

Criar a estrutura de diretórios e arquivos base para o servidor HTTP, seguindo convenções Go e preparando para os handlers que serão implementados nas próximas issues.

---

## Por que `cmd/server/` e não `main.go` na raiz?

O `go.mod` declara `module gerador_cookies`. Um `main.go` na raiz tornaria o módulo inteiro um `package main`, quebrando o import path do pacote `scraper`. Outros projetos que fazem `import "gerador_cookies/scraper"` parariam de funcionar.

---

## Estrutura a Criar

```
gerador_cookies/
├── cmd/
│   └── server/
│       └── main.go              # Entry point com wiring
├── internal/
│   ├── config/
│   │   └── config.go            # Struct Config + Load()
│   ├── handler/
│   │   └── .gitkeep             # Placeholder (handlers nas próximas issues)
│   ├── middleware/
│   │   └── .gitkeep             # Placeholder
│   ├── response/
│   │   └── types.go             # Tipos de response padronizados
│   └── service/
│       └── .gitkeep             # Placeholder
```

---

## Implementação

### `cmd/server/main.go`

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "gerador_cookies/internal/config"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("failed to load config: %v", err)
    }

    mux := http.NewServeMux()

    // Handlers serão adicionados nas próximas issues
    // mux.HandleFunc("POST /sbsd", sbsdHandler.Handle)
    // mux.HandleFunc("POST /abck", abckHandler.Handle)
    // mux.HandleFunc("POST /search", searchHandler.Handle)
    // mux.HandleFunc("GET /health", healthHandler.Handle)
    // mux.HandleFunc("GET /ready", healthHandler.Ready)

    srv := &http.Server{
        Addr:         fmt.Sprintf(":%d", cfg.Port),
        Handler:      mux,
        ReadTimeout:  cfg.ReadTimeout,
        WriteTimeout: cfg.WriteTimeout,
    }

    go func() {
        log.Printf("Starting server on :%d", cfg.Port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("server error: %v", err)
        }
    }()

    // Graceful shutdown (detalhes na issue #7)
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
    <-quit

    log.Println("Shutting down server...")
}
```

### `internal/config/config.go`

```go
package config

import (
    "os"
    "strconv"
    "time"
)

type Config struct {
    // Server
    Port            int
    ReadTimeout     time.Duration
    WriteTimeout    time.Duration
    ShutdownTimeout time.Duration

    // TLS-API
    TLSAPIUrl   string
    TLSAPIToken string

    // Providers (serão adicionados na issue #3)
    // JeviAPIKey    string
    // N4SAPIKey     string
    // RoolinkAPIKey string

    // Cache
    CacheEnabled bool

    // Debug
    Debug bool
}

func Load() (*Config, error) {
    cfg := &Config{
        // Defaults
        Port:            getEnvInt("SERVER_PORT", 9999),
        ReadTimeout:     getEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second),
        WriteTimeout:    getEnvDuration("SERVER_WRITE_TIMEOUT", 60*time.Second),
        ShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second),

        TLSAPIUrl:   getEnv("TLS_API_URL", "http://localhost:8080"),
        TLSAPIToken: getEnv("TLS_API_TOKEN", ""),

        CacheEnabled: getEnvBool("REQS_PROVIDER_CACHE_ENABLE", true),
        Debug:        getEnvBool("DEBUG", false),
    }

    return cfg, nil
}

func getEnv(key, defaultValue string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if v := os.Getenv(key); v != "" {
        if i, err := strconv.Atoi(v); err == nil {
            return i
        }
    }
    return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
    if v := os.Getenv(key); v != "" {
        return v == "1" || v == "true" || v == "yes"
    }
    return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
    if v := os.Getenv(key); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            return d
        }
    }
    return defaultValue
}
```

### `internal/response/types.go`

```go
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
    Success        bool     `json:"success"`
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
```

---

## Variáveis de Ambiente

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `SERVER_PORT` | 9999 | Porta do servidor |
| `SERVER_READ_TIMEOUT` | 10s | Timeout de leitura |
| `SERVER_WRITE_TIMEOUT` | 60s | Timeout de escrita |
| `SERVER_SHUTDOWN_TIMEOUT` | 15s | Timeout de graceful shutdown |
| `TLS_API_URL` | http://localhost:8080 | URL da TLS-API |
| `TLS_API_TOKEN` | - | Token da TLS-API |
| `REQS_PROVIDER_CACHE_ENABLE` | true | Habilitar cache de providers |
| `DEBUG` | false | Modo debug |

---

## Critérios de Aceitação

- [ ] Estrutura de diretórios criada
- [ ] `cmd/server/main.go` compila sem erros
- [ ] `internal/config/config.go` carrega variáveis de ambiente
- [ ] `internal/response/types.go` define tipos de response
- [ ] `go build ./cmd/server/` funciona
- [ ] `go vet ./...` sem warnings

---

## Validação

```bash
# Criar estrutura
mkdir -p cmd/server internal/{config,handler,middleware,response,service}

# Build
go build ./cmd/server/

# Testar execução
./server
# Deve iniciar na porta 9999
```
