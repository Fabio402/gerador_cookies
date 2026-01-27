# Gerador Cookies - Technical Documentation

## Project Context Profile

| Attribute | Value |
|-----------|-------|
| **Project Name** | Gerador Cookies |
| **Type** | Go Library |
| **Domain** | Security Testing / Anti-Bot Research |
| **Status** | Active Development |
| **Go Version** | 1.24.1 (toolchain 1.24.4) |
| **Target Platforms** | Linux, macOS |
| **Primary Language** | Go |

### Technology Stack

- **Core**: Go 1.24.1
- **HTTP Client**: bogdanfinn/tls-client v1.11.2 (migration to TLS-API planned)
- **HTML Parsing**: goquery v1.10.3
- **Compression**: brotli, gzip
- **TLS**: bogdanfinn/utls v1.7.4-barnius

### Team & Development

- **Consumers**: Authorized developers for security testing
- **CI/CD**: Manual testing (no automated pipeline)
- **Release Process**: Manual distribution
- **Testing Scope**: Client sites with explicit authorization

---

## Layer 1: Core Project Context

### Foundational Documents

- [Project Charter](project_charter.md) - Vision, scope, and objectives
- [Architecture Decision Records](adr/) - Key technical decisions
  - [ADR-001: Multi-Provider Strategy](adr/ADR-001-multi-provider-strategy.md)
  - [ADR-002: TLS Fingerprinting Approach](adr/ADR-002-tls-fingerprinting.md)
  - [ADR-003: Cache Strategy](adr/ADR-003-cache-strategy.md)

---

## Layer 2: AI-Optimized Context Files

### Development Guides

- [AI Development Guide](CLAUDE.meta.md) - AI assistant optimization
- [Codebase Navigation Guide](CODEBASE_GUIDE.md) - Structure and key files

---

## Layer 3: Domain-Specific Context

### Technical Documentation

- [Business Logic Documentation](BUSINESS_LOGIC.md) - Core domain concepts
- [API Specification](API_SPECIFICATION.md) - Provider and internal APIs

---

## Layer 4: Development Workflow Context

### Operational Guides

- [Development Workflow Guide](CONTRIBUTING.md) - Setup and contribution
- [Troubleshooting Guide](TROUBLESHOOTING.md) - Common issues and solutions
- [Architecture Challenges](ARCHITECTURE_CHALLENGES.md) - Known issues and improvements

---

## Quick Links

### For New Developers
1. Start with [Project Charter](project_charter.md)
2. Read [Codebase Navigation Guide](CODEBASE_GUIDE.md)
3. Review [Development Workflow](CONTRIBUTING.md)

### For AI Assistants
1. Load [CLAUDE.meta.md](CLAUDE.meta.md) for context
2. Reference [API Specification](API_SPECIFICATION.md) for integrations
3. Check [Troubleshooting](TROUBLESHOOTING.md) for known issues

### For Debugging
1. Check [Troubleshooting Guide](TROUBLESHOOTING.md)
2. Review [Architecture Challenges](ARCHITECTURE_CHALLENGES.md)
3. Understand [Business Logic](BUSINESS_LOGIC.md) flows

---

## Document Maintenance

| Document | Update Frequency | Owner |
|----------|------------------|-------|
| Project Charter | On scope changes | Tech Lead |
| ADRs | On architectural decisions | Development Team |
| CLAUDE.meta.md | On pattern changes | Development Team |
| API Specification | On API changes | Development Team |
| Troubleshooting | On new issues discovered | Development Team |

---

*Last Updated: 2026-01-27*
*Documentation Version: 1.0.0*
