# Gerador Cookies API

Este serviço expõe rotas HTTP para gerar cookies Akamai utilizando os solvers já existentes no pacote `scraper`.

## Executando

```bash
go run ./cmd/api
```

Variáveis suportadas:

| Variável     | Descrição                              | Default |
|--------------|----------------------------------------|---------|
| `PORT`       | Porta de escuta (se `API_ADDR` vazio)  | `8080`  |
| `API_ADDR`   | Endereço completo `host:porta`         | `:8080` |
| `TLS_API_URL`| URL do serviço TLS-API                 | `http://localhost:8080` |

## Rotas

### `POST /api/v1/abck`

Gera o cookie `_abck`.

Payload:

```json
{
  "config": {
    "domain": "example.com",
    "sensorUrl": "/akam/12345",
    "akamaiProvider": "jevi",
    "sensorPostLimit": 5,
    "sbSd": false
  },
  "script": "base64_do_script",
  "proxyUrl": "http://user:pass@proxy:8080"
}
```

### `POST /api/v1/sbsd`

Executa todo o fluxo SBSD e retorna o cookie `1~2`.

Payload:

```json
{
  "config": {
    "domain": "shop.com",
    "sensorUrl": "/sbsd/script?v=uuid",
    "akamaiProvider": "n4s",
    "sbSdProvider": "n4s"
  },
  "script": "base64_do_script",
  "bmSo": "valor_do_bm_so",
  "proxyUrl": "http://user:pass@proxy:8080"
}
```

### Resposta

Ambas as rotas retornam:

```json
{
  "success": true,
  "cookieString": "1~2=....; _abck=....",
  "cookies": [
    {"name":"1","value":"~2", "domain":"example.com"}
  ],
  "session": {
    "provider": "jevi",
    "domain": "example.com",
    "generatedAt": "2026-02-02T20:00:00Z"
  },
  "error": null
}
```

Quando há falha de geração, `success` é `false`, `error` descreve o problema e o status HTTP é `502`.

## Testes

```bash
go test ./internal/api
```

> **Nota:** Execute os testes em um ambiente com o Go toolchain disponível.
