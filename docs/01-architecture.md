# Architecture Overview

## System Architecture

Gerador Cookies follows a layered architecture with clear separation of concerns between HTTP handling, challenge solving, and caching.

```
┌─────────────────────────────────────────────────────────────────┐
│                         Consumer Layer                          │
│                    (External Applications)                      │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Scraper Interface                        │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    scraper.Scraper                        │  │
│  │  - NewScraper()     - GenerateSession()                  │  │
│  │  - GetCookies()     - GetAntiBotScript()                 │  │
│  │  - SetCookies()     - GetAntiBotScriptURL()              │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                               │
          ┌────────────────────┼────────────────────┐
          ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   HTTP Layer    │  │  Solver Layer   │  │   Cache Layer   │
│ ┌─────────────┐ │  │ ┌─────────────┐ │  │ ┌─────────────┐ │
│ │ TLS Client  │ │  │ │AkamaiSolver │ │  │ │ProviderCache│ │
│ │ (bogdanfinn)│ │  │ │             │ │  │ │             │ │
│ └─────────────┘ │  │ └─────────────┘ │  │ └─────────────┘ │
│ ┌─────────────┐ │  └────────┬────────┘  └─────────────────┘
│ │Simple Client│ │           │
│ │ (net/http)  │ │           ▼
│ └─────────────┘ │  ┌─────────────────────────────────────┐
└─────────────────┘  │         Provider APIs               │
                     │  ┌─────────┬─────────┬─────────┐   │
                     │  │  Jevi   │   N4S   │ Roolink │   │
                     │  └─────────┴─────────┴─────────┘   │
                     └─────────────────────────────────────┘
```

---

## Core Components

### 1. Scraper (Main Entry Point)

The `Scraper` struct serves as the primary interface for consumers.

```go
type Scraper struct {
    client         tls_client.HttpClient    // TLS fingerprinting client
    simpleClient   *http.Client             // Standard HTTP client
    userAgent      UserAgent                // Browser identification
    solver         *AkamaiSolver            // Challenge solver
    config         *Config                  // Configuration
    providerCache  *ProviderCache           // Cache manager
    report         *requestReport           // Debug report writer
}
```

**Responsibilities:**
- HTTP request execution with TLS fingerprinting
- Cookie management
- Script extraction from target websites
- Coordination of challenge solving

### 2. AkamaiSolver (Challenge Logic)

Handles the core Akamai challenge-solving logic.

```go
type AkamaiSolver struct {
    scraper      *Scraper
    apiType      string         // "localhost" or "hwk"
    apiKey       string         // API key for solver services
    requestCount int32          // Request counter
}
```

**Responsibilities:**
- Provider API communication
- Sensor data generation
- Challenge submission to Akamai
- Response validation

### 3. ProviderCache (Persistence)

Manages caching of provider data to improve performance.

```go
type providerCacheEntry struct {
    ScriptURL  string
    Dynamic    string
    ExpiresAt  time.Time
    UpdatedAt  time.Time
    Domain     string
    Provider   string
    Mode       string
}
```

**Responsibilities:**
- Caching script URLs and dynamic data
- 24-hour TTL management
- Thread-safe access

---

## Request Flow

### Standard Sensor Generation Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│                        Target Website (Akamai Protected)              │
└──────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    │                               │
        1. GET Homepage                 2. Extract Script URL
        (seed cookies)                  (from HTML body)
                    │                               │
        ┌───────────┴────────────┬─────────────────┤
        │                        │                  │
    ┌───▼────────────────────────▼──────────────────▼──┐
    │           Scraper (with TLS Fingerprinting)      │
    │  • Proxy Support                                 │
    │  • Cookie Management                             │
    │  • Custom User-Agent/Headers                     │
    │  • HTTP/2 with custom settings                   │
    └───┬──────────────────────────────────────────────┘
        │
        │          3. Fetch Anti-Bot Script
        │          (base64 encoded)
        │
        ├────────────────────────┐
        │                        │
    ┌───▼────────────────────────▼───────────────────┐
    │      AkamaiSolver (Provider Logic)             │
    │  ┌─────────────────────────────────────────┐   │
    │  │ Provider Options:                       │   │
    │  │ • Jevi (jevi.dev)                      │   │
    │  │ • N4S  (n4s.xyz)                       │   │
    │  │ • Roolink (roolink.io)                 │   │
    │  └─────────────────────────────────────────┘   │
    │                                                │
    │  4. Call Provider API:                         │
    │  • Send: script, cookies, user-agent           │
    │  • Get: sensor_data                            │
    │                                                │
    │  5. Cache: Dynamic/Hash Data                   │
    │     (for reuse, 24-hour TTL)                   │
    └───────────┬────────────────────────────────────┘
                │
                │  6. Generate Sensor
                │
        ┌───────▼──────────────┐
        │  Sensor Payload      │
        │  (validated JSON)    │
        │  + Cookies           │
        └───────┬──────────────┘
                │
        ┌───────▼──────────────────────────────────┐
        │  7. POST to Akamai Sensor Endpoint       │
        │     (Content-Type: text/plain)           │
        │     (with all previous cookies)          │
        └───────┬──────────────────────────────────┘
                │
        ┌───────▼──────────────────────────────────┐
        │     Validation Check:                    │
        │  • Response contains no newlines?  ✓     │
        │  • _abck cookie contains "~0~"?    ✓     │
        │  • _abck length == 541 (low-sec)?  ✓     │
        └───────┬──────────────────────────────────┘
                │
    Success: ~0~ Token          Failure: Retry up to
    in _abck cookie             SensorPostLimit times
