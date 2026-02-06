# #8 - Consolidar funções duplicadas

**Etiqueta:** `Improvement`
**Prioridade:** Baixa
**Dependência:** Nenhuma (pode ser feito em paralelo)
**Epic:** [Refatorar API HTTP](00-epic-refatorar-api-http.md)

---

## Descrição

Remover código duplicado entre `abck_solver.go` e `sbsd_solver.go`, consolidando funções comuns em `utils.go` ou em uma struct compartilhada.

---

## Duplicações Identificadas

### 1. `compressGzip`

**Localização:**
- `scraper/abck_solver.go` → `abckCompressGzip`
- `scraper/sbsd_solver.go` → `sbsdCompressGzip`

**Código (idêntico em ambos):**
```go
func abckCompressGzip(data []byte) (string, error) {
    var buf bytes.Buffer
    gz := gzip.NewWriter(&buf)
    if _, err := gz.Write(data); err != nil {
        return "", err
    }
    if err := gz.Close(); err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
```

**Solução:** Mover para `scraper/utils.go` como `compressGzip`.

---

### 2. `providerLanguage`

**Localização:**
- `scraper/abck_solver.go` → `abckProviderLanguage`
- `scraper/sbsd_solver.go` → `sbsdProviderLanguage`

**Código (idêntico):**
```go
func (s *ABCKSolver) abckProviderLanguage() string {
    lang := s.scraper.config.Language
    if lang == "" {
        return "en-US"
    }
    // Parse primeiro idioma do Accept-Language
    if idx := strings.Index(lang, ","); idx > 0 {
        lang = lang[:idx]
    }
    return lang
}
```

**Solução:** Mover para `scraper/utils.go` como função ou para Config como método.

---

### 3. `buildHeaders` e `buildHeadersOrder`

**Localização:**
- `scraper/abck_solver.go:682-703`
- `scraper/sbsd_solver.go:424-445`

**Código (idêntico):**
```go
func (s *ABCKSolver) buildHeaders() map[string]string {
    return map[string]string{
        "Accept":          "*/*",
        "Accept-Language": s.scraper.config.Language,
        "Content-Type":    "text/plain;charset=UTF-8",
        "Origin":          fmt.Sprintf("https://%s", s.scraper.config.Domain),
        "Referer":         fmt.Sprintf("https://%s/", s.scraper.config.Domain),
        "User-Agent":      s.scraper.userAgent.UserAgent,
    }
}

func (s *ABCKSolver) buildHeadersOrder() []string {
    return []string{
        "Accept",
        "Accept-Language",
        "Content-Type",
        "Origin",
        "Referer",
        "User-Agent",
    }
}
```

**Solução:** Criar struct `BaseSolver` com métodos compartilhados, ou funções em `utils.go`.

---

### 4. Função `min` shadowing builtin

**Localização:**
- `scraper/scraper.go:601`

**Código:**
```go
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

**Problema:** Go 1.21+ tem `min` como builtin. Esta função faz shadowing.

**Solução:** Remover a função, usar o builtin.

---

## Implementação

### 1. Atualizar `scraper/utils.go`

```go
package scraper

import (
    "bytes"
    "compress/gzip"
    "encoding/base64"
    "fmt"
    "strings"
)

// compressGzip comprime dados usando gzip e retorna base64
func compressGzip(data []byte) (string, error) {
    var buf bytes.Buffer
    gz := gzip.NewWriter(&buf)
    if _, err := gz.Write(data); err != nil {
        return "", err
    }
    if err := gz.Close(); err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// parseLanguage extrai o primeiro idioma do Accept-Language header
func parseLanguage(acceptLanguage string) string {
    if acceptLanguage == "" {
        return "en-US"
    }
    if idx := strings.Index(acceptLanguage, ","); idx > 0 {
        return acceptLanguage[:idx]
    }
    return acceptLanguage
}

// buildSensorHeaders retorna headers e ordem para requests de sensor
func buildSensorHeaders(domain, language, userAgent string) (map[string]string, []string) {
    headers := map[string]string{
        "Accept":          "*/*",
        "Accept-Language": language,
        "Content-Type":    "text/plain;charset=UTF-8",
        "Origin":          fmt.Sprintf("https://%s", domain),
        "Referer":         fmt.Sprintf("https://%s/", domain),
        "User-Agent":      userAgent,
    }

    order := []string{
        "Accept",
        "Accept-Language",
        "Content-Type",
        "Origin",
        "Referer",
        "User-Agent",
    }

    return headers, order
}
```

### 2. Atualizar `scraper/abck_solver.go`

```go
// Antes
func (s *ABCKSolver) abckCompressGzip(data []byte) (string, error) {
    // ... código duplicado ...
}

// Depois
func (s *ABCKSolver) compress(data []byte) (string, error) {
    return compressGzip(data)
}

// Antes
func (s *ABCKSolver) buildHeaders() map[string]string {
    // ... código duplicado ...
}

// Depois
func (s *ABCKSolver) buildHeaders() map[string]string {
    headers, _ := buildSensorHeaders(
        s.scraper.config.Domain,
        s.scraper.config.Language,
        s.scraper.userAgent.UserAgent,
    )
    return headers
}

func (s *ABCKSolver) buildHeadersOrder() []string {
    _, order := buildSensorHeaders("", "", "")
    return order
}
```

### 3. Atualizar `scraper/sbsd_solver.go`

Fazer as mesmas alterações.

### 4. Remover `min` de `scraper/scraper.go`

```go
// REMOVER esta função (linha ~601)
// func min(a, b int) int {
//     if a < b {
//         return a
//     }
//     return b
// }

// Usar o builtin min() diretamente onde era chamado
```

---

## Checklist de Alterações

| Arquivo | Alteração |
|---------|-----------|
| `scraper/utils.go` | Adicionar `compressGzip`, `parseLanguage`, `buildSensorHeaders` |
| `scraper/abck_solver.go` | Remover `abckCompressGzip`, `abckProviderLanguage`, simplificar `buildHeaders`/`buildHeadersOrder` |
| `scraper/sbsd_solver.go` | Remover `sbsdCompressGzip`, `sbsdProviderLanguage`, simplificar `buildHeaders`/`buildHeadersOrder` |
| `scraper/scraper.go` | Remover função `min` (usar builtin) |

---

## Critérios de Aceitação

- [ ] `compressGzip` em um único lugar (`utils.go`)
- [ ] `parseLanguage` em um único lugar
- [ ] `buildSensorHeaders` em um único lugar
- [ ] Função `min` removida
- [ ] `go vet ./...` sem warnings
- [ ] `go build ./...` sem erros
- [ ] Testes existentes continuam passando
- [ ] Comportamento funcional inalterado

---

## Validação

```bash
# Verificar que não há mais duplicação
grep -r "CompressGzip" scraper/
# Deve retornar apenas utils.go

grep -r "ProviderLanguage" scraper/
# Deve retornar apenas utils.go (ou nenhum, se inlined)

# Build
go build ./...

# Vet
go vet ./...

# Test (se houver)
go test ./scraper/...
```

---

## Notas

Esta tarefa pode ser executada em paralelo com as outras, pois não afeta a estrutura dos handlers ou services. É uma refatoração interna do pacote `scraper/`.

A prioridade é baixa porque o código funciona mesmo com duplicação - é apenas uma questão de manutenibilidade.
