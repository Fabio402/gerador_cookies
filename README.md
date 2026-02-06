# Gerador de Cookies Akamai

Biblioteca Go para geração de cookies de bypass da proteção anti-bot Akamai. Suporta múltiplos providers de sensor e perfis de navegador para fingerprinting TLS realista.

## Índice

- [Requisitos](#requisitos)
- [Instalação](#instalação)
- [Configuração](#configuração)
- [Uso Básico](#uso-básico)
- [Exemplos](#exemplos)
- [Variáveis de Ambiente](#variáveis-de-ambiente)
- [Providers Suportados](#providers-suportados)
- [Perfis de Navegador](#perfis-de-navegador)
- [Tratamento de Erros](#tratamento-de-erros)
- [Estrutura do Projeto](#estrutura-do-projeto)

## Requisitos

- Go 1.24.1+
- Serviço TLS-API rodando (padrão: `localhost:8080`)
- Acesso aos providers de sensor (jevi.dev, n4s.xyz, roolink.io)

## Instalação

```bash
go get github.com/seu-usuario/gerador_cookies
```

Ou adicione como dependência no seu `go.mod`:

```go
require gerador_cookies v0.0.0
```

## Configuração

### Struct Config

```go
type Config struct {
    // Domínio e URLs
    Domain           string  // Domínio alvo (ex: "example.com")
    SensorUrl        string  // Endpoint do script anti-bot (ex: "/on/abck")

    // Configurações do Provider
    AkamaiProvider   string  // "jevi", "n4s" ou "roolink"
    SbSdProvider     string  // Provider alternativo para SBSD (opcional)
    SbSd             bool    // Usar fluxo SBSD ao invés de ABCK

    // Parâmetros de Request
    SensorPostLimit  int     // Número de tentativas de retry
    Language         string  // Header Accept-Language (ex: "pt-BR")
    UserAgent        string  // User-Agent customizado
    SecChUa          string  // Header Sec-CH-UA
    ProfileType      string  // "chrome_133", "firefox_135", "safari_ios_18_5"

    // Comportamento
    LowSecurity      bool    // Validação de cookie menos rigorosa
    UseScript        bool    // Incluir script nas requests
    ForceUpdateDynamics bool // Ignorar cache de dados dinâmicos

    // Debug
    GenerateReport   bool    // Gerar relatório em /tmp/getsensor-report-*.txt

    // TLS-API
    TLSAPIBrowser    string  // Perfil de navegador para TLS-API
    Proxy            string  // URL do proxy
}
```

## Uso Básico

### 1. Criar o Scraper

```go
package main

import (
    "log"
    "gerador_cookies/scraper"
)

func main() {
    config := &scraper.Config{
        Domain:          "example.com",
        SensorUrl:       "/on/abck",
        AkamaiProvider:  "jevi",
        UserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        Language:        "pt-BR",
        SensorPostLimit: 5,
        ProfileType:     "chrome_133",
    }

    s, err := scraper.NewScraper("http://proxy:port", config)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. Gerar Cookie ABCK

```go
// Buscar homepage (seta cookies iniciais)
resp, err := s.GetHomepage()
if err != nil {
    log.Fatal(err)
}

// Extrair URL do script anti-bot
scriptURL, err := s.GetAntiBotScriptURL("")
if err != nil {
    log.Fatal(err)
}

// Obter conteúdo do script
script, err := s.GetAntiBotScript()
if err != nil {
    log.Fatal(err)
}

// Gerar cookie ABCK
result, err := s.GenerateABCK(script)
if err != nil {
    log.Fatal(err)
}

if result.Success {
    log.Printf("Sucesso! Cookie: %s", result.CookieString)
    // Usar result.Cookies para requests subsequentes
}
```

### 3. Gerar Challenge SBSD

```go
config := &scraper.Config{
    Domain:         "example.com",
    SensorUrl:      "/on/abck",
    AkamaiProvider: "jevi",
    SbSd:           true, // Ativar modo SBSD
}

s, _ := scraper.NewScraper("http://proxy:port", config)

// bmSo é extraído do HTML da página quando há challenge
result, err := s.GenerateSBSD(script, bmSo)
if result.Success {
    log.Printf("SBSD Cookie: %s", result.CookieString)
}
```

## Exemplos

### Exemplo Completo

```go
package main

import (
    "fmt"
    "log"
    "gerador_cookies/scraper"
)

func main() {
    // Configuração
    config := &scraper.Config{
        Domain:          "www.nike.com.br",
        SensorUrl:       "/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/ips.js",
        AkamaiProvider:  "jevi",
        ProfileType:     "chrome_133",
        Language:        "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7",
        SensorPostLimit: 3,
        GenerateReport:  true, // Gera relatório para debug
    }

    // Criar scraper com proxy
    s, err := scraper.NewScraper("http://user:pass@proxy.example.com:8080", config)
    if err != nil {
        log.Fatalf("Erro ao criar scraper: %v", err)
    }

    // Fluxo completo
    fmt.Println("1. Buscando homepage...")
    homepage, err := s.GetHomepage()
    if err != nil {
        log.Fatalf("Erro na homepage: %v", err)
    }
    fmt.Printf("   Status: %d\n", homepage.Status)

    fmt.Println("2. Extraindo script...")
    script, err := s.GetAntiBotScript()
    if err != nil {
        log.Fatalf("Erro no script: %v", err)
    }
    fmt.Printf("   Script size: %d bytes\n", len(script))

    fmt.Println("3. Gerando ABCK...")
    result, err := s.GenerateABCK(script)
    if err != nil {
        log.Fatalf("Erro no ABCK: %v", err)
    }

    if result.Success {
        fmt.Println("✓ Cookie gerado com sucesso!")
        fmt.Printf("  Cookie: %s\n", result.CookieString)
        fmt.Printf("  Provider: %s\n", result.Session.Provider)
        fmt.Printf("  Browser: %s\n", result.Session.Browser)
    } else {
        fmt.Printf("✗ Falha: %s\n", result.Error.RawError)
        fmt.Printf("  Fase: %s\n", result.Error.Phase)
        fmt.Printf("  Retentável: %v\n", result.Error.Retryable)
    }
}
```

### Usando Cookies em Requests HTTP

```go
// Após gerar o cookie
result, _ := s.GenerateABCK(script)

if result.Success {
    // Opção 1: Usar string diretamente no header
    req, _ := http.NewRequest("GET", "https://example.com/api", nil)
    req.Header.Set("Cookie", result.CookieString)

    // Opção 2: Usar []*http.Cookie com jar
    jar, _ := cookiejar.New(nil)
    u, _ := url.Parse("https://example.com")
    jar.SetCookies(u, result.Cookies)

    client := &http.Client{Jar: jar}
    client.Do(req)
}
```

### Gerenciamento de Cookies do Scraper

```go
// Obter todos os cookies do domínio
cookies := s.GetCookies()

// Obter cookies como string
cookieStr := s.GetCookieString("https://example.com")

// Definir cookies manualmente
s.SetCookies("https://example.com", []*http.Cookie{
    {Name: "session", Value: "abc123"},
})
```

## Variáveis de Ambiente

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| `TLS_API_URL` | URL base do serviço TLS-API | `http://localhost:8080` |
| `TLS_API_TOKEN` | Token de autorização para TLS-API | - |
| `isDebug` | Ativa logs de debug | `false` |
| `DEBUG_PROXY` | Proxy para debug (Charles, Burp) | - |
| `REQS_PROVIDER_CACHE_ENABLE` | Força ativação do cache | - |
| `REQS_PROVIDER_CACHE_DISABLE` | Desativa cache | - |
| `REQS_PROVIDER_CACHE_CLEAR_ON_START` | Limpa cache ao iniciar | - |

### Exemplo de Configuração

```bash
export TLS_API_URL=http://localhost:8080
export TLS_API_TOKEN=seu-token-aqui
export isDebug=true
```

## Providers Suportados

| Provider | Endpoint | Descrição |
|----------|----------|-----------|
| `jevi` | jevi.dev | Provider principal |
| `n4s` | n4s.xyz | Provider alternativo |
| `roolink` | roolink.io | Provider alternativo |

### Configurando Provider

```go
// Usar Jevi (recomendado)
config.AkamaiProvider = "jevi"

// Usar N4S
config.AkamaiProvider = "n4s"

// Usar Roolink
config.AkamaiProvider = "roolink"
```

## Perfis de Navegador

| Perfil | Descrição |
|--------|-----------|
| `chrome_133` | Chrome 133 (padrão) |
| `firefox_135` | Firefox 135 |
| `safari_ios_18_5` | Safari iOS 18.5 |

```go
config.ProfileType = "chrome_133"    // Windows Chrome
config.ProfileType = "firefox_135"   // Firefox
config.ProfileType = "safari_ios_18_5" // Mobile Safari
```

## Tratamento de Erros

### Estrutura de Erro

```go
type SolverError struct {
    Phase      ErrorPhase // Fase onde ocorreu o erro
    Step       string     // Descrição do passo
    Provider   string     // Provider envolvido
    Domain     string     // Domínio alvo
    StatusCode int        // Código HTTP
    RawError   string     // Mensagem original
    Retryable  bool       // Se pode tentar novamente
}
```

### Fases de Erro

| Fase | Descrição |
|------|-----------|
| `PhaseInit` | Inicialização |
| `PhaseHomepage` | Busca da homepage |
| `PhaseScriptExtract` | Extração da URL do script |
| `PhaseScriptFetch` | Download do script |
| `PhaseProviderCall` | Chamada ao provider |
| `PhaseSensorPost` | Envio do sensor |
| `PhaseCookieValidation` | Validação do cookie |
| `PhaseSBSDPost` | Envio do SBSD |
| `PhaseTLSAPI` | Comunicação com TLS-API |

### Exemplo de Tratamento

```go
result, err := s.GenerateABCK(script)

if !result.Success {
    switch result.Error.Phase {
    case scraper.PhaseProviderCall:
        log.Printf("Erro no provider %s: %s", result.Error.Provider, result.Error.RawError)
        if result.Error.Retryable {
            // Tentar com outro provider
        }
    case scraper.PhaseTLSAPI:
        log.Printf("TLS-API indisponível: %s", result.Error.RawError)
    default:
        log.Printf("Erro: %s", result.Error.RawError)
    }
}
```

## Estrutura do Projeto

```
gerador_cookies/
├── go.mod                  # Definição do módulo Go
├── go.sum                  # Checksums das dependências
├── README.md               # Esta documentação
├── akt/
│   └── logger.go           # Utilitários de logging
└── scraper/
    ├── scraper.go          # Implementação principal
    ├── abck_solver.go      # Fluxo de geração ABCK
    ├── sbsd_solver.go      # Fluxo de challenge SBSD
    ├── tls_api_client.go   # Cliente TLS-API
    ├── site_client.go      # Cliente para requests aos sites
    ├── cookie_jar.go       # Gerenciamento de cookies
    ├── provider_cache.go   # Cache de providers
    ├── types.go            # Tipos e estruturas
    ├── errors.go           # Tratamento de erros
    └── utils.go            # Funções utilitárias
```

## Tipos de Retorno

### ABCKResult

```go
type ABCKResult struct {
    Success      bool           // Se geração teve sucesso
    Cookies      []*http.Cookie // Cookies gerados
    CookieString string         // Formato "name=value; ..."
    Session      SessionInfo    // Metadados da sessão
    Error        *SolverError   // Detalhes do erro (se falhou)
}
```

### SBSDResult

```go
type SBSDResult struct {
    Success      bool           // Se geração teve sucesso
    Cookies      []*http.Cookie // Cookies gerados
    CookieString string         // Formato "name=value; ..."
    Session      SessionInfo    // Metadados da sessão
    Error        *SolverError   // Detalhes do erro (se falhou)
}
```

### SessionInfo

```go
type SessionInfo struct {
    Proxy       string    // Proxy utilizado
    UserAgent   string    // User-Agent utilizado
    Browser     string    // Perfil TLS (ex: "chrome_133")
    Domain      string    // Domínio alvo
    Provider    string    // Provider (jevi, n4s, roolink)
    GeneratedAt time.Time // Timestamp da geração
}
```

## Sistema de Cache

O cache de providers é armazenado em:
- Linux/macOS: `~/.cache/reqs/provider-cache.json`
- Fallback: `/tmp/reqs-provider-cache.json`

**Características:**
- Expira em 24 horas
- Armazena URLs de scripts e dados dinâmicos por domínio/provider
- Pode ser controlado via variáveis de ambiente

## Licença

Uso interno - Todos os direitos reservados.
