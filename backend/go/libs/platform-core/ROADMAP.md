# Platform Core - Temporary Roadmap

This roadmap is temporary and will be removed once the library is finalized.

- Config validation: add optional required-field checks (database URL, JWT secret, brokers).
- Secrets: document/offer helper to load sensitive values from secret manager (keep env first).
- Logging: add a small logger wrapper with structured fields and service metadata.
- Health/check tools: add helper to expose basic health endpoints or readiness probes.
- Testing: add unit tests for config defaults and env parsing (Kafka list, ports, JWT expiration).
- Examples: add runnable sample service showing config wiring for HTTP + database + Kafka.
