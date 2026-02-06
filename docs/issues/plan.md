# Epic: Refatorar API HTTP - Plano de Execução

## Status Geral
- **Epic:** [Refatorar API HTTP](00-epic-refatorar-api-http.md)
- **Última Atualização:** 2026-02-03
- **Fase Atual:** Issue #4 - Handler /sbsd (Concluída)

---

## Issues

### ✅ Issue #1 - Estrutura Base
**Status:** ✅ CONCLUÍDA
**Arquivo:** [01-estrutura-base.md](01-estrutura-base.md)

**Implementado:**
- ✅ Estrutura de diretórios criada (cmd/, internal/)
- ✅ `internal/config/config.go` - Sistema de configuração via env vars
- ✅ `internal/response/types.go` - Tipos de response padronizados
- ✅ `cmd/server/main.go` - Entry point do servidor
- ✅ Build validado: `go build ./cmd/server/` ✓
- ✅ Vet validado: `go vet ./...` ✓

**Decisões Técnicas:**
- Mantido `module gerador_cookies` para compatibilidade com imports existentes
- Entry point em `cmd/server/main.go` ao invés de raiz
- Placeholders (.gitkeep) criados para handler/, middleware/, service/
- Config com defaults sensatos (porta 9999, timeouts adequados)
- Response types preparados para erros detalhados (step, provider, retryable)

**Próximos Passos:**
- Aguardando validação do usuário antes de prosseguir para Issue #2

---

### ✅ Issue #2 - Sistema de Erros Detalhados
**Status:** ✅ CONCLUÍDA
**Arquivo:** [02-sistema-erros.md](02-sistema-erros.md)
**Dependências:** Issue #1 ✅

**Implementado:**
- ✅ Package `internal/errors/` criado
- ✅ `errors.go` - 11 StepCodes mapeados com metadados (descrição, HTTP status, retryable)
- ✅ `SolverError` struct com métodos Error(), HTTPStatus(), IsRetryable(), etc.
- ✅ Constructors específicos para cada tipo de erro
- ✅ `converter.go` - Método ToErrorResponse() para conversão JSON
- ✅ Helpers WithPartialCookies() e WithDebug()
- ✅ Build validado: `go build ./internal/errors/` ✓
- ✅ Vet validado: `go vet ./...` ✓

**Decisões Técnicas:**
- Cada step tem número sequencial, descrição em português, HTTP status e flag retryable
- SolverError rastreia tempo de início para calcular elapsed_ms automaticamente
- Suporte a tentativas (attempt/maxAttempts) para erros de retry
- Conversão automática para response.ErrorResponse com todos os campos
- Header Retry-After já implementado em response.WriteError()

**Próximos Passos:**
- Aguardando validação do usuário antes de prosseguir para Issue #3

---

### ✅ Issue #3 - Migrar API Keys para Env Vars
**Status:** ✅ CONCLUÍDA
**Arquivo:** [03-migrar-api-keys.md](03-migrar-api-keys.md)
**Dependências:** Issue #1 ✅

**Implementado:**
- ✅ Campos de API keys adicionados em `internal/config/config.go`
- ✅ Campos de API keys adicionados em `scraper/scraper.go` Config
- ✅ API key hardcoded do Jevi removida de `abck_solver.go` (linha 387)
- ✅ API key hardcoded do Jevi removida de `sbsd_solver.go` (linha 218)
- ✅ API key hardcoded do Roolink removida de `abck_solver.go` (linha 558)
- ✅ API key hardcoded do Roolink removida de `sbsd_solver.go` (linha 407)
- ✅ Validação em runtime quando provider chamado sem key configurada
- ✅ Build validado: `go build ./...` ✓
- ✅ Vet validado: `go vet ./...` ✓
- ✅ Verificado: nenhuma key hardcoded restante no código

**Decisões Técnicas:**
- Keys carregadas via env vars: JEVI_API_KEY, N4S_API_KEY, ROOLINK_API_KEY
- Sem defaults - keys devem ser explicitamente configuradas
- Erro claro em runtime se provider chamado sem key: "JEVI_API_KEY not configured"
- ABCKSolver e SBSDSolver já tinham campo `config *Config`, facilitou migração

**Próximos Passos:**
- Aguardando validação do usuário antes de prosseguir para Issue #4

---

### ✅ Issue #4 - Handler /sbsd
**Status:** ✅ CONCLUÍDA
**Arquivo:** [04-handler-sbsd.md](04-handler-sbsd.md)
**Dependências:** Issues #1, #2, #3

**Implementado:**
- ✅ `internal/service/solver.go` - SolverService base com helpers
- ✅ `internal/service/sbsd.go` - Fluxo completo GenerateSbsd (8 steps)
- ✅ `internal/handler/sbsd.go` - SbsdHandler com validação e defaults
- ✅ Endpoint `POST /sbsd` registrado em `cmd/server/main.go`
- ✅ Integração com sistema de erros detalhados
- ✅ Suporte a cookies parciais em caso de erro
- ✅ Build validado: `go build ./cmd/server/` ✓
- ✅ Vet validado: `go vet ./...` ✓

**Decisões Técnicas:**
- Service layer criado para orquestrar scraper
- Handler valida request, aplica defaults e delega para service
- Fluxo de 8 steps implementado conforme especificação
- Erros mapeados para SolverError com step específico
- Cookies parciais coletados em cada ponto de falha
- Telemetria com abck_token, bm_sz_encoded, bm_s_encoded

**Próximos Passos:**
- Issue #5 - Handler /abck (similar ao /sbsd)

---

### ⏳ Issue #5 - Handler /abck
**Status:** PENDENTE
**Arquivo:** [05-handler-abck.md](05-handler-abck.md)
**Dependências:** Issues #1, #2, #3

---

### ⏳ Issue #6 - Handler /search
**Status:** PENDENTE
**Arquivo:** [06-handler-search.md](06-handler-search.md)
**Dependências:** Issues #4, #5

---

### ⏳ Issue #7 - Health Checks + Graceful Shutdown
**Status:** PENDENTE
**Arquivo:** [07-health-graceful.md](07-health-graceful.md)
**Dependências:** Issue #1 ✅

---

### ⏳ Issue #8 - Consolidar Funções Duplicadas
**Status:** PENDENTE
**Arquivo:** [08-consolidar-duplicatas.md](08-consolidar-duplicatas.md)
**Dependências:** Nenhuma (pode ser feita em paralelo)

---

## Checklist do Epic

- [x] Issue #1 - Estrutura Base
- [x] Issue #2 - Sistema de Erros
- [x] Issue #3 - Migrar API Keys
- [x] Issue #4 - Handler /sbsd
- [ ] Issue #5 - Handler /abck
- [ ] Issue #6 - Handler /search
- [ ] Issue #7 - Health + Graceful Shutdown
- [ ] Issue #8 - Consolidar Duplicatas
- [ ] Build final: `go build ./cmd/server/`
- [ ] Vet final: `go vet ./...`
- [ ] Teste manual de todos os endpoints
- [ ] Deploy em PM2 local
