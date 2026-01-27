# Project Charter: Gerador Cookies

## Executive Summary

**Gerador Cookies** is a Go library designed for authorized security testing that generates valid session cookies for websites protected by Akamai anti-bot systems. The library provides sophisticated TLS fingerprinting capabilities and integrates with external sensor generation providers to facilitate security research and penetration testing.

---

## Vision Statement

To provide a reliable, performant, and maintainable library for authorized security researchers to test Akamai-protected websites, enabling comprehensive security assessments with minimal configuration overhead.

---

## Project Objectives

### Primary Objectives

1. **Cookie Generation**: Generate valid Akamai session cookies (`_abck`, `bm_sz`) for authorized security testing
2. **Browser Simulation**: Accurately simulate real browser TLS fingerprints (Chrome, Safari, Firefox)
3. **Provider Flexibility**: Support multiple sensor generation providers for redundancy and comparison
4. **Performance**: Minimize latency and maximize throughput for testing scenarios

### Secondary Objectives

1. **Caching**: Reduce redundant API calls through intelligent caching
2. **Debugging**: Provide comprehensive logging and reporting for troubleshooting
3. **Maintainability**: Keep browser profiles up-to-date with current browser versions

---

## Scope

### In Scope

| Feature | Description |
|---------|-------------|
| TLS Fingerprinting | Chrome 142/143, Safari iOS 18, Firefox 135 profiles |
| Provider Integration | Jevi, N4S, Roolink sensor generation APIs |
| Cookie Management | Automatic cookie jar management per session |
| Proxy Support | HTTP/HTTPS proxy with country-based language detection |
| Caching | 24-hour TTL provider data cache |
| Debugging | Request/response report generation |
| Two Modes | Standard sensor (_abck) and SbSd challenge modes |

### Out of Scope

| Feature | Reason |
|---------|--------|
| Standalone CLI | Library-only design, no main.go entry point |
| GUI Interface | Designed for programmatic use |
| Automated Testing | No CI/CD pipeline (manual testing only) |
| Windows Support | Linux and macOS only |
| Rate Limiting | Delegated to provider APIs |

### Future Scope (Planned)

| Feature | Priority | Notes |
|---------|----------|-------|
| TLS-API Migration | High | Replace bogdanfinn/tls-client with internal TLS-API |
| Performance Optimization | High | Reduce latency, improve throughput |
| Additional Browser Profiles | Medium | Edge, Opera, mobile variants |
| Automated Testing | Medium | Unit and integration tests |

---

## Stakeholders

| Role | Responsibility |
|------|----------------|
| **Authorized Testers** | Primary consumers - execute security tests |
| **TK Development Team** | Maintain and enhance the library |
| **Provider Services** | External APIs (Jevi, N4S, Roolink) for sensor generation |
| **Client Sites** | Target sites with explicit testing authorization |

---

## Technical Constraints

### Hard Constraints

1. **Authorization Required**: Only test sites with explicit owner permission
2. **Provider Dependency**: Requires active API access to at least one provider
3. **Platform Limited**: Unix-like systems only (Linux, macOS)
4. **Library Design**: No standalone execution capability

### Soft Constraints

1. **API Keys**: Currently hardcoded (should migrate to environment variables)
2. **Profile Updates**: Manual updates required when browser versions change
3. **No Parallelism**: Single challenge flow per scraper instance

---

## Success Criteria

| Metric | Target | Current |
|--------|--------|---------|
| Cookie Generation Success Rate | >90% | Varies by site |
| Average Latency | <500ms | ~500-1000ms |
| Provider Failover | Automatic | Manual selection |
| Profile Currency | Latest -2 versions | Chrome 143, Firefox 135 |

---

## Risk Assessment

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Provider API unavailability | High | Medium | Multi-provider support |
| TLS fingerprint detection | High | Medium | Regular profile updates |
| Akamai script changes | Medium | High | Dynamic script extraction |
| API key exposure | Medium | Low | Migration to env vars planned |

---

## Project Timeline

### Current Phase: Active Development

- Maintaining current functionality
- Responding to provider/Akamai changes
- Planning TLS-API migration

### Planned Phases

1. **Phase 1**: TLS-API integration
2. **Phase 2**: Performance optimization
3. **Phase 3**: Test coverage implementation
4. **Phase 4**: Additional browser profiles

---

## Governance

### Decision Making

- Technical decisions documented via ADRs
- Major changes require team review
- Provider selection based on reliability metrics

### Communication

- Code changes via Git commits
- Documentation updates alongside code changes
- Issue tracking for bugs and enhancements

---

## Related Documentation

- [Architecture Decision Records](adr/)
- [Codebase Navigation Guide](CODEBASE_GUIDE.md)
- [API Specification](API_SPECIFICATION.md)
- [TLS-API Documentation](https://github.com/Fabio402/tls-api) (external)

---

*Document Version: 1.0.0*
*Last Updated: 2026-01-27*
*Status: Approved*
