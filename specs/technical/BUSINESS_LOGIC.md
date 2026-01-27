# Business Logic Documentation

## Domain Concepts

### Core Domain: Akamai Anti-Bot Bypass for Security Testing

The library operates in the domain of authorized security testing against Akamai-protected websites. Understanding the following concepts is essential for working with the codebase.

---

## Key Entities

### 1. Scraper

The main orchestrator that coordinates all operations.

**Responsibilities**:
- HTTP request execution with TLS fingerprinting
- Cookie management (storage, retrieval, injection)
- Script extraction from target websites
- Coordination of challenge solving
- Debug reporting

**Lifecycle**:
```
NewScraper() → Configure → GetScriptURL → GetScript → GenerateSession → GetCookies
```

### 2. AkamaiSolver

Handles the challenge-solving logic with external providers.

**Responsibilities**:
- Provider API communication
- Sensor data generation
- Challenge submission to Akamai
- Response validation

**State**:
```go
type AkamaiSolver struct {
    scraper      *Scraper    // Parent scraper reference
    apiType      string      // API type (legacy)
    apiKey       string      // Provider authentication
    requestCount int32       // Request counter
}
```

### 3. Provider Cache

Manages caching of provider-specific data.

**Cached Data**:
- Script URLs (per domain)
- Dynamic data (per domain/provider/mode)
- TTL metadata

---

## Business Rules

### Rule 1: Authorization Requirement

**Rule**: All testing must be authorized by site owners.

**Enforcement**: Documentation and usage guidelines (no code enforcement).

### Rule 2: Cookie Validation

**Rule**: A generated cookie is valid when:

```go
// Standard validation
isValid := !strings.Contains(responseBody, "\n") &&
           strings.Contains(abckCookie.Value, "~0~")

// Low security mode
isValid := len(abckCookie.Value) == 541
```

**Context**: The `~0~` marker in the `_abck` cookie indicates successful challenge completion. The 541-character length is specific to certain low-security configurations.

### Rule 3: Retry Logic

**Rule**: Challenge attempts retry up to `SensorPostLimit` times.

```go
for i := 0; i < config.SensorPostLimit; i++ {
    success, err := solver.solveSingle(script, i)
    if success {
        return true, nil
    }
}
return false, nil
```

**Rationale**: First attempt may fail due to timing, randomness, or Akamai's probabilistic acceptance.

### Rule 4: Cache TTL

**Rule**: Cached data expires after 24 hours.

```go
cur.ExpiresAt = time.Now().Add(24 * time.Hour)
```

**Rationale**: Balance between API efficiency and data freshness.

### Rule 5: Provider Selection

**Rule**: Provider is selected via configuration, not automatic failover.

```go
switch config.AkamaiProvider {
case "n4s":    // Use N4S
case "jevi":   // Use Jevi
case "roolink": // Use Roolink
}
```

**Rationale**: Different providers may have different success rates per site; manual selection allows optimization.

---

## Workflows

### Workflow 1: Standard Cookie Generation

```
┌─────────────────────────────────────────────────────────────┐
│                   Standard Cookie Generation                 │
└─────────────────────────────────────────────────────────────┘

1. INITIALIZE
   │
   ├── Create Scraper with config and TLS profile
   ├── Load provider cache
   └── Initialize solver
   │
2. EXTRACT SCRIPT URL
   │
   ├── Check cache for script URL
   │   ├── Cache hit → Use cached URL
   │   └── Cache miss → Continue
   │
   ├── GET target homepage
   │   └── Parse HTML for <script> tags
   │
   ├── Find Akamai script (no extension, specific pattern)
   │   ├── SbSd mode: look for ?v= parameter
   │   └── Standard mode: look for non-deferred scripts
   │
   └── Cache script URL
   │
3. DOWNLOAD SCRIPT
   │
   ├── Check cache for dynamic data
   │   ├── Cache hit → Seed cookies only (minimal request)
   │   └── Cache miss → Full download
   │
   ├── Request script (with cookies from previous step)
   ├── Decompress if needed (gzip, brotli)
   └── Base64 encode script body
   │
4. GENERATE SESSION
   │
   ├── Select provider based on config
   │
   ├── FOR each attempt (1 to SensorPostLimit):
   │   │
   │   ├── Get current cookies (_abck, bm_sz)
   │   │
   │   ├── Call provider API
   │   │   ├── Send: site, cookies, user-agent, script/dynamic
   │   │   └── Receive: sensor_data
   │   │
   │   ├── POST sensor to Akamai endpoint
   │   │   ├── URL: https://{domain}{sensorUrl}
   │   │   ├── Body: {"sensor_data": "..."}
   │   │   └── Headers: matching browser profile
   │   │
   │   ├── Validate response
   │   │   ├── Check for newlines in body
   │   │   ├── Check _abck cookie for "~0~"
   │   │   └── Update cookie jar
   │   │
   │   └── If valid → Return success
   │
   └── Return failure (all attempts exhausted)
   │
5. RETRIEVE COOKIES
   │
   └── GetCookies() returns _abck, bm_sz, etc.
```