```

---

## TLS Fingerprinting Architecture

### Browser Profile System

The library supports multiple browser profiles for TLS fingerprinting:

```
┌─────────────────────────────────────────────────────────────┐
│                    Browser Profile Factory                   │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │   Chrome 142    │  │   Chrome 143    │  (Default)        │
│  │  ┌───────────┐  │  │  ┌───────────┐  │                   │
│  │  │HTTP/2 Cfg │  │  │  │HTTP/2 Cfg │  │                   │
│  │  │TLS Spec   │  │  │  │TLS Spec   │  │                   │
│  │  │Headers    │  │  │  │Headers    │  │                   │
│  │  └───────────┘  │  │  └───────────┘  │                   │
│  └─────────────────┘  └─────────────────┘                   │
│                                                              │
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │  Safari iOS 18  │  │  Firefox 135    │                   │
│  │  ┌───────────┐  │  │  ┌───────────┐  │                   │
│  │  │HTTP/2 Cfg │  │  │  │HTTP/2 Cfg │  │                   │
│  │  │TLS Spec   │  │  │  │TLS Spec   │  │                   │
│  │  │Headers    │  │  │  │Headers    │  │                   │
│  │  └───────────┘  │  │  └───────────┘  │                   │
│  └─────────────────┘  └─────────────────┘                   │
└─────────────────────────────────────────────────────────────┘
```

### HTTP/2 Settings by Profile

| Profile | Header Table Size | Initial Window Size | Max Concurrent Streams |
|---------|-------------------|---------------------|------------------------|
| Chrome 142/143 | 65,536 | 6,291,456 | 1,000 |
| Safari iOS | 4,096 | 2,097,152 | 100 |
| Firefox 135 | 65,536 | 12,517,377 | 200 |

---

## Provider Integration Architecture

### Provider Selection Flow

```
┌─────────────────────────────────────────────────────────────┐
│                      Config.AkamaiProvider                   │
├─────────────────────────────────────────────────────────────┤
│                              │                               │
│           ┌──────────────────┼──────────────────┐           │
│           ▼                  ▼                  ▼           │
│    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐   │
│    │    Jevi     │    │     N4S     │    │   Roolink   │   │
│    │  jevi.dev   │    │   n4s.xyz   │    │ roolink.io  │   │
│    │             │    │             │    │             │   │
│    │ POST /solve │    │ POST /sensor│    │POST /sensor │   │
│    │             │    │ GET /values │    │             │   │
│    └─────────────┘    └─────────────┘    └─────────────┘   │
│                              │                               │
│                              ▼                               │
│                    ┌─────────────────┐                      │
│                    │  Sensor Data    │                      │
│                    │  Generation     │                      │
│                    └─────────────────┘                      │
└─────────────────────────────────────────────────────────────┘
```

### Provider-Specific Endpoints

| Provider | Sensor Endpoint | Dynamic Endpoint | SbSd Support |
|----------|-----------------|------------------|--------------|
| Jevi | `POST /v3/solve` | N/A (inline) | Yes |
| N4S | `POST /sensor` | `GET /v3_values` | Yes |
| Roolink | `POST /akamai/sensor` | N/A (inline) | Yes |

---

## Caching Architecture

### Cache Key Structure

```
{domain}|{provider}|{mode}

Examples:
- "example.com|jevi|sensor"
- "shop.com|n4s|sbsd"
```

### Cache Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    ProviderCache                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │                    Cache Lookup                      │   │
│   │                         │                            │   │
│   │    ┌────────────────────┴────────────────────┐      │   │
│   │    │                                         │      │   │
│   │    ▼                                         ▼      │   │
│   │  Hit?                                    Expired?   │   │
│   │    │                                         │      │   │
│   │   Yes                                       Yes     │   │
│   │    │                                         │      │   │
│   │    ▼                                         ▼      │   │
│   │  Return                                  Refresh    │   │
│   │  Cached                                  from API   │   │
│   │                                              │      │   │
│   │                                              ▼      │   │
│   │                                          Update     │   │
│   │                                          Cache      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Storage: ~/.cache/reqs/provider-cache.json                │
│   TTL: 24 hours                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Error Handling Strategy

### Retry Mechanism

```
┌─────────────────────────────────────────────────────────────┐
│                    Sensor Submission Loop                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   for i := 0; i < SensorPostLimit; i++ {                    │
│       sensor := solver.GenerateSensor()                      │
│       response := sendAntiBotRequest(sensor)                │
│                                                              │
│       if validateResponse(response) {                        │
│           return SUCCESS                                     │
│       }                                                      │
│                                                              │
│       // Retry with new sensor                               │
│   }                                                          │
│   return FAILURE                                             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Validation Criteria

| Criterion | Description |
|-----------|-------------|
| No Newlines | Response body must not contain newline characters |
| `~0~` Token | `_abck` cookie must contain the `~0~` success marker |
| Cookie Length | For `LowSecurity` mode, `_abck` cookie length must equal 541 |

---

## Thread Safety

### Concurrent Access Patterns

```go
// ProviderCache uses mutex for thread-safe access
type ProviderCache struct {
    entries map[string]*providerCacheEntry
    mu      sync.RWMutex  // Read-write mutex
}

// Safe read pattern
func (pc *ProviderCache) Get(key string) *providerCacheEntry {
    pc.mu.RLock()
    defer pc.mu.RUnlock()
    return pc.entries[key]
}

// Safe write pattern
func (pc *ProviderCache) Set(key string, entry *providerCacheEntry) {
    pc.mu.Lock()
    defer pc.mu.Unlock()
    pc.entries[key] = entry
}
```

---

## Related Documentation

- [Technical Context](00-technical-context.md)
- [Component Documentation](02-components.md)
- [API and Data Flow](03-api-data-flow.md)
- [Configuration and Security](04-configuration-security.md)
