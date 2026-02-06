# Guia de Testes - Gerador Cookies API

## Requisitos

- Servidor rodando: `go run cmd/server/main.go`
- Variáveis de ambiente configuradas (`.env`)
- `jq` instalado (opcional, para formatar JSON): `sudo apt install jq`

---

## 1. Teste Básico (Mínimo)

```bash
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br"
  }'
```

**Defaults aplicados:**
- `randomUserAgent`: `chrome_144`
- `language`: `en-US`
- `akamaiProvider`: `jevi`

---

## 2. Teste com N4S Provider

```bash
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "akamaiProvider": "n4s"
  }'
```

---

## 3. Teste com Roolink Provider

```bash
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "akamaiProvider": "roolink"
  }'
```

---

## 4. Teste Completo (Todos os Parâmetros)

```bash
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "akamaiUrl": "/path/to/script.js",
    "proxy": "http://user:pass@proxy.example.com:8080",
    "randomUserAgent": "chrome_144",
    "userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
    "secChUa": "\"Not(A:Brand\";v=\"8\", \"Chromium\";v=\"144\"",
    "language": "pt-BR",
    "akamaiProvider": "n4s",
    "generateReport": true
  }'
```

---

## 5. Teste com Diferentes Profiles TLS

### Chrome 144 (default)
```bash
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "randomUserAgent": "chrome_144"
  }'
```

### Chrome 120
```bash
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "randomUserAgent": "chrome_120"
  }'
```

### Firefox 133
```bash
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "randomUserAgent": "firefox_133"
  }'
```

---

## 6. Teste com Proxy

```bash
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "proxy": "http://username:password@proxy.example.com:8080",
    "akamaiProvider": "n4s"
  }'
```

---

## 7. Teste com Geração de Report

```bash
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "akamaiProvider": "n4s",
    "generateReport": true
  }'
```

**Response esperado:**
```json
{
  "success": true,
  "cookies": { ... },
  "telemetry": { ... },
  "session": { ... },
  "debug": {
    "report_path": "/path/to/report.html"
  }
}
```

---

## 8. Testes de Erro

### URL vazia
```bash
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": ""
  }'
```

**Response esperado:**
```json
{
  "success": false,
  "error": {
    "step": "request_validation",
    "step_number": 0,
    "description": "Campo 'url' é obrigatório",
    "raw_error": "missing required field: url",
    "retryable": false,
    "http_status": 400
  }
}
```

### JSON inválido
```bash
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{ invalid json }'
```

**Response esperado:**
```json
{
  "success": false,
  "error": {
    "step": "request_decode",
    "step_number": 0,
    "description": "Falha ao decodificar request JSON",
    "retryable": false,
    "http_status": 400
  }
}
```

### Provider sem API Key
```bash
# Sem N4S_API_KEY configurado
curl -X POST http://localhost:9999/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "akamaiProvider": "n4s"
  }'
```

**Response esperado:**
```json
{
  "success": false,
  "error": {
    "step": "sbsd_generation",
    "step_number": 6,
    "description": "Falha ao gerar challenge SbSd",
    "raw_error": "N4S_API_KEY not configured",
    "retryable": false,
    "http_status": 518
  },
  "partial_cookies": { ... }
}
```

---

## 9. Response de Sucesso Esperado

```json
{
  "success": true,
  "cookies": {
    "full_string": "_abck=...; bm_sz=...; bm_s=...",
    "items": [
      {
        "name": "_abck",
        "value": "...",
        "domain": "www.nike.com.br"
      },
      {
        "name": "bm_sz",
        "value": "...",
        "domain": "www.nike.com.br"
      },
      {
        "name": "bm_s",
        "value": "...",
        "domain": "www.nike.com.br"
      }
    ]
  },
  "telemetry": {
    "abck_token": "...",
    "bm_sz_encoded": "...",
    "bm_s_encoded": "..."
  },
  "session": {
    "provider": "n4s",
    "profile": "chrome_144"
  }
}
```

---

## 10. Executar Todos os Testes

```bash
# Dar permissão de execução
chmod +x curl-examples.sh

# Executar script de testes
./curl-examples.sh
```

---

## Providers Disponíveis

| Provider | Variável de Ambiente | Status |
|----------|---------------------|--------|
| `jevi` | `JEVI_API_KEY` | Default |
| `n4s` | `N4S_API_KEY` | ✅ Configurado |
| `roolink` | `ROOLINK_API_KEY` | ✅ Configurado |

---

## Troubleshooting

### Erro: "connection refused"
```bash
# Verificar se o servidor está rodando
ps aux | grep "go run"

# Iniciar o servidor
source .env && go run cmd/server/main.go
```

### Erro: "API_KEY not configured"
```bash
# Verificar variáveis de ambiente
echo $N4S_API_KEY
echo $ROOLINK_API_KEY

# Carregar .env
source .env
```

### Ver logs do servidor
```bash
# Logs aparecem no terminal onde o servidor foi iniciado
# Procure por linhas como:
# [INFO] Starting server on :9999
# [ERROR] ...
```
