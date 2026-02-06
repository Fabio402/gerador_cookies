# [EPIC] Refatorar API HTTP - Endpoints /abck e /sbsd com fluxos completos

**Etiqueta:** `Improvement`
**Prioridade:** Alta
**Estimativa:** Média complexidade

---

## Resumo

Refatorar o servidor HTTP de geração de cookies Akamai para:
1. Separar a lógica em 3 endpoints distintos (`/sbsd`, `/abck`, `/search`)
2. Integrar com a TLS-API externa
3. Implementar tratamento de erros detalhado indicando exatamente onde e por que cada falha ocorreu
4. Preparar para produção em PM2/EC2

---

## Contexto

O código atual possui um único endpoint `/search` com ~400 linhas que lida com ambos os fluxos (ABCK e SbSd) de forma condicional. Isso dificulta manutenção, debug e evolução do código.

---

## Objetivos

- Endpoints separados e focados (`/sbsd`, `/abck`, `/search`)
- Cada endpoint executa fluxo completo e retorna todos os cookies gerados
- Erros indicam exatamente o step onde falhou, com contexto completo
- Estrutura de código limpa seguindo padrões Go
- Pronto para produção em PM2/EC2 com graceful shutdown

---

## Arquitetura Proposta

```
gerador_cookies/
├── cmd/
│   └── server/
│       └── main.go              # Entry point (~60 linhas)
├── internal/
│   ├── config/
│   │   └── config.go            # Configuração via env vars
│   ├── handler/
│   │   ├── sbsd.go              # POST /sbsd
│   │   ├── abck.go              # POST /abck
│   │   ├── search.go            # POST /search (compatibilidade)
│   │   └── health.go            # GET /health, GET /ready
│   ├── middleware/
│   │   ├── logging.go           # Request logging com request ID
│   │   └── timeout.go           # Context deadline por request
│   ├── response/
│   │   └── types.go             # Tipos de response padronizados
│   └── service/
│       └── solver.go            # Orquestração do scraper
├── scraper/                     # Pacote existente (ajustes menores)
└── akt/                         # Pacote existente
```

---

## Sub-tarefas

| # | Issue | Prioridade | Dependência |
|---|-------|------------|-------------|
| 1 | [Estrutura base](01-estrutura-base.md) | Alta | - |
| 2 | [Sistema de erros detalhados](02-sistema-erros.md) | Alta | #1 |
| 3 | [Migrar API keys para env vars](03-migrar-api-keys.md) | Alta | #1 |
| 4 | [Handler /sbsd](04-handler-sbsd.md) | Alta | #1, #2, #3 |
| 5 | [Handler /abck](05-handler-abck.md) | Alta | #1, #2, #3 |
| 6 | [Handler /search](06-handler-search.md) | Média | #4, #5 |
| 7 | [Health checks + graceful shutdown](07-health-graceful.md) | Média | #1 |
| 8 | [Consolidar funções duplicadas](08-consolidar-duplicatas.md) | Baixa | - |

---

## Critérios de Aceitação do Epic

- [ ] Todas as sub-tarefas concluídas
- [ ] Build funciona: `go build ./cmd/server/`
- [ ] Sem warnings: `go vet ./...`
- [ ] Servidor inicia e responde em todos os endpoints
- [ ] Testado manualmente com requests de sucesso e erro
- [ ] Funcionando em PM2 local antes de deploy

---

## Ambiente de Deploy

- **Runtime:** PM2
- **Infraestrutura:** EC2 AWS
- **Porta:** 9999 (ou configurável via env)
- **Dependência:** TLS-API rodando externamente
