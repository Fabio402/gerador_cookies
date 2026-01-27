# Troubleshooting Guide

## Quick Diagnosis

```
┌─────────────────────────────────────────────────────────────┐
│                    Issue Diagnosis Tree                      │
└─────────────────────────────────────────────────────────────┘

Problem: Cookie generation failing
    │
    ├── Homepage request fails?
    │   ├── Yes → Check proxy, TLS profile, site authorization
    │   └── No → Continue
    │
    ├── Script URL not found?
    │   ├── Yes → Check HTML parsing, site uses Akamai?
    │   └── No → Continue
    │
    ├── Provider API error?
    │   ├── Yes → Check API key, provider availability
    │   └── No → Continue
    │
    ├── Sensor submission fails?
    │   ├── Yes → Check profile/UA match, headers order
    │   └── No → Continue
    │
    └── Cookie validation fails?
        ├── Yes → Try different provider, increase retries
        └── No → Success!
```

---

## Common Issues

### 1. Homepage Request Blocked (403/503)

**Symptoms**:
```
homepage blocked: status=403 Forbidden
```

**Causes**:
- IP blocked by site
- TLS fingerprint detected as bot
- Missing required headers
- Site requires specific geolocation

**Solutions**:

| Solution | How |
|----------|-----|
| Use proxy | `scraper.NewScraper("http://user:pass@proxy:8080", config, profile)` |
| Change TLS profile | Try `profiles.Chrome_133`, `StandardIOS18`, `firefox_135` |
| Check authorization | Ensure you have permission to test this site |

**Debug**:
```bash
export DEBUG_PROXY=http://127.0.0.1:8888
# Check request in Charles/Burp
```

---

### 2. Script URL Not Found

**Symptoms**:
```
Found sensor URL: (empty)
```

**Causes**:
- Site doesn't use Akamai protection
- Script embedded differently than expected
- Site changed structure

**Solutions**:

1. **Verify Akamai presence**:
   - Check page source for `/akam/` or `/on/` paths
   - Look for `_abck` cookie in browser DevTools

2. **Manual URL extraction**:
   ```go
   config.SensorUrl = "/akam/13/pixel_xxx"  // Manually set
   ```

3. **Check SbSd mode**:
   ```go
   config.SbSd = true  // Try SbSd mode if standard fails
   ```

**Debug**:
```go
config.GenerateReport = true
// Check report for homepage HTML
```

---

### 3. Provider API Errors

**Symptoms**:
```
jevi solver failed: status=400 body=...
n4s error: Invalid request
roolink sensor error: ...
```

**Causes**:
- Invalid API key
- Malformed request
- Provider service down
- Rate limiting

**Solutions**:

| Provider | Check |
|----------|-------|
| Jevi | Verify `x-key` header, check gzip compression |
| N4S | Verify `X-API-KEY` header, check JSON format |
| Roolink | Verify `x-api-key` header, check script format |

**Try different provider**:
```go
config.AkamaiProvider = "n4s"  // Switch from jevi
```

**Debug**:
```go
// Enable report to see exact request/response
config.GenerateReport = true
```

---

### 4. Sensor Submission Fails

**Symptoms**:
- Multiple retry attempts
- No `~0~` in `_abck` cookie
- Response contains newlines

**Causes**:
- User-Agent mismatch with TLS profile
- Headers order incorrect
- Sensor data stale
- Site-specific requirements

**Solutions**:

1. **Verify UA/Profile match**:
   ```go
   // These must match!
   config.ProfileType = "chrome_143"
   config.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"
   ```

2. **Force cache refresh**:
   ```go
   config.ForceUpdateDynamics = true
   ```

3. **Increase retry limit**:
   ```go
   config.SensorPostLimit = 10
   ```

4. **Try different provider**:
   ```go
   config.AkamaiProvider = "roolink"
   ```

---

### 5. Cookies Not Persisting

**Symptoms**:
- `GetCookies()` returns empty
- Cookies disappear between requests

**Causes**:
- Domain mismatch
- Cookie jar not initialized
- Scraper recreated between calls

**Solutions**:

1. **Check domain consistency**:
   ```go
   // Use same domain format everywhere
   cookies, _ := scraper.GetCookies("https://example.com")
   // NOT "example.com" or "http://example.com"
   ```

2. **Reuse scraper instance**:
   ```go
   // Don't create new scraper between operations
   scraper, _ := NewScraper(...)
   scraper.GetAntiBotScriptURL("")
   scraper.GetAntiBotScript()
   scraper.GenerateSession(script)
   scraper.GetCookies(url)  // All on same instance
   ```

