# API Specification

## Overview

This document specifies the internal library API and external provider API integrations.

---

## Library Public API

### Scraper

#### NewScraper

Creates a new Scraper instance.

```go
func NewScraper(proxyURL string, config *Config, profile profiles.ClientProfile) (*Scraper, error)
```

**Parameters**:

| Name | Type | Description |
|------|------|-------------|
| `proxyURL` | `string` | HTTP proxy URL (optional, empty for direct) |
| `config` | `*Config` | Scraper configuration |
| `profile` | `profiles.ClientProfile` | TLS client profile |

**Returns**: `*Scraper`, `error`

**Example**:
```go
scraper, err := scraper.NewScraper(
    "http://user:pass@proxy:8080",
    &scraper.Config{
        Domain:         "example.com",
        SensorUrl:      "/akam/13/pixel_xxx",
        SensorPostLimit: 5,
        AkamaiProvider: "jevi",
    },
    profiles.Chrome_133,
)
```

---

#### GetAntiBotScriptURL

Extracts the Akamai script URL from the target homepage.

```go
func (s *Scraper) GetAntiBotScriptURL(providedUrl string) (string, error)
```

**Parameters**:

| Name | Type | Description |
|------|------|-------------|
| `providedUrl` | `string` | Custom homepage URL (optional) |

**Returns**: Script URL path (e.g., `/akam/13/pixel_xxx`), `error`

**Notes**:
- If cache contains script URL, returns cached value
- Parses HTML `<script>` tags looking for Akamai patterns
- Different patterns for SbSd vs standard mode

---

#### GetAntiBotScript

Downloads and encodes the Akamai script.

```go
func (s *Scraper) GetAntiBotScript() (string, error)
```

**Returns**: Base64-encoded script content, `error`

**Notes**:
- If cached dynamic exists, only seeds cookies (minimal request)
- Handles gzip and brotli compression
- Uses `simpleClient` (no TLS fingerprinting) for download

---

#### GenerateSession

Generates a valid session by solving the Akamai challenge.

```go
func (s *Scraper) GenerateSession(script string) (bool, error)
```

**Parameters**:

| Name | Type | Description |
|------|------|-------------|
| `script` | `string` | Base64-encoded script from GetAntiBotScript |

**Returns**: `true` if successful, `false` if failed, `error` for fatal errors

**Notes**:
- Retries up to `SensorPostLimit` times
- Updates cookie jar on success
- Caches dynamic data for future use

---

#### GetCookies / SetCookies

Cookie management methods.

```go
func (s *Scraper) GetCookies(urlStr string) ([]*http.Cookie, error)
func (s *Scraper) SetCookies(urlStr string, cookies []*http.Cookie) error
func (s *Scraper) GetCookieString(urlStr string) string
```

---

#### SbSd Methods

For SbSd challenge mode.

```go
func (s *Scraper) GenerateSbSdChallenge(script string, bmSo string) (string, error)
func (s *Scraper) PostSbSdChallenge(data string) error
```

---

### Config Structure

```go
type Config struct {
    // Required
    Domain          string  // Target website domain
    SensorUrl       string  // Akamai script path
    AkamaiProvider  string  // Provider: "jevi", "n4s", "roolink"

    // Optional
    SensorPostLimit     int     // Max retry attempts (default: varies)
    Language            string  // Accept-Language header
    LowSecurity         bool    // Relaxed cookie validation
    UseScript           bool    // Send script to provider
    ForceUpdateDynamics bool    // Bypass cache
    EncodedData         string  // Pre-cached dynamic data
    SbSdProvider        string  // SbSd-specific provider
    SbSd                bool    // Enable SbSd mode
    UserAgent           string  // Custom User-Agent
    SecChUa             string  // Custom sec-ch-ua header
    ProfileType         string  // TLS profile type
    GenerateReport      bool    // Enable request logging
}
```

---

## External Provider APIs

### Jevi (jevi.dev)

#### Sensor Generation

**Endpoint**: `POST https://new.jevi.dev/Solver/solve`

**Headers**:
```
Content-Type: application/json
Content-Encoding: gzip
User-Agent: curiousT
x-key: curiousT-a23f417f-096e-4258-adea-7ea874a57e56
```

**Request Body** (mode 1 - sensor):
```json
{
    "mode": 1,
    "akamaiRequest": {
        "site": "example.com",
        "abck": "_abck_cookie_value",
        "bmsz": "bm_sz_cookie_value",
        "userAgent": "Mozilla/5.0...",
        "language": "en-US",
        "script": "base64_script_or_empty",
        "encodedData": "cached_dynamic_or_empty",
        "payloadCounter": 0
    }
}
```

**Response**:
```json
{
    "sensor_data": "sensor_payload_string"
}
```

**Notes**:
- Request body must be gzip compressed
- `encodedData` returned in `EncodedData` response header
- Cache `encodedData` for subsequent requests

#### SbSd Generation

**Endpoint**: `POST https://new.jevi.dev/Solver/solve`

**Request Body** (mode 3 - SbSd):
```json
{
    "mode": 3,
    "SbsdRequest": {
        "NewVersion": true,
        "ScriptHash": "",
        "Script": "base64_script",
        "Site": "https://example.com/",
        "sbsd_o": "bm_so_cookie_value",
        "userAgent": "Mozilla/5.0...",
        "uuid": "uuid_from_script_url"
    }
}
```

