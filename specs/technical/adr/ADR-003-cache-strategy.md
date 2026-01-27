# ADR-003: Cache Strategy

## Status

**Accepted**

## Context

Provider API calls for dynamic data generation are expensive operations:
- Network latency to external services
- API rate limits and quotas
- Redundant computation for same domain/provider combinations

We need a caching strategy to optimize performance without sacrificing correctness.

## Decision

Implement a **file-based provider cache** with the following characteristics:

| Attribute | Value |
|-----------|-------|
| Storage | JSON file (`~/.cache/reqs/provider-cache.json`) |
| TTL | 24 hours |
| Key Structure | `{domain}|{provider}|{mode}` |
| Scope | Per-user, persists across sessions |
| Thread Safety | Mutex-protected read/write |

## Rationale

### Why Cache?

1. **Script URLs**: Akamai script URLs don't change frequently per domain
2. **Dynamic Data**: Provider-generated dynamic data valid for extended periods
3. **API Efficiency**: Reduces redundant provider API calls
4. **Performance**: Eliminates script download when cached dynamic exists

### Cache Key Design

```
{domain}|{provider}|{mode}

Examples:
- "example.com|jevi|sensor"
- "shop.com|n4s|sbsd"
- "site.com|roolink|sensor"
```

This allows:
- Same domain with different providers cached separately
- Sensor vs SbSd modes cached separately
- Easy cache invalidation per domain

### TTL Selection (24 hours)

| Duration | Pros | Cons |
|----------|------|------|
| 1 hour | Fresh data | Too many API calls |
| 24 hours | Good balance | Data may be stale occasionally |
| 7 days | Minimal API calls | Higher risk of stale data |

24 hours selected as balance between freshness and efficiency.

## Implementation

### Cache Entry Structure

```go
type providerCacheEntry struct {
    ScriptURL  string    `json:"scriptUrl"`   // Akamai script URL
    Dynamic    string    `json:"dynamic"`     // Provider dynamic data
    ExpiresAt  time.Time `json:"expiresAt"`   // TTL expiration
    UpdatedAt  time.Time `json:"updatedAt"`   // Last update time
    Domain     string    `json:"domain"`      // Target domain
    Provider   string    `json:"provider"`    // Provider name
    Mode       string    `json:"mode"`        // sensor or sbsd
}
```

### Cache Operations

```go
// Get - returns entry if valid, deletes if expired
func (pc *ProviderCache) Get(domain, provider, mode string) (providerCacheEntry, bool)

// Upsert - updates or inserts entry, extends TTL
func (pc *ProviderCache) Upsert(domain, provider, mode string, scriptURL *string, dynamic *string)
```

### Environment Controls

| Variable | Effect |
|----------|--------|
| `REQS_PROVIDER_CACHE_ENABLE=1` | Enable caching |
| `REQS_PROVIDER_CACHE_DISABLE=1` | Disable caching |
| `REQS_PROVIDER_CACHE_CLEAR_ON_START=1` | Clear cache on startup |

## Cache Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Request Flow                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   1. Check Cache                                             │
│      │                                                       │
│      ├─── Hit & Valid ──► Use cached Dynamic                │
│      │                    Skip script download               │
│      │                                                       │
│      └─── Miss/Expired ──► Fetch script                     │
│                            Call provider API                 │
│                            Update cache                      │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Consequences

### Positive

- Significant reduction in provider API calls
- Faster subsequent requests for same domain
- Reduced latency when cache hits
- Persistent across application restarts

### Negative

- Stale data possible within TTL window
- Disk I/O for cache operations
- Cache file can grow with many domains
- Manual cache clearing may be needed

### Mitigation

- `ForceUpdateDynamics` config option bypasses cache
- Environment variable to clear cache on start
- 24-hour TTL limits staleness window

## Related Decisions

- [ADR-001: Multi-Provider Strategy](ADR-001-multi-provider-strategy.md)
- [ADR-002: TLS Fingerprinting Approach](ADR-002-tls-fingerprinting.md)

---

*Decision Date: 2024*
*Last Reviewed: 2026-01-27*
