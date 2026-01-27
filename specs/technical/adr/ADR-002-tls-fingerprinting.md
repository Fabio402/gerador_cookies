# ADR-002: TLS Fingerprinting Approach

## Status

**Accepted** (with planned migration)

## Context

Akamai's anti-bot protection analyzes TLS fingerprints to detect automated requests. To successfully generate valid cookies, the library must present TLS fingerprints that match real browsers.

### Current State

The library currently uses `bogdanfinn/tls-client` directly for TLS fingerprinting.

### Planned State

Migration to internal **TLS-API** service for centralized fingerprint management.

## Decision

### Current Implementation

Use `bogdanfinn/tls-client` library with custom browser profiles defined in the codebase.

### Planned Migration

Migrate to TLS-API service with the following separation of concerns:

| Component | Responsibility |
|-----------|----------------|
| **TLS-API** | HTTP requests with TLS fingerprinting |
| **Gerador Cookies** | Cookie management, session state, provider orchestration |

**Important**: Cookie management remains in Gerador Cookies. TLS-API only handles the HTTP transport layer with fingerprinting.

## Rationale

### Current Approach (bogdanfinn/tls-client)

**Pros:**
- Direct control over TLS settings
- No external service dependency
- All-in-one solution

**Cons:**
- Duplicated fingerprint code across projects
- Manual profile updates required
- No centralized management

### Planned Approach (TLS-API)

**Pros:**
- Centralized fingerprint management
- Single point of update for new browser versions
- Shared across multiple projects
- Better performance through connection pooling

**Cons:**
- Network dependency on TLS-API service
- Additional infrastructure to maintain

## Browser Profiles Supported

### Current Profiles

| Browser | Profile ID | HTTP/2 Settings |
|---------|-----------|-----------------|
| Chrome 142/143 | `Chrome142Simple` | Header Table: 65536, Initial Window: 6291456 |
| Safari iOS 18.5 | `StandardIOS18` | Initial Window: 2097152, Max Streams: 100 |
| Firefox 135 | `firefox_135` | Custom HTTP/2 settings |

### Profile Update Strategy

Profiles are updated **on-demand** when:
- Detection issues are observed
- New browser versions gain significant market share
- Provider compatibility requires updates

## Implementation Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Gerador Cookies                           │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              Cookie Management                       │    │
│  │  - Cookie Jar (per session)                         │    │
│  │  - Cookie extraction from responses                 │    │
│  │  - Cookie injection into requests                   │    │
│  └─────────────────────────────────────────────────────┘    │
│                           │                                  │
│                           ▼                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              Provider Orchestration                  │    │
│  │  - Sensor generation coordination                   │    │
│  │  - Challenge submission                             │    │
│  │  - Response validation                              │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼ (HTTP requests)
┌─────────────────────────────────────────────────────────────┐
│                       TLS-API                                │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              TLS Fingerprinting                      │    │
│  │  - Browser profile simulation                       │    │
│  │  - HTTP/2 settings                                  │    │
│  │  - Cipher suites                                    │    │
│  │  - NO cookie management (handled by caller)         │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Consequences

### Positive

- Accurate browser simulation
- Multiple browser support
- Centralized profile management (with TLS-API)

### Negative

- Requires manual profile updates (current)
- Additional service dependency (with TLS-API)
- Complexity in maintaining fingerprint accuracy

### Migration Considerations

When migrating to TLS-API:
1. Gerador Cookies continues to manage all cookies
2. TLS-API receives cookies via request headers
3. TLS-API returns cookies via response headers
4. Gerador Cookies updates its cookie jar from responses

## Related Decisions

- [ADR-001: Multi-Provider Strategy](ADR-001-multi-provider-strategy.md)
- [ADR-003: Cache Strategy](ADR-003-cache-strategy.md)

---

*Decision Date: 2024*
*Last Reviewed: 2026-01-27*