**Response**:
```json
{
    "body": "sbsd_challenge_payload"
}
```

---

### N4S (n4s.xyz)

#### Dynamic Data Generation

**Endpoint**: `POST https://n4s.xyz/v3_values`

**Headers**:
```
Content-Type: application/json
X-API-KEY: 4DD7-F8F7-A935-972F-45B4-1A04
```

**Request Body**:
```json
{
    "script": "base64_encoded_script"
}
```

**Response**:
```json
{
    "data": {
        "key1": "value1",
        "key2": "value2"
    }
}
```

#### Sensor Generation

**Endpoint**: `POST https://n4s.xyz/sensor`

**Headers**:
```
Content-Type: application/json
X-API-KEY: 4DD7-F8F7-A935-972F-45B4-1A04
```

**Request Body**:
```json
{
    "targetURL": "https://example.com",
    "abck": "_abck_cookie_value",
    "bm_sz": "bm_sz_cookie_value",
    "user_agent": "Mozilla/5.0...",
    "dynamic": { /* dynamic data from v3_values */ },
    "first_sensor": true,
    "req_number": 0
}
```

**Response**:
```json
{
    "sensor_data": "sensor_payload_string"
}
```

#### SbSd Generation

**Endpoint**: `POST https://n4s.xyz/sbsd`

**Request Body**:
```json
{
    "user_agent": "Mozilla/5.0...",
    "targetURL": "https://example.com",
    "v_url": "https://example.com/script?v=xxx",
    "bm_so": "bm_so_cookie_value",
    "language": "en-US",
    "script": "base64_encoded_script"
}
```

**Response**:
```json
{
    "body": "sbsd_challenge_payload"
}
```

---

### Roolink (roolink.io)

#### Script Parsing

**Endpoint**: `POST https://www.roolink.io/api/v1/parse`

**Headers**:
```
Content-Type: text/plain
x-api-key: 2710d9bf-26fd-4add-8172-805ba613d66b
```

**Request Body**: Raw JavaScript (decoded from base64)

**Response**:
```json
{
    "ver": "version_string",
    "key": 12345,
    "dvc": "device_string",
    "din": [1, 2, 3]
}
```

#### Sensor Generation

**Endpoint**: `POST https://www.roolink.io/api/v1/sensor`

**Headers**:
```
Content-Type: application/json
x-api-key: 2710d9bf-26fd-4add-8172-805ba613d66b
```

**Request Body**:
```json
{
    "url": "https://example.com",
    "userAgent": "Mozilla/5.0...",
    "language": "en-US",
    "_abck": "_abck_cookie_value",
    "bm_sz": "bm_sz_cookie_value",
    "scriptUrl": "https://example.com/script",
    "scriptData": { /* from parse endpoint */ },
    "index": 0,
    "stepper": true
}
```

**Response**:
```json
{
    "sensor": "sensor_payload_string"
}
```

#### SbSd Generation

**Endpoint**: `POST https://www.roolink.io/api/v1/sbsd`

**Request Body**:
```json
{
    "userAgent": "Mozilla/5.0...",
    "language": "en-US",
    "vid": "uuid_from_script_url",
    "bm_o": "bm_o_cookie_value",
    "url": "https://example.com",
    "static": false
}
```

**Response**:
```json
{
    "body": "sbsd_challenge_payload"
}
```

---

## Akamai Endpoints

### Sensor Submission

**Endpoint**: `POST https://{domain}{sensorUrl}`

**Headers**:
```
Content-Type: text/plain;charset=UTF-8
Origin: https://{domain}
Referer: https://{domain}/
sec-ch-ua: "Google Chrome";v="143"...
sec-ch-ua-mobile: ?0
sec-ch-ua-platform: "Windows"
sec-fetch-dest: empty
sec-fetch-mode: cors
sec-fetch-site: same-origin
User-Agent: Mozilla/5.0...
```

**Request Body**:
```json
{
    "sensor_data": "sensor_payload_from_provider"
}
```

**Success Response**:
- Status: 200
- Body: Short string without newlines
- Cookie: `_abck` containing `~0~`

**Failure Response**:
- Body contains newlines, OR
- `_abck` cookie does not contain `~0~`

### SbSd Submission

**Endpoint**: `POST https://{domain}{sensorUrl}`

**Request Body**:
```json
{
    "body": "sbsd_payload_from_provider"
}
```

**Success Response**:
- Status: 200 or 202
- Content-Length: 0

---

## Error Codes

### Library Errors

| Error | Description |
|-------|-------------|
| `config is nil` | Missing configuration |
| `missing _abck cookie` | Cookie not received from site |
| `missing bm_sz cookie` | Cookie not received from site |
| `homepage blocked` | Site rejected initial request |
| `invalid akamaiProvider` | Unknown provider specified |

### Provider Errors

| Provider | Error Pattern |
|----------|--------------|
| Jevi | Non-200 status code |
| N4S | `{"error": "message"}` in response |
| Roolink | `{"error": "message"}` in response |

---

## Rate Limits

No rate limiting implemented in the library. Limits are enforced by:
- Provider APIs (varies by provider and subscription)
- Target websites (varies by site)

---

*Last Updated: 2026-01-27*
