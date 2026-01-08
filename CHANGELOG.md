# Changelog

All notable changes to Goryu will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0-alpha] - 2024-01-08

### Added
- Initial alpha release 🎉
- High-performance HTTP router with radix tree algorithm
- Smart Context API with fluent, chainable methods
- 25+ middleware packages including:
  - Authentication (JWT, Basic Auth)
  - Session management with encryption
  - Rate limiting and circuit breaker
  - CORS, CSRF protection
  - Caching, compression
  - Structured logging and metrics
- Powerful CLI with:
  - Project scaffolding (`goryu init`)
  - Code generation (`goryu generate`)
  - Hot reload development server (`goryu dev`)
- Built-in monitoring system with:
  - Health checks at `/_health`
  - Prometheus metrics at `/_metrics`
  - Real-time dashboard at `/_dashboard`
- Comprehensive security features:
  - Path traversal protection
  - Secure file uploads
  - IP spoofing prevention
  - Timing attack protection
- Full test coverage across core packages
- Extensive documentation and examples

### Security
- All session data is encrypted using AES-GCM
- Secure defaults for cookies (HttpOnly, Secure, SameSite)
- Built-in CSRF token generation and validation
- Rate limiting to prevent abuse

### Known Issues
- This is an alpha release - API may change
- Some advanced features still in development
- Limited database integration examples

[0.1.0-alpha]: https://github.com/arthurlch/goryu/releases/tag/v0.1.0-alpha