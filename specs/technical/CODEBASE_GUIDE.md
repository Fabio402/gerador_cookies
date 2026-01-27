# Codebase Navigation Guide

## Directory Structure

```
gerador_cookies/
├── go.mod                          # Go module definition
├── go.sum                          # Dependency checksums
│
├── akt/                            # Utility package
│   └── logger.go                   # Debug logging utilities (148 bytes)
│
├── scraper/                        # Core package (~2,927 LoC)
│   ├── scraper.go                  # Main entry point (1,114 LoC)
│   ├── akamaiSolver.go             # Challenge solver (1,139 LoC)
│   ├── provider_cache.go           # Cache management (137 LoC)
│   ├── chrome142_simple.go         # Chrome TLS profile (120 LoC)
│   ├── ios_profiles.go             # iOS TLS profiles (412 LoC)
│   └── utils.go                    # Utility functions (5 LoC)
│
├── docs/                           # Legacy documentation
│   ├── 00-technical-context.md
│   └── 01-architecture.md
│
└── specs/                          # Technical specifications
    └── technical/                  # Current documentation
        ├── index.md
        ├── project_charter.md
        ├── CLAUDE.meta.md
        ├── CODEBASE_GUIDE.md       # (this file)
        ├── BUSINESS_LOGIC.md
        ├── API_SPECIFICATION.md
        ├── CONTRIBUTING.md
        ├── TROUBLESHOOTING.md
        ├── ARCHITECTURE_CHALLENGES.md
        └── adr/
            ├── ADR-001-multi-provider-strategy.md
            ├── ADR-002-tls-fingerprinting.md
            └── ADR-003-cache-strategy.md
```

---

## Key Files Deep Dive

### scraper/scraper.go

**Purpose**: Main scraper implementation, HTTP client management, cookie handling.

**Key Structures**:

```go
type Config struct {
    Domain          string  // Target website domain
    SensorUrl       string  // Akamai script path
    SensorPostLimit int     // Max retry attempts
    Language        string  // Accept-Language header
    LowSecurity     bool    // Relaxed validation mode
    UseScript       bool    // Send script to provider
    ForceUpdateDynamics bool // Bypass cache
    EncodedData     string  // Cached dynamic data
    AkamaiProvider  string  // Provider selection
    SbSdProvider    string  // SbSd-specific provider
    SbSd            bool    // SbSd challenge mode
    UserAgent       string  // Custom User-Agent
    SecChUa         string  // Custom sec-ch-ua header
    ProfileType     string  // TLS profile type
    GenerateReport  bool    // Enable request logging
}

type Scraper struct {
    client         tls_client.HttpClient  // TLS fingerprinting client
    simpleClient   *http.Client           // Standard HTTP client
    userAgent      UserAgent              // Browser identification
    solver         *AkamaiSolver          // Challenge solver
    config         *Config                // Configuration
    providerCache  *ProviderCache         // Cache manager
    report         *requestReport         // Debug report writer
}
```

**Key Methods**:

| Method | Line | Purpose |
|--------|------|---------|
| `NewScraper()` | 303 | Constructor, initializes clients and cache |
| `GetHomepage()` | 396 | Fetches target homepage |
| `GetAntiBotScriptURL()` | 545 | Extracts Akamai script URL from HTML |
| `GetAntiBotScript()` | 426 | Downloads and base64 encodes script |
| `GenerateSession()` | 721 | Orchestrates challenge solving |
| `GetCookies()` | 1043 | Retrieves cookies for URL |
| `SetCookies()` | 1065 | Sets cookies for URL |
| `setHeaders()` | 962 | Configures request headers by profile |

---

### scraper/akamaiSolver.go

**Purpose**: Provider integration, sensor generation, challenge submission.

**Key Structures**:

```go
type AkamaiSolver struct {
    scraper      *Scraper
    apiType      string   // "localhost" (legacy)
    apiKey       string   // Provider API key
    requestCount int32    // Request counter
}

// Provider-specific request structures
type LocalhostRequest struct { ... }      // Jevi
type LocalhostRequestN4S struct { ... }   // N4S
type roolinkScriptData struct { ... }     // Roolink
```

**Key Methods**:

| Method | Line | Purpose |
|--------|------|---------|
| `Solve()` | 113 | Main solver dispatcher by provider |
| `solveSingle()` | 486 | Jevi single attempt |
| `solveSingleN4S()` | 437 | N4S single attempt |
| `solveSingleRoolink()` | 218 | Roolink single attempt |
| `generateDynamic()` | 535 | Generates N4S dynamic data |
| `GenerateSbSd()` | 695 | SbSd challenge generator |
| `sendAntiBotRequest()` | 1079 | Submits sensor to Akamai |

**Provider Endpoints**:

| Provider | Sensor | Dynamic | SbSd |
|----------|--------|---------|------|
| Jevi | `new.jevi.dev/Solver/solve` | Inline | `new.jevi.dev/Solver/solve` (mode 3) |
| N4S | `n4s.xyz/sensor` | `n4s.xyz/v3_values` | `n4s.xyz/sbsd` |
| Roolink | `roolink.io/api/v1/sensor` | `roolink.io/api/v1/parse` | `roolink.io/api/v1/sbsd` |

---

### scraper/provider_cache.go

