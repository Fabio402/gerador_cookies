# Technical Context: Gerador Cookies

## Project Overview

**Gerador Cookies** is a Go-based library designed to generate and manage Akamai anti-bot protection bypass cookies and sensor data. The project implements sophisticated mechanisms to handle Akamai's security challenges by generating valid sensor payloads that can be submitted to protected websites.

### Primary Purpose

This is a specialized library that handles Akamai's bot detection system, which protects many e-commerce and high-traffic websites. The library provides:

- TLS fingerprinting to mimic real browsers
- Sensor data generation through multiple provider APIs
- Cookie management for session persistence
- Support for multiple browser profiles (Chrome, Safari, Firefox)

### Project Classification

| Attribute | Value |
|-----------|-------|
| Type | Go Library |
| Domain | Web Scraping / Anti-Bot Bypass |
| Status | Active Development |
| Go Version | 1.24.1 (toolchain 1.24.4) |

---

## Technology Stack

### Core Language

- **Go 1.24.1** - Primary development language
- Module path: `github.com/freitas-tech/scraper` (implied from structure)

### Primary Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/bogdanfinn/tls-client` | v1.11.2 | TLS fingerprinting & proxy support |
| `github.com/bogdanfinn/fhttp` | v0.6.3 | HTTP client compatible with TLS spoofing |
| `github.com/bogdanfinn/utls` | v1.7.4-barnius | uTLS for custom TLS ClientHello |
| `github.com/PuerkitoBio/goquery` | v1.10.3 | HTML parsing and CSS selectors |
| `github.com/Noooste/akamai-v2-deobfuscator` | v0.0.0-20240221 | Akamai JavaScript deobfuscation |
| `github.com/andybalholm/brotli` | v1.2.0 | Brotli compression support |

### Key Capabilities

1. **TLS Fingerprinting** - Mimics real browser TLS signatures
2. **HTTP/2 Support** - Custom settings matching browser profiles
3. **Proxy Support** - With country/language auto-detection
4. **Multiple Compression Formats** - gzip, brotli, deflate
5. **Cookie Management** - Per-domain, per-profile isolation
6. **Multiple Browser Profiles** - Chrome, Safari iOS, Firefox

---

## Project Structure

```
gerador_cookies/
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
├── akt/                        # Utility package
│   └── logger.go               # Debug logging utilities
├── scraper/                    # Core package (~2,927 LoC)
│   ├── scraper.go              # Main scraper implementation (1,114 LoC)
│   ├── akamaiSolver.go         # Akamai challenge solver (1,139 LoC)
│   ├── provider_cache.go       # Provider cache management (137 LoC)
│   ├── chrome142_simple.go     # Chrome 142 TLS fingerprint (120 LoC)
│   ├── ios_profiles.go         # iOS TLS profiles (412 LoC)
│   └── utils.go                # Utility functions (5 LoC)
└── docs/                       # Technical documentation
```

### Code Distribution

| Module | Lines of Code | Percentage |
|--------|---------------|------------|
| akamaiSolver.go | 1,139 | 39% |
| scraper.go | 1,114 | 38% |
| ios_profiles.go | 412 | 14% |
| provider_cache.go | 137 | 5% |
| chrome142_simple.go | 120 | 4% |
| **Total** | **~2,927** | **100%** |

---

## External Integrations

### Provider APIs

The library integrates with external sensor generation services:

| Provider | Endpoint | Purpose |
|----------|----------|---------|
| Jevi | `jevi.dev` | Primary sensor generation |
| N4S | `n4s.xyz` | Alternative sensor generation |
| Roolink | `roolink.io` | Backup sensor generation |

### Akamai Integration Points

- **Homepage Request** - Seed cookie collection
- **Script Extraction** - Anti-bot script URL discovery
- **Sensor Endpoint** - Challenge submission (`/on/<path>`)

---

## Development Environment

### Requirements

- Go 1.24.1 or later
- Network access to provider APIs
- Optional: Debug proxy (Charles, Burp Suite)

### Environment Variables

```bash
# Debug mode
isDebug=true

# Debug proxy for traffic inspection
DEBUG_PROXY=http://127.0.0.1:8888

# Provider cache control
REQS_PROVIDER_CACHE_ENABLE=1
REQS_PROVIDER_CACHE_DISABLE=1
REQS_PROVIDER_CACHE_CLEAR_ON_START=1
```

---

## Constraints and Limitations

### Technical Constraints

1. **Library Only** - No standalone entry point (main.go)
2. **External API Dependency** - Requires access to provider APIs
3. **Platform Specific** - Report files written to `/tmp/` (Unix-like systems)
4. **Linear Processing** - Single challenge flow, no parallel attempts

### Operational Constraints

1. **API Rate Limits** - Subject to provider API limitations
2. **Script Versioning** - Akamai scripts change frequently
3. **Profile Freshness** - TLS profiles must match current browser versions

---

## Security Considerations

### Sensitive Data Handling

- API keys stored in code (not recommended for production)
- Cookie data managed in memory
- Proxy credentials passed through configuration

### Network Security

- TLS 1.3 support
- Custom cipher suites matching browser profiles
- Certificate pinning not implemented (intentional for proxy support)

---

## Related Documentation

- [Architecture Overview](01-architecture.md)
- [Component Documentation](02-components.md)
- [API and Data Flow](03-api-data-flow.md)
- [Configuration and Security](04-configuration-security.md)
