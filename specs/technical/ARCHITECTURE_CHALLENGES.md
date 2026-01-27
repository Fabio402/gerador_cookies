# Architecture Challenges

## Overview

This document catalogs known architectural challenges, technical debt, and planned improvements for the Gerador Cookies library.

---

## Current Challenges

### 1. TLS Client Integration

**Challenge**: Direct dependency on `bogdanfinn/tls-client`

**Impact**: High

**Description**:
The library directly integrates with `bogdanfinn/tls-client` for TLS fingerprinting. This creates:
- Duplicated fingerprint code if used in multiple projects
- No centralized profile management
- Updates require library changes

**Current State**:
```go
// scraper/scraper.go
import tls_client "github.com/bogdanfinn/tls-client"

client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
```

**Planned Solution**:
Migrate to TLS-API service:
- Centralized fingerprint management
- Single point of update for browser profiles
- Cookie management remains in Gerador Cookies

**Migration Approach**:
```go
// Future: Replace tls_client calls with TLS-API HTTP calls
// TLS-API handles: TLS fingerprinting, HTTP requests
// Gerador Cookies handles: Cookies, session state, provider orchestration
```

**Priority**: High

---

### 2. Hardcoded API Keys

**Challenge**: Provider API keys embedded in source code

**Impact**: Medium

**Description**:
API keys for all providers are hardcoded:
```go
// akamaiSolver.go
apiKey := "curiousT-a23f417f-096e-4258-adea-7ea874a57e56"  // Jevi
req.Header.Set("X-API-KEY", "4DD7-F8F7-A935-972F-45B4-1A04")  // N4S
return "2710d9bf-26fd-4add-8172-805ba613d66b"  // Roolink
```

**Risks**:
- Keys visible in version control
- No key rotation without code changes
- Difficult to use different keys per environment

**Planned Solution**:
```go
// Future: Environment variable or config-based keys
apiKey := os.Getenv("JEVI_API_KEY")
if apiKey == "" {
    apiKey = config.JeviApiKey  // Fallback to config
}
```

**Priority**: Medium

---

### 3. No Automated Testing

**Challenge**: Complete absence of automated tests

**Impact**: Medium

**Description**:
No unit tests, integration tests, or CI/CD pipeline exists. Testing is entirely manual.

**Risks**:
- Regressions undetected
- Difficult to refactor safely
- No confidence in changes

**Planned Solution**:
1. Add unit tests for core logic:
   - Cache operations
   - Cookie parsing
   - Response validation

2. Add integration tests with mocked providers

3. Set up CI pipeline for automated builds

**Example Test Structure**:
```go
// scraper/scraper_test.go
func TestCookieValidation(t *testing.T) {
    tests := []struct {
        name     string
        cookie   string
        lowSec   bool
        expected bool
    }{
        {"valid ~0~", "abc~0~def", false, true},
        {"invalid", "abcdef", false, false},
        {"low sec valid", strings.Repeat("x", 541), true, true},
    }
    // ...
}
```

**Priority**: Medium

---

### 4. Sequential Processing Only

**Challenge**: No parallel challenge attempts

**Impact**: Low-Medium

**Description**:
Challenge solving is strictly sequential:
```go
for i := 0; i < config.SensorPostLimit; i++ {
    success, err := solver.solveSingle(script, i)
    if success {
        return true, nil
    }
}
```

**Limitations**:
- Cannot try multiple providers simultaneously
- No automatic failover
- Slower total execution time

**Planned Solution**:
```go
// Future: Parallel provider attempts
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

results := make(chan bool, len(providers))
for _, provider := range providers {
    go func(p string) {
        success, _ := solveWithProvider(ctx, p)
        if success {
            cancel()  // Stop others
        }
        results <- success
    }(provider)
}
```

**Priority**: Low

---

### 5. Profile Staleness

**Challenge**: Browser profiles require manual updates

**Impact**: Medium

**Description**:
TLS fingerprints must match current browser versions. When browsers update:
- Profiles become detectable
- Manual code changes required
- No automated freshness checking

**Current Profiles**:
| Profile | Browser Version | Status |
|---------|-----------------|--------|
| Chrome142Simple | Chrome 142/143 | Current |
| StandardIOS18 | Safari iOS 18 | Current |
| firefox_135 | Firefox 135 | Current |

**Planned Solution**:
1. Monitor browser release cycles
2. Automate profile generation from browser analysis
3. With TLS-API: centralized profile updates

**Priority**: Medium (ongoing maintenance)

---

### 6. Limited Error Context

**Challenge**: Errors lack detailed context

**Impact**: Low

**Description**:
Many errors provide minimal information:
```go
return false, fmt.Errorf("error getting cookies: %v", err)
```

**Improvement**:
```go
return false, fmt.Errorf("getting cookies for domain %s: %w", domain, err)
```

**Priority**: Low

---

### 7. Report File Location

**Challenge**: Debug reports hardcoded to `/tmp/`

**Impact**: Low

**Description**:
```go
name := fmt.Sprintf("/tmp/getsensor-report-%d.txt", time.Now().UnixNano())
```

**Limitations**:
- Not configurable
- May not exist on all systems
- No cleanup mechanism

**Planned Solution**:
```go
// Future: Configurable report path
reportDir := config.ReportDir
if reportDir == "" {
    reportDir = os.TempDir()
}
```

**Priority**: Low

---

## Technical Debt Inventory

| Item | Severity | Effort | Priority |
|------|----------|--------|----------|
| TLS-API migration | High | High | 1 |
| API keys to env vars | Medium | Low | 2 |
| Add unit tests | Medium | Medium | 3 |
| Profile update automation | Medium | High | 4 |
| Parallel processing | Low | Medium | 5 |
| Error context | Low | Low | 6 |
| Configurable reports | Low | Low | 7 |

---

## Improvement Roadmap

### Phase 1: Security & Stability
- [ ] Move API keys to environment variables
- [ ] Add basic unit tests for core functions
- [ ] Document all error codes

### Phase 2: TLS-API Migration
- [ ] Design integration interface
- [ ] Implement TLS-API client
- [ ] Migrate HTTP calls (keep cookie management)
- [ ] Test and validate

### Phase 3: Performance
- [ ] Implement parallel provider attempts
- [ ] Add connection pooling (via TLS-API)
- [ ] Optimize cache hit rates

### Phase 4: Maintainability
- [ ] Add CI/CD pipeline
- [ ] Automate profile updates
- [ ] Improve error messages

---

## Contribution Opportunities

For developers looking to contribute, these areas need attention:

### Easy (Good First Issues)
- Improve error messages with more context
- Add configuration for report directory
- Document undocumented functions

### Medium
- Add unit tests for `provider_cache.go`
- Add unit tests for cookie validation logic
- Implement environment variable API key loading

### Hard
- Design TLS-API integration interface
- Implement parallel provider execution
- Create automated profile update system

---

## Related Documentation

- [Project Charter](project_charter.md) - Project scope and vision
- [ADR-001](adr/ADR-001-multi-provider-strategy.md) - Provider decisions
- [ADR-002](adr/ADR-002-tls-fingerprinting.md) - TLS approach
- [Contributing](CONTRIBUTING.md) - How to contribute

---

*Last Updated: 2026-01-27*
