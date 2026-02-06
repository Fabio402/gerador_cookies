# #7 - Health checks + Graceful shutdown

**Etiqueta:** `Improvement`
**Prioridade:** Média
**Dependência:** #1
**Epic:** [Refatorar API HTTP](00-epic-refatorar-api-http.md)

---

## Descrição

Implementar endpoints de health check e graceful shutdown para funcionamento correto em PM2/EC2.

---

## Endpoints

### GET /health (Liveness)

Verifica se o processo está rodando. Não faz verificações externas.

```bash
curl http://localhost:9999/health
```

**Response (200 OK):**
```json
{
  "status": "ok",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /ready (Readiness)

Verifica se o serviço está pronto para receber requests. Valida dependências externas.

```bash
curl http://localhost:9999/ready
```

**Response (200 OK):**
```json
{
  "status": "ready",
  "checks": {
    "tls_api": "ok"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Response (503 Service Unavailable):**
```json
{
  "status": "not_ready",
  "checks": {
    "tls_api": "failed: connection refused"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Implementação

### `internal/handler/health.go`

```go
package handler

import (
    "encoding/json"
    "net/http"
    "time"

    "gerador_cookies/internal/config"
    "gerador_cookies/scraper"
)

type HealthResponse struct {
    Status    string `json:"status"`
    Timestamp string `json:"timestamp"`
}

type ReadyResponse struct {
    Status    string            `json:"status"`
    Checks    map[string]string `json:"checks"`
    Timestamp string            `json:"timestamp"`
}

type HealthHandler struct {
    config *config.Config
}

func NewHealthHandler(cfg *config.Config) *HealthHandler {
    return &HealthHandler{config: cfg}
}

// Health - Liveness check
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)

    json.NewEncoder(w).Encode(&HealthResponse{
        Status:    "ok",
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    })
}

// Ready - Readiness check
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    checks := make(map[string]string)
    allOk := true

    // Check TLS-API
    tlsClient := scraper.NewTLSAPIClient(h.config.TLSAPIUrl, h.config.TLSAPIToken)
    if err := tlsClient.Ping(); err != nil {
        checks["tls_api"] = "failed: " + err.Error()
        allOk = false
    } else {
        checks["tls_api"] = "ok"
    }

    status := "ready"
    httpStatus := http.StatusOK
    if !allOk {
        status = "not_ready"
        httpStatus = http.StatusServiceUnavailable
    }

    w.WriteHeader(httpStatus)
    json.NewEncoder(w).Encode(&ReadyResponse{
        Status:    status,
        Checks:    checks,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    })
}
```

---

## Graceful Shutdown

PM2 envia `SIGTERM` quando quer reiniciar um processo. O servidor deve:
1. Parar de aceitar novas conexões
2. Aguardar requests em andamento terminarem
3. Encerrar após timeout

### Atualizar `cmd/server/main.go`

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "gerador_cookies/internal/config"
    "gerador_cookies/internal/handler"
    "gerador_cookies/internal/service"
)

func main() {
    // 1. Load config
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("failed to load config: %v", err)
    }

    // 2. Create services
    solverService := service.NewSolverService(cfg)

    // 3. Create handlers
    healthHandler := handler.NewHealthHandler(cfg)
    sbsdHandler := handler.NewSbsdHandler(cfg, solverService)
    abckHandler := handler.NewAbckHandler(cfg, solverService)
    searchHandler := handler.NewSearchHandler(cfg, solverService)

    // 4. Setup router
    mux := http.NewServeMux()

    // Health endpoints
    mux.HandleFunc("GET /health", healthHandler.Health)
    mux.HandleFunc("GET /ready", healthHandler.Ready)

    // API endpoints
    mux.HandleFunc("POST /sbsd", sbsdHandler.Handle)
    mux.HandleFunc("POST /abck", abckHandler.Handle)
    mux.HandleFunc("POST /search", searchHandler.Handle)

    // 5. Create server
    srv := &http.Server{
        Addr:         fmt.Sprintf(":%d", cfg.Port),
        Handler:      mux,
        ReadTimeout:  cfg.ReadTimeout,
        WriteTimeout: cfg.WriteTimeout,
    }

    // 6. Start server in goroutine
    go func() {
        log.Printf("Starting server on :%d", cfg.Port)
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            log.Fatalf("server error: %v", err)
        }
    }()

    // 7. Wait for shutdown signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

    sig := <-quit
    log.Printf("Received signal %v, initiating graceful shutdown...", sig)

    // 8. Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }

    log.Println("Server shutdown complete")
}
```

