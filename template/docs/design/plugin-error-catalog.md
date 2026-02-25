# Plugin Error Catalog

Last updated: 2026-02-19

## Error Codes

| Code | HTTP | Retryable | UI Behavior |
|---|---:|---|---|
| `PERMISSION_DENIED` | 403 | no | Show blocked-state with policy reason |
| `CAPABILITY_REQUIRES_CONSENT` | 403 | no (until consent change) | Show consent prompt |
| `TOKEN_EXPIRED` | 401 | yes (refresh token) | Silent refresh then retry once |
| `TOKEN_REVOKED` | 401 | no | Force restart workflow/run |
| `RUN_REVOKED` | 403 | no | Mark run as revoked |
| `CAPABILITY_NOT_ALLOWED` | 403 | no | Show capability mismatch |
| `PLUGIN_NOT_ENABLED` | 409 | no | Show enable CTA |
| `PLUGIN_NOT_HEALTHY` | 503 | yes (backoff) | Show transient warning |
| `REGISTRY_SNAPSHOT_INVALID` | 409 | no | Keep last-known-good snapshot |
| `DELEGATION_DEPTH_EXCEEDED` | 400 | no | Explain chain limit reached |
| `DELEGATION_EXPORT_MISMATCH` | 400 | no | Explain action not exported |
| `SCHEDULER_SLOT_LOCKED` | 409 | yes | Skip duplicate slot run |
| `DLQ_ENQUEUED` | 202 | no | Show run moved to DLQ |

## Retry Guidance

- Retryable: `TOKEN_EXPIRED`, `PLUGIN_NOT_HEALTHY`, transient infra/network errors
- Non-retryable without state change: consent/policy/delegation errors

## UI Guidance

- Always display stable error code + short reason
- For policy denies, surface applied policy scope in diagnostics panel
- For consent-related denies, direct to consent flow instead of generic failure toast
