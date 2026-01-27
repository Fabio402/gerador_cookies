# ADR-001: Multi-Provider Strategy

## Status

**Accepted**

## Context

The library needs to generate Akamai sensor data to obtain valid session cookies. This sensor generation is complex and requires specialized services. Several third-party providers offer this capability:

- **Jevi** (jevi.dev)
- **N4S** (n4s.xyz)
- **Roolink** (roolink.io)

We need to decide whether to use a single provider or support multiple providers.

## Decision

**Support multiple providers with configurable selection.**

The library will integrate with all three providers (Jevi, N4S, Roolink), allowing consumers to:
1. Select their preferred provider via configuration
2. Use different providers for different use cases
3. Switch providers without code changes

## Rationale

### Reasons for Multi-Provider Support

1. **Redundancy/Failover**: If one provider experiences downtime or issues, testing can continue with another provider

2. **Comparative Testing**: Different providers may have different success rates for different sites, allowing testers to compare effectiveness

3. **Client Requirements**: Some clients may have existing relationships or preferences for specific providers

4. **Risk Mitigation**: Reduces single point of failure dependency

### Trade-offs Considered

| Approach | Pros | Cons |
|----------|------|------|
| Single Provider | Simpler code, less maintenance | Single point of failure, no comparison |
| Multi-Provider | Redundancy, flexibility, comparison | More code, more maintenance, complexity |

### Provider Comparison

| Provider | Sensor Endpoint | Dynamic Endpoint | SbSd Support |
|----------|-----------------|------------------|--------------|
| Jevi | POST /v3/solve | Inline (EncodedData header) | Yes |
| N4S | POST /sensor | GET /v3_values | Yes |
| Roolink | POST /akamai/sensor | POST /parse | Yes |

## Consequences

### Positive

- High availability through provider redundancy
- Flexibility in testing different provider approaches
- Ability to optimize per-site provider selection
- Future-proofing for new providers

### Negative

- Increased code complexity (3x integration code)
- More maintenance when providers change APIs
- API keys management for multiple services
- Inconsistent response formats require normalization

### Neutral

- Provider selection is manual (no automatic failover implemented)
- Each provider has different API contracts

## Implementation Notes

```go
// Provider selection via Config.AkamaiProvider
switch as.scraper.config.AkamaiProvider {
case "n4s":
    return as.solveSingleN4S(hash, i)
case "jevi":
    return as.solveSingle(script, i)
case "roolink":
    return as.solveSingleRoolink(script, scriptData, i)
}
```

## Related Decisions

- [ADR-002: TLS Fingerprinting Approach](ADR-002-tls-fingerprinting.md)
- [ADR-003: Cache Strategy](ADR-003-cache-strategy.md)

---

*Decision Date: 2024*
*Last Reviewed: 2026-01-27*