---

### 6. Cache Issues

**Symptoms**:
- Stale data causing failures
- Cache not being used
- Cache not persisting

**Solutions**:

1. **Clear cache**:
   ```bash
   rm ~/.cache/reqs/provider-cache.json
   ```

2. **Force bypass**:
   ```go
   config.ForceUpdateDynamics = true
   ```

3. **Enable cache**:
   ```bash
   export REQS_PROVIDER_CACHE_ENABLE=1
   ```

4. **Clear on start**:
   ```bash
   export REQS_PROVIDER_CACHE_CLEAR_ON_START=1
   ```

---

### 7. Proxy Connection Issues

**Symptoms**:
```
error parsing proxy URL: ...
connection refused
proxy authentication failed
```

**Solutions**:

1. **Verify proxy format**:
   ```go
   // Correct formats
   "http://host:port"
   "http://user:pass@host:port"
   "socks5://host:port"
   ```

2. **Test proxy independently**:
   ```bash
   curl -x http://user:pass@proxy:8080 https://httpbin.org/ip
   ```

3. **Check proxy credentials**:
   - URL-encode special characters in password
   - Verify whitelist includes your IP

---

### 8. SbSd Mode Issues

**Symptoms**:
- SbSd challenge rejected
- Missing `bm_so` cookie
- Status not 200/202

**Causes**:
- Wrong mode selected
- Missing required cookies
- Script version mismatch

**Solutions**:

1. **Ensure SbSd mode enabled**:
   ```go
   config.SbSd = true
   config.SbSdProvider = "n4s"  // or "jevi", "roolink"
   ```

2. **Check for bm_so cookie**:
   ```go
   cookies, _ := scraper.GetCookies(url)
   for _, c := range cookies {
       if c.Name == "bm_so" {
           // Extract value for SbSd
       }
   }
   ```

---

## Debug Checklist

### Before Reporting Issue

- [ ] Enabled `GenerateReport: true`
- [ ] Checked debug report file
- [ ] Verified authorization for target site
- [ ] Tested with debug proxy
- [ ] Tried different provider
- [ ] Tried different TLS profile
- [ ] Cleared cache

### Information to Collect

1. **Configuration used** (sanitize API keys)
2. **Error messages** (full stack trace)
3. **Debug report** (request/response details)
4. **Target site** (if shareable)
5. **Provider used**
6. **TLS profile used**

---

## Performance Troubleshooting

### Slow Cookie Generation

**Causes**:
- Network latency to providers
- Full script download each time
- Many retry attempts

**Solutions**:

1. **Enable caching**:
   ```bash
   export REQS_PROVIDER_CACHE_ENABLE=1
   ```

2. **Use geographically closer proxy**:
   - Reduce latency to target site

3. **Reduce retry limit if timing out**:
   ```go
   config.SensorPostLimit = 3
   ```

### Memory Issues

**Causes**:
- Large script downloads
- Many concurrent scrapers

**Solutions**:
- Reuse scraper instances
- Close report files: `scraper.CloseReport()`
- Limit concurrent operations

---

## Provider-Specific Issues

### Jevi

| Issue | Solution |
|-------|----------|
| 400 Bad Request | Check gzip compression, JSON format |
| Invalid credentials | Verify `x-key` header |
| Empty EncodedData | First request, cache will populate |

### N4S

| Issue | Solution |
|-------|----------|
| v3_values empty | Check script base64 encoding |
| sensor error | Verify dynamic data format |
| API key invalid | Check `X-API-KEY` header |

### Roolink

| Issue | Solution |
|-------|----------|
| parse failed | Decode base64 before sending |
| sensor missing | Check scriptData format |
| sbsd missing vid | Extract from script URL `?v=` param |

---

## Emergency Procedures

### Complete Failure - All Providers

1. Verify target site still uses Akamai
2. Check if Akamai updated their system
3. Test with fresh TLS profiles
4. Contact provider support

### API Keys Compromised

1. Rotate keys with providers
2. Update keys in codebase
3. Clear all caches
4. Review access logs

---

## Related Documentation

- [Business Logic](BUSINESS_LOGIC.md) - Workflow details
- [API Specification](API_SPECIFICATION.md) - Provider API details
- [Architecture Challenges](ARCHITECTURE_CHALLENGES.md) - Known limitations

---

*Last Updated: 2026-01-27*
