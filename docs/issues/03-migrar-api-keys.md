# #3 - Migrar API keys para variáveis de ambiente

**Etiqueta:** `Improvement`
**Prioridade:** Alta
**Dependência:** #1
**Epic:** [Refatorar API HTTP](00-epic-refatorar-api-http.md)

---

## Descrição

Remover API keys hardcoded do código fonte e migrar para variáveis de ambiente. Isso é essencial para segurança e para permitir configuração diferente entre ambientes (dev, staging, prod).

---

## Problema Atual

API keys estão hardcoded nos seguintes arquivos:

| Arquivo | Linha | Provider | Problema |
|---------|-------|----------|----------|
| `scraper/abck_solver.go` | ~387 | Jevi | Key como string literal |
| `scraper/abck_solver.go` | ~558 | Roolink | Key em método return |
| `scraper/sbsd_solver.go` | ~218 | Jevi | Duplicação da key Jevi |

---

## Solução

### 1. Adicionar campos no Config

Atualizar `internal/config/config.go`:

```go
type Config struct {
    // ... campos existentes ...

    // Provider API Keys
    JeviAPIKey    string
    N4SAPIKey     string
    RoolinkAPIKey string
}

func Load() (*Config, error) {
    cfg := &Config{
        // ... campos existentes ...

        // Provider API Keys (sem defaults - devem ser configurados)
        JeviAPIKey:    getEnv("JEVI_API_KEY", ""),
        N4SAPIKey:     getEnv("N4S_API_KEY", ""),
        RoolinkAPIKey: getEnv("ROOLINK_API_KEY", ""),
    }

    // Validação (warning, não erro - permite rodar sem todos os providers)
    if cfg.JeviAPIKey == "" {
        log.Println("WARNING: JEVI_API_KEY not set - Jevi provider will fail")
    }
    if cfg.N4SAPIKey == "" {
        log.Println("WARNING: N4S_API_KEY not set - N4S provider will fail")
    }
    if cfg.RoolinkAPIKey == "" {
        log.Println("WARNING: ROOLINK_API_KEY not set - Roolink provider will fail")
    }

    return cfg, nil
}
```

### 2. Passar keys via construtor do Scraper

Atualizar `scraper/scraper.go`:

```go
type Config struct {
    // ... campos existentes ...

    // Provider API Keys
    JeviAPIKey    string
    N4SAPIKey     string
    RoolinkAPIKey string
}
```

### 3. Atualizar solvers para usar config

Atualizar `scraper/abck_solver.go`:

```go
// Antes (hardcoded)
func (s *ABCKSolver) jeviAPIKey() string {
    return "xxxx-hardcoded-key-xxxx"
}

// Depois (via config)
func (s *ABCKSolver) jeviAPIKey() string {
    return s.scraper.config.JeviAPIKey
}

func (s *ABCKSolver) roolinkAPIKey() string {
    return s.scraper.config.RoolinkAPIKey
}

func (s *ABCKSolver) n4sAPIKey() string {
    return s.scraper.config.N4SAPIKey
}
```

Fazer o mesmo para `scraper/sbsd_solver.go`.

---

## Variáveis de Ambiente

| Variável | Descrição | Obrigatório |
|----------|-----------|-------------|
| `JEVI_API_KEY` | API key do provider Jevi (jevi.dev) | Sim (se usar Jevi) |
| `N4S_API_KEY` | API key do provider N4S (n4s.xyz) | Sim (se usar N4S) |
| `ROOLINK_API_KEY` | API key do provider Roolink (roolink.io) | Sim (se usar Roolink) |

---

## PM2 Ecosystem File

Atualizar `ecosystem.config.js`:

```json
{
  "apps": [
    {
      "name": "gerador-cookies",
      "script": "./server",
      "env": {
        "SERVER_PORT": "9999",
        "TLS_API_URL": "http://localhost:8080",
        "TLS_API_TOKEN": "",
        "REQS_PROVIDER_CACHE_ENABLE": "1",
        "JEVI_API_KEY": "${JEVI_API_KEY}",
        "N4S_API_KEY": "${N4S_API_KEY}",
        "ROOLINK_API_KEY": "${ROOLINK_API_KEY}"
      }
    }
  ]
}
```

> **Nota:** As variáveis `${JEVI_API_KEY}` etc. devem ser definidas no ambiente da EC2 (via AWS Secrets Manager, Parameter Store, ou arquivo `.env` protegido).

---

## Validação de Provider em Runtime

Adicionar validação quando um provider é chamado sem key configurada:

```go
func (s *ABCKSolver) callJevi(script string) (string, error) {
    key := s.jeviAPIKey()
    if key == "" {
        return "", fmt.Errorf("JEVI_API_KEY not configured")
    }
    // ... resto da implementação
}
```

---

## Arquivos a Modificar

| Arquivo | Alteração |
|---------|-----------|
| `internal/config/config.go` | Adicionar campos de API keys |
| `scraper/scraper.go` | Adicionar campos ao Config struct |
| `scraper/abck_solver.go` | Remover keys hardcoded, usar config |
| `scraper/sbsd_solver.go` | Remover keys hardcoded, usar config |

---

## Critérios de Aceitação

- [ ] Nenhuma API key hardcoded no código
- [ ] Keys lidas de variáveis de ambiente
- [ ] Warning no startup se keys não configuradas
- [ ] Erro claro se provider chamado sem key
- [ ] `git diff` não mostra keys em nenhum arquivo
- [ ] Documentação de variáveis de ambiente atualizada

---

## Segurança

- **Nunca** commitar keys no repositório
- Adicionar ao `.gitignore`:
  ```
  .env
  .env.local
  ecosystem.config.js
  ```
- Usar AWS Secrets Manager ou Parameter Store em produção
- Rodar `git log -p | grep -i "api_key"` para verificar histórico

---

## Validação

```bash
# Testar sem keys (deve dar warning)
unset JEVI_API_KEY N4S_API_KEY ROOLINK_API_KEY
go run ./cmd/server/
# Output: WARNING: JEVI_API_KEY not set...

# Testar com keys
export JEVI_API_KEY="test-key"
go run ./cmd/server/
# Deve iniciar sem warning para Jevi

# Testar chamada sem key configurada
curl -X POST http://localhost:9999/abck \
  -H "Content-Type: application/json" \
  -d '{"url": "example.com", "akamaiProvider": "roolink"}'
# Deve retornar erro: "ROOLINK_API_KEY not configured"
```
