#!/bin/bash

# ==============================================================================
# CURL Examples - Gerador Cookies API
# ==============================================================================

BASE_URL="http://localhost:9999"

# Cores para output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Gerador Cookies - Exemplos de Requisições ===${NC}\n"

# ==============================================================================
# 1. TESTE BÁSICO - Requisição mínima (usa defaults)
# ==============================================================================
echo -e "${GREEN}1. Teste Básico (apenas URL)${NC}"
curl -X POST $BASE_URL/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br"
  }' | jq '.'

echo -e "\n---\n"

# ==============================================================================
# 2. TESTE COM N4S PROVIDER
# ==============================================================================
echo -e "${GREEN}2. Teste com N4S Provider${NC}"
curl -X POST $BASE_URL/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "akamaiProvider": "n4s"
  }' | jq '.'

echo -e "\n---\n"

# ==============================================================================
# 3. TESTE COM ROOLINK PROVIDER
# ==============================================================================
echo -e "${GREEN}3. Teste com Roolink Provider${NC}"
curl -X POST $BASE_URL/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "akamaiProvider": "roolink"
  }' | jq '.'

echo -e "\n---\n"

# ==============================================================================
# 4. TESTE COMPLETO - Todos os parâmetros
# ==============================================================================
echo -e "${GREEN}4. Teste Completo (todos os parâmetros)${NC}"
curl -X POST $BASE_URL/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "akamaiUrl": "/path/to/script.js",
    "proxy": "http://user:pass@proxy.example.com:8080",
    "randomUserAgent": "chrome_144",
    "userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36",
    "secChUa": "\"Not(A:Brand\";v=\"8\", \"Chromium\";v=\"144\", \"Google Chrome\";v=\"144\"",
    "language": "pt-BR",
    "akamaiProvider": "n4s",
    "generateReport": true
  }' | jq '.'

echo -e "\n---\n"

# ==============================================================================
# 5. TESTE COM DIFERENTES PROFILES TLS
# ==============================================================================
echo -e "${GREEN}5. Teste com Chrome 120${NC}"
curl -X POST $BASE_URL/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "randomUserAgent": "chrome_120",
    "akamaiProvider": "n4s"
  }' | jq '.'

echo -e "\n---\n"

echo -e "${GREEN}6. Teste com Firefox 133${NC}"
curl -X POST $BASE_URL/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "randomUserAgent": "firefox_133",
    "akamaiProvider": "n4s"
  }' | jq '.'

echo -e "\n---\n"

# ==============================================================================
# 7. TESTE DE ERRO - URL vazia
# ==============================================================================
echo -e "${RED}7. Teste de Erro - URL vazia${NC}"
curl -X POST $BASE_URL/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": ""
  }' | jq '.'

echo -e "\n---\n"

# ==============================================================================
# 8. TESTE DE ERRO - JSON inválido
# ==============================================================================
echo -e "${RED}8. Teste de Erro - JSON inválido${NC}"
curl -X POST $BASE_URL/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br"
    "invalid": json
  }'

echo -e "\n---\n"

# ==============================================================================
# 9. TESTE COM PROXY
# ==============================================================================
echo -e "${GREEN}9. Teste com Proxy${NC}"
curl -X POST $BASE_URL/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "proxy": "http://username:password@proxy.example.com:8080",
    "akamaiProvider": "n4s"
  }' | jq '.'

echo -e "\n---\n"

# ==============================================================================
# 10. TESTE COM GERAÇÃO DE REPORT
# ==============================================================================
echo -e "${GREEN}10. Teste com Geração de Report${NC}"
curl -X POST $BASE_URL/sbsd \
  -H "Content-Type: application/json" \
  -d '{
    "url": "www.nike.com.br",
    "akamaiProvider": "n4s",
    "generateReport": true
  }' | jq '.'

echo -e "\n${BLUE}=== Testes Concluídos ===${NC}\n"