---

## PM2 Configuration

### `ecosystem.config.js`

```javascript
module.exports = {
  apps: [{
    name: 'gerador-cookies',
    script: './server',
    instances: 1,
    autorestart: true,
    watch: false,
    max_memory_restart: '256M',

    // Graceful shutdown
    kill_timeout: 15000,        // Wait 15s before SIGKILL
    wait_ready: true,           // Wait for process.send('ready')
    listen_timeout: 10000,      // Timeout waiting for ready

    // Environment
    env: {
      SERVER_PORT: 9999,
      SERVER_SHUTDOWN_TIMEOUT: '10s',
      TLS_API_URL: 'http://localhost:8080',
      REQS_PROVIDER_CACHE_ENABLE: '1'
    }
  }]
}
```

### Importante sobre `instances: 1`

**Não use múltiplas instâncias PM2** porque:
- Cada instância tem seu próprio `ProviderCache` in-memory
- Cache diverge entre instâncias
- Para escalar horizontalmente, use múltiplas EC2 com ALB

---

## AWS ALB Health Check

Se estiver atrás de um Application Load Balancer:

| Configuração | Valor |
|--------------|-------|
| Protocol | HTTP |
| Path | `/ready` |
| Port | 9999 |
| Healthy threshold | 2 |
| Unhealthy threshold | 3 |
| Timeout | 5 seconds |
| Interval | 30 seconds |

---

## Middleware de Timeout por Request

Para evitar que requests lentas travem o servidor:

### `internal/middleware/timeout.go`

```go
package middleware

import (
    "context"
    "net/http"
    "time"
)

func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx, cancel := context.WithTimeout(r.Context(), timeout)
            defer cancel()

            r = r.WithContext(ctx)

            done := make(chan struct{})
            go func() {
                next.ServeHTTP(w, r)
                close(done)
            }()

            select {
            case <-done:
                return
            case <-ctx.Done():
                w.WriteHeader(http.StatusGatewayTimeout)
                w.Write([]byte(`{"error": "request timeout"}`))
            }
        })
    }
}
```

**Uso:**
```go
mux.Handle("POST /sbsd", middleware.Timeout(60*time.Second)(sbsdHandler))
```

---

## Critérios de Aceitação

- [ ] `GET /health` retorna 200 sempre que processo está rodando
- [ ] `GET /ready` retorna 200 quando TLS-API está acessível
- [ ] `GET /ready` retorna 503 quando TLS-API está indisponível
- [ ] SIGTERM inicia graceful shutdown
- [ ] Requests em andamento completam antes do shutdown
- [ ] Timeout de shutdown respeitado (15s default)
- [ ] PM2 pode reiniciar sem perda de requests

---

## Validação

```bash
# Build
go build -o server ./cmd/server/

# Iniciar com PM2
pm2 start ecosystem.config.js

# Verificar health
curl http://localhost:9999/health

# Verificar ready
curl http://localhost:9999/ready

# Testar graceful shutdown
pm2 reload gerador-cookies

# Verificar logs
pm2 logs gerador-cookies
# Deve mostrar: "Received signal terminated, initiating graceful shutdown..."
# Seguido de: "Server shutdown complete"

# Monitorar
pm2 monit
```

---

## Troubleshooting

| Problema | Causa | Solução |
|----------|-------|---------|
| /ready retorna 503 | TLS-API indisponível | Verificar se TLS-API está rodando |
| Shutdown demora muito | Requests lentas | Verificar timeout das requests |
| PM2 mata processo | kill_timeout muito baixo | Aumentar para 15000+ |
