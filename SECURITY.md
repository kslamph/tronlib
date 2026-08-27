# Security Policy

## Supported versions

Only the latest released version of tronlib receives security fixes.

## Reporting a vulnerability

**Do not open a public issue for security problems.**

Report privately via [GitHub security advisories](https://github.com/kslamph/tronlib/security/advisories/new).
Include a description, impact assessment, and reproduction steps if possible.

You will get an acknowledgement within 7 days. Please avoid public disclosure
until a fix is released; we will credit you in the advisory unless you prefer
to remain anonymous.

## Scope

tronlib is a client-side SDK. In scope:

- Key or mnemonic handling that could leak secrets (logs, error messages,
  serialization)
- Signature construction flaws that produce invalid or replayable signatures
- Parsing of external input (ABI JSON, event logs, node responses) that can
  panic or corrupt state

Out of scope:

- Vulnerabilities in dependencies themselves — report upstream, but tell us so
  we can bump the dependency
- Compromised node endpoints, key theft from your own environment, or network
  attacks between you and your TRON node