**Purpose**: Caching provider data to reduce API calls.

**Key Structure**:

```go
type ProviderCache struct {
    mu      sync.Mutex                    // Thread safety
    path    string                        // Cache file path
    entries map[string]providerCacheEntry // Cache data
}

type providerCacheEntry struct {
    ScriptURL  string    // Cached script URL
    Dynamic    string    // Cached dynamic data
    ExpiresAt  time.Time // TTL expiration
    UpdatedAt  time.Time // Last update
    Domain     string    // Target domain
    Provider   string    // Provider name
    Mode       string    // sensor or sbsd
}
```

**Key Methods**:

| Method | Line | Purpose |
|--------|------|---------|
| `LoadProviderCacheDefault()` | 64 | Loads cache with env var controls |
| `Get()` | 103 | Retrieves cached entry if valid |
| `Upsert()` | 119 | Updates or inserts cache entry |

**Cache Location**: `~/.cache/reqs/provider-cache.json`

---

### scraper/chrome142_simple.go

**Purpose**: Chrome 142+ TLS fingerprint profile.

**Key Elements**:

```go
var Chrome142Simple = profiles.NewClientProfile(
    HelloChrome_142,
    // HTTP/2 SETTINGS
    map[http2.SettingID]uint32{
        http2.SettingHeaderTableSize:   65536,
        http2.SettingEnablePush:        0,
        http2.SettingInitialWindowSize: 6291456,
        http2.SettingMaxHeaderListSize: 262144,
    },
    // Pseudo header order
    []string{":method", ":authority", ":scheme", ":path"},
    // Connection flow
    15663105,
    // Priority frames
    []http2.Priority{...},
)
```

---

### scraper/ios_profiles.go

**Purpose**: Safari iOS TLS fingerprint profiles.

**Profiles Available**:

| Profile | Description |
|---------|-------------|
| `StandardIOS` | Generic iOS profile |
| `SecondaryIOS` | Alternative iOS profile |
| `SecondaryIOS26` | iOS with unknown setting ID 9 |
| `StandardIOS18` | iOS 18 specific profile |

---

## Data Flow

### Standard Cookie Generation Flow

```
1. NewScraper(proxyURL, config, profile)
   │
2. GetAntiBotScriptURL(providedUrl)
   │
   ├── GET homepage
   ├── Parse HTML for script tags
   └── Return script URL (cached if available)
   │
3. GetAntiBotScript()
   │
   ├── Check cache for dynamic data
   ├── If cached: seed cookies only
   └── If not cached: full script download
   │
4. GenerateSession(script)
   │
   ├── Select provider (config.AkamaiProvider)
   ├── Loop up to SensorPostLimit times:
   │   ├── Call provider API for sensor
   │   ├── POST sensor to Akamai endpoint
   │   └── Validate response cookies
   └── Return success/failure
   │
5. GetCookies(url)
   │
   └── Return _abck, bm_sz, etc.
```

### Cookie Management

```
┌─────────────────────────────────────────────────────────┐
│                    Scraper Instance                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│   tls_client.HttpClient                                 │
│   └── CookieJar (automatic)                             │
│       ├── Stores cookies from responses                 │
│       ├── Sends cookies in requests                     │
│       └── Domain-scoped isolation                       │
│                                                          │
│   Methods:                                               │
│   ├── GetCookies(url) → []*http.Cookie                 │
│   ├── SetCookies(url, cookies)                         │
│   └── GetCookieString(url) → "name=value; ..."         │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## Configuration Matrix

### Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `isDebug` | Enable debug logging | `false` |
| `DEBUG_PROXY` | Debug proxy URL | none |
| `REQS_PROVIDER_CACHE_ENABLE` | Enable cache | `0` |
| `REQS_PROVIDER_CACHE_DISABLE` | Disable cache | `0` |
| `REQS_PROVIDER_CACHE_CLEAR_ON_START` | Clear cache on start | `0` |

### Profile Type to Headers

| ProfileType | User-Agent Pattern |
|-------------|-------------------|
| `safari_ios_18_5` | `Mozilla/5.0 (iPhone; CPU iPhone OS 18_5...)` |
| `firefox_135` | `Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:145.0...)` |
| default | `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36...Chrome/143...` |

---

## Integration Points

### External Dependencies

| Service | Purpose | Required |
|---------|---------|----------|
| Target Website | Cookie generation target | Yes |
| Jevi API | Sensor generation | One of three |
| N4S API | Sensor generation | One of three |
| Roolink API | Sensor generation | One of three |
| Proxy Server | IP rotation | Optional |

### Future Integration

| Service | Purpose | Status |
|---------|---------|--------|
| TLS-API | HTTP with fingerprinting | Planned |

---

## Quick Reference

### Find Provider Logic
`scraper/akamaiSolver.go:113` - `Solve()` method

### Find Cookie Handling
`scraper/scraper.go:1043-1072` - Cookie methods

### Find TLS Profiles
`scraper/chrome142_simple.go` and `scraper/ios_profiles.go`

### Find Cache Logic
`scraper/provider_cache.go:103-137` - Get/Upsert methods

### Find Header Configuration
`scraper/scraper.go:962-1037` - `setHeaders()` method

---

*Last Updated: 2026-01-27*