### Workflow 2: SbSd Challenge Mode

```
┌─────────────────────────────────────────────────────────────┐
│                    SbSd Challenge Mode                       │
└─────────────────────────────────────────────────────────────┘

1. INITIALIZE (same as standard)
   │
2. EXTRACT SCRIPT URL
   │
   ├── Look for scripts with ?v= parameter
   └── Different parsing logic for SbSd scripts
   │
3. DOWNLOAD SCRIPT
   │
   ├── SbSd script fetched without proxy (simpleClient)
   └── Full download required (no cache shortcut)
   │
4. GENERATE SBSD CHALLENGE
   │
   ├── Extract bm_so cookie value
   │
   ├── Call provider SbSd API
   │   ├── Jevi: mode 3 with uuid from script URL
   │   ├── N4S: /sbsd endpoint with script and bm_so
   │   └── Roolink: /api/v1/sbsd with vid and bm_o
   │
   └── Receive challenge body
   │
5. POST SBSD CHALLENGE
   │
   ├── POST to Akamai endpoint
   │   ├── Body: {"body": "challenge_data"}
   │   └── Expect: 200/202 status, empty body
   │
   └── Update cookies from response
```

---

## State Transitions

### Cookie State Machine

```
┌─────────────┐
│   Initial   │
│  (no _abck) │
└──────┬──────┘
       │ GET homepage/script
       ▼
┌─────────────┐
│  Seed _abck │
│ (invalid)   │
└──────┬──────┘
       │ POST sensor
       ▼
┌─────────────────────────────────────┐
│            Validation               │
│  ┌─────────────┐  ┌─────────────┐  │
│  │   Success   │  │   Failure   │  │
│  │ (~0~ token) │  │ (retry)     │  │
│  └──────┬──────┘  └──────┬──────┘  │
└─────────┼────────────────┼─────────┘
          │                │
          ▼                ▼
┌─────────────┐    ┌─────────────┐
│ Valid _abck │    │ Exhausted   │
│ (usable)    │    │ (failed)    │
└─────────────┘    └─────────────┘
```

---

## Edge Cases

### 1. Homepage Blocked

**Scenario**: Target site blocks initial homepage request.

**Detection**:
```go
if response.StatusCode < 200 || response.StatusCode > 299 {
    return "", fmt.Errorf("homepage blocked: status=%s", response.Status)
}
```

**Resolution**: Use different proxy, different TLS profile, or verify authorization.

### 2. Script Not Found

**Scenario**: Akamai script pattern not found in HTML.

**Detection**: Empty `akamaiUrl` after parsing.

**Resolution**: Site may not use Akamai, or script embedded differently.

### 3. Provider API Failure

**Scenario**: Provider returns error or invalid response.

**Detection**:
```go
if errMsg, hasError := result["error"].(string); hasError {
    return "", fmt.Errorf("provider error: %s", errMsg)
}
```

**Resolution**: Try different provider, check API key validity.

### 4. Cache Stale Data

**Scenario**: Cached dynamic data no longer valid.

**Detection**: Repeated challenge failures despite valid provider response.

**Resolution**:
```go
config.ForceUpdateDynamics = true  // Bypass cache
```

### 5. Profile Mismatch

**Scenario**: User-Agent doesn't match TLS profile.

**Detection**: Akamai rejects sensor despite valid generation.

**Resolution**: Ensure `ProfileType` matches `UserAgent` in config.

---

## Validation Rules

### Input Validation

| Field | Rule |
|-------|------|
| `Domain` | Non-empty, valid hostname |
| `SensorUrl` | Starts with `/`, valid path |
| `SensorPostLimit` | Positive integer |
| `AkamaiProvider` | One of: `jevi`, `n4s`, `roolink` |

### Output Validation

| Cookie | Valid When |
|--------|-----------|
| `_abck` | Contains `~0~` OR length == 541 (low security) |
| `bm_sz` | Present after homepage request |

---

## Related Documentation

- [API Specification](API_SPECIFICATION.md) - Provider API details
- [Troubleshooting Guide](TROUBLESHOOTING.md) - Error resolution
- [ADR-001](adr/ADR-001-multi-provider-strategy.md) - Provider decision

---

*Last Updated: 2026-01-27*
