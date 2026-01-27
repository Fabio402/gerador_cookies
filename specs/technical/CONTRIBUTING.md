# Development Workflow Guide

## Prerequisites

### Required Software

| Software | Version | Purpose |
|----------|---------|---------|
| Go | 1.24.1+ | Primary language |
| Git | Latest | Version control |

### Optional Tools

| Tool | Purpose |
|------|---------|
| Charles Proxy | HTTP debugging |
| Burp Suite | HTTP debugging |
| VS Code | IDE with Go extension |

---

## Environment Setup

### 1. Clone Repository

```bash
git clone <repository-url>
cd gerador_cookies
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Verify Setup

```bash
go build ./...
```

---

## Project Configuration

### Environment Variables

Create a `.env` file or export these variables:

```bash
# Debug mode
export isDebug=true

# Debug proxy (optional)
export DEBUG_PROXY=http://127.0.0.1:8888

# Provider cache control
export REQS_PROVIDER_CACHE_ENABLE=1
# export REQS_PROVIDER_CACHE_DISABLE=1
# export REQS_PROVIDER_CACHE_CLEAR_ON_START=1
```

---

## Development Workflow

### Creating a Consumer Application

Since this is a library, create a separate project to test:

```go
// main.go in your test project
package main

import (
    "fmt"
    "log"

    "gerador_cookies/scraper"
    "github.com/bogdanfinn/tls-client/profiles"
)

func main() {
    config := &scraper.Config{
        Domain:          "target-site.com",
        SensorUrl:       "/akam/13/pixel_xxx",
        SensorPostLimit: 5,
        AkamaiProvider:  "jevi",
        Language:        "en-US",
        GenerateReport:  true,
    }

    s, err := scraper.NewScraper("", config, profiles.Chrome_133)
    if err != nil {
        log.Fatal(err)
    }
    defer s.CloseReport()

    // Get script URL
    scriptURL, err := s.GetAntiBotScriptURL("")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Script URL: %s\n", scriptURL)

    // Update config with script URL
    config.SensorUrl = scriptURL

    // Get script
    script, err := s.GetAntiBotScript()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Script length: %d\n", len(script))

    // Generate session
    success, err := s.GenerateSession(script)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Success: %v\n", success)

    // Get cookies
    cookies, _ := s.GetCookies(fmt.Sprintf("https://%s", config.Domain))
    for _, c := range cookies {
        fmt.Printf("Cookie: %s = %s...\n", c.Name, c.Value[:min(50, len(c.Value))])
    }

    // Check report
    fmt.Printf("Report: %s\n", s.ReportPath())
}
```

### Running Tests

Currently no automated tests. Manual testing process:

1. Set up test target (authorized site)
2. Run consumer application
3. Verify cookie generation
4. Check debug report if enabled

### Debug Mode

Enable detailed logging:

```bash
export isDebug=true
go run main.go 2>&1 | tee debug.log
```

Use debug proxy:

```bash
export DEBUG_PROXY=http://127.0.0.1:8888
# Start Charles/Burp on port 8888
go run main.go
```

---

## Code Guidelines

### Go Style

Follow standard Go conventions:
- `gofmt` for formatting
- Meaningful variable names
- Comments for exported functions
- Error handling with context

### File Organization

| Directory | Content |
|-----------|---------|
| `scraper/` | All library code |
| `akt/` | Utility functions |
| `docs/` | Legacy documentation |
| `specs/technical/` | Current documentation |

### Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Exported functions | PascalCase | `GetCookies` |
| Internal functions | camelCase | `setHeaders` |
| Structs | PascalCase | `AkamaiSolver` |
| Constants | camelCase | `defaultUserAgent` |

---

## Making Changes

### Adding a New Provider

1. **Define request/response structs** in `akamaiSolver.go`:
```go
type NewProviderRequest struct {
    Site      string `json:"site"`
    Cookies   string `json:"cookies"`
    UserAgent string `json:"user_agent"`
}
```

2. **Implement solve method**:
```go
func (as *AkamaiSolver) solveSingleNewProvider(script string, index int) (bool, error) {
    // Implementation
}
```

3. **Add case in Solve()**:
```go
case "newprovider":
    return as.solveSingleNewProvider(script, i)
```

4. **Implement SbSd variant** if supported

5. **Update documentation**:
   - `API_SPECIFICATION.md`
   - `ADR-001-multi-provider-strategy.md`

### Adding a New Browser Profile

1. **Create profile file** `scraper/{browser}_profiles.go`:
```go
package scraper

import (
    "github.com/bogdanfinn/fhttp/http2"
    tls "github.com/bogdanfinn/utls"
    "github.com/bogdanfinn/tls-client/profiles"
)

var HelloNewBrowser = tls.ClientHelloID{
    Client:  "NewBrowser",
    Version: "1.0",
    // ...
}

var NewBrowserProfile = profiles.NewClientProfile(
    HelloNewBrowser,
    // HTTP/2 settings
    // ...
)

func newBrowserSpec() (tls.ClientHelloSpec, error) {
    return tls.ClientHelloSpec{
        // TLS configuration
    }, nil
}
```

2. **Add profile type handling** in `setHeaders()`:
```go
case "new_browser":
    req.Header = http.Header{
        // Browser-specific headers
    }
```

3. **Update documentation**:
   - `CODEBASE_GUIDE.md`
   - `ADR-002-tls-fingerprinting.md`

### Modifying Cache Behavior

Cache logic in `provider_cache.go`:

1. **Change TTL**:
```go
cur.ExpiresAt = time.Now().Add(48 * time.Hour)  // Change from 24h
```

2. **Change cache location**:
```go
func defaultProviderCachePath() string {
    return "/custom/path/cache.json"
}
```

3. **Update ADR-003** with rationale

---

## Debugging Tips

### Common Issues

#### 1. Cookies Not Received

Check debug report for:
- HTTP status codes
- Set-Cookie headers
- Request headers matching profile

#### 2. Provider API Errors

Enable debug mode and check:
- Request payload sent to provider
- Response from provider
- API key validity

#### 3. TLS Fingerprint Detection

Use tools like:
- https://tls.peet.ws/api/all
- https://ja3.io

Compare with real browser fingerprint.

### Debug Report Format

When `GenerateReport: true`, report written to `/tmp/getsensor-report-{timestamp}.txt`:

```
--- REQUEST (site/tls-client) ---
GET https://example.com/
curl -i -X GET 'https://example.com/' -H 'User-Agent: ...'
User-Agent: Mozilla/5.0...
Accept: text/html...

--- RESPONSE (site/tls-client) ---
Status: 200 OK
Content-Length: 12345
Set-Cookie: _abck=...

<response body>
```

---

## Release Process

### Current Process (Manual)

1. Make changes in development environment
2. Test against authorized target sites
3. Commit changes with descriptive message
4. Tag version if significant change

### Version Numbering

No formal versioning currently. Recommended:
- Patch: Bug fixes, profile updates
- Minor: New providers, new features
- Major: Breaking API changes

---

## Future Improvements

### Planned

| Improvement | Priority | Notes |
|-------------|----------|-------|
| TLS-API migration | High | Replace bogdanfinn/tls-client |
| Automated testing | Medium | Unit and integration tests |
| API key from env | Medium | Security improvement |
| CI/CD pipeline | Low | Automated builds and tests |

### Contributing Ideas

1. Check `ARCHITECTURE_CHALLENGES.md` for known issues
2. Discuss with team before major changes
3. Update documentation with changes

---

## Support

For questions or issues:
1. Check existing documentation
2. Review code comments
3. Consult team members

---

*Last Updated: 2026-01-27*
