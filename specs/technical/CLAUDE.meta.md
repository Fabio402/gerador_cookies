# AI Development Guide - Gerador Cookies

## Project Overview for AI Assistants

This is a Go library for **authorized security testing** that generates valid session cookies for Akamai-protected websites. The library is used by authorized developers to test sites with explicit owner permission.

---

## Critical Constraints

### Authorization Requirement

**IMPORTANT**: This library is ONLY for authorized security testing. Any assistance should assume:
- The user has explicit authorization from site owners
- Testing is for legitimate security research purposes
- Usage complies with applicable laws and regulations

### Cookie Management Architecture

**Cookie control stays in Gerador Cookies, NOT in TLS-API.**

```
Gerador Cookies:
  ├── Cookie Jar management
  ├── Cookie extraction from responses
  ├── Cookie injection into requests
  └── Session state management

TLS-API (external):
  ├── TLS fingerprinting only
  ├── HTTP request execution
  └── NO cookie management
```

---

## Code Patterns

### Configuration Pattern

```go
config := &scraper.Config{
    Domain:          "example.com",
    SensorUrl:       "/akam/13/pixel_xxxxxxxx", // Akamai script path
    SensorPostLimit: 5,                          // Max retry attempts
    AkamaiProvider:  "jevi",                     // or "n4s", "roolink"
    ProfileType:     "chrome_143",               // TLS profile
    Language:        "en-US",                    // Accept-Language
    LowSecurity:     false,                      // Validation mode
    SbSd:            false,                      // SbSd challenge mode
}
```

### Scraper Initialization

```go
scraper, err := scraper.NewScraper(proxyURL, config, profiles.Chrome_133)
if err != nil {
    return err
}
defer scraper.CloseReport()
```

### Standard Flow

```go
// 1. Get anti-bot script URL from homepage
scriptURL, err := scraper.GetAntiBotScriptURL("")

// 2. Download and encode script
script, err := scraper.GetAntiBotScript()

// 3. Generate session (solve challenge)
success, err := scraper.GenerateSession(script)

// 4. Get resulting cookies
cookies, err := scraper.GetCookies("https://example.com")
```

### SbSd Flow (Alternative)

```go
// 1. Get SbSd script URL
scriptURL, err := scraper.GetAntiBotScriptURL("")

// 2. Download script
script, err := scraper.GetAntiBotScript()

// 3. Generate SbSd challenge
sbsdData, err := scraper.GenerateSbSdChallenge(script, bmSo)

// 4. Post challenge
err = scraper.PostSbSdChallenge(sbsdData)
```

---

## Key Files Reference

| File | Purpose | Lines |
|------|---------|-------|
| `scraper/scraper.go` | Main scraper, HTTP handling, cookie management | ~1,114 |
| `scraper/akamaiSolver.go` | Provider integration, sensor generation | ~1,139 |
| `scraper/provider_cache.go` | Cache management | ~137 |
| `scraper/chrome142_simple.go` | Chrome TLS profile | ~120 |
| `scraper/ios_profiles.go` | Safari iOS TLS profiles | ~412 |

---

## Common Tasks

### Adding a New Browser Profile

1. Create new file `scraper/{browser}_profiles.go`
2. Define `ClientHelloID` with TLS spec factory
3. Define `ClientProfile` with HTTP/2 settings
4. Add profile type handling in `setHeaders()` function
5. Update documentation

### Adding a New Provider

1. Add provider-specific structs in `akamaiSolver.go`
2. Implement `solveSingle{Provider}()` method
3. Add case in `Solve()` switch statement
4. Implement SbSd variant if supported
5. Add API key management
6. Update ADR-001 documentation

### Debugging Requests

```go
config := &scraper.Config{
    // ... other config
    GenerateReport: true,  // Enable request/response logging
}

// After scraper operations:
reportPath := scraper.ReportPath()
// Report written to /tmp/getsensor-report-{timestamp}.txt
```

### Using Debug Proxy

```bash
# Set environment variable before running
export DEBUG_PROXY=http://127.0.0.1:8888  # Charles/Burp proxy
```

---

## Error Handling Patterns

### Cookie Validation

```go
// Success markers in _abck cookie
if strings.Contains(cookie.Value, "~0~") {
    // Full success
}
if config.LowSecurity && len(cookie.Value) == 541 {
    // Low security mode success
}
```

### Response Validation

```go
// Valid sensor response has no newlines
isValid := !strings.Contains(string(body), "\n")
```

### Provider Error Handling

```go
// N4S error check
if errMsg, hasError := result["error"].(string); hasError {
    return "", fmt.Errorf("provider error: %s", errMsg)
}

// Jevi returns errors in response body with non-200 status
if resp.StatusCode != 200 {
    return "", fmt.Errorf("jevi solver failed: status=%d", resp.StatusCode)
}
```

---

## Testing Considerations

### No Automated Tests

Currently no CI/CD or automated tests. Manual testing required:
1. Test against known Akamai-protected test sites
2. Verify cookie generation success
3. Check provider responses
4. Validate TLS fingerprints (using tools like ja3.io)

### Debug Mode

```go
// Environment variables for debugging
os.Setenv("isDebug", "true")
os.Setenv("DEBUG_PROXY", "http://127.0.0.1:8888")
```

---

## Performance Notes

### Current Bottlenecks

1. **Script Download**: Full script download even when cached dynamic exists
2. **Sequential Processing**: No parallel challenge attempts
3. **External API Latency**: Provider API calls add ~100-500ms

### Optimization Opportunities

1. Cache script URLs to skip homepage parsing
2. Use TLS-API connection pooling
3. Implement parallel provider fallback

---

## Security Patterns

### API Keys

Currently hardcoded in code:
```go
// Jevi
apiKey := "curiousT-a23f417f-096e-4258-adea-7ea874a57e56"

// N4S
req.Header.Set("X-API-KEY", "4DD7-F8F7-A935-972F-45B4-1A04")

// Roolink
return "2710d9bf-26fd-4add-8172-805ba613d66b"
```

**Recommendation**: Migrate to environment variables.

### Proxy Credentials

Supported in proxy URL format:
```
http://username:password@proxy.example.com:8080
```

Country detection from proxy username pattern:
```
username-country-BR-session-xxx
```

---

## Common Gotchas

1. **Cookie Domain**: Cookies are domain-scoped, ensure correct domain in config
2. **Script URL Parsing**: Different sites embed scripts differently, check `GetAntiBotScriptURL` logic
3. **SbSd vs Standard**: Some sites require SbSd mode, config `SbSd: true`
4. **Profile Mismatch**: User-Agent must match TLS profile
5. **Cache Staleness**: Use `ForceUpdateDynamics: true` if cache issues suspected

---

## Related Documentation

- [Project Charter](project_charter.md)
- [Codebase Navigation Guide](CODEBASE_GUIDE.md)
- [API Specification](API_SPECIFICATION.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)

---

*Last Updated: 2026-01-27*
