# Plugin Threat Model Review

Last updated: 2026-02-19

Scope reviewed:

- `internal/plugin/sandbox/types.go`
- `internal/plugin/sandbox/raw_local.go`
- `internal/plugin/scheduler/scheduler.go`
- `internal/plugin/scheduler/news_assistant_adapter.go`
- `internal/plugin/newsassistant/news_assistant.go`
- `internal/plugin/newsassistant/demo_gateway.go`
- `internal/workflow/job/cron_service.go`
- `cmd/server/main.go`

## Findings and Mitigations

### Blockers (must fix before broad rollout)

1. Non-atomic slot idempotency lock (`Get` + `Set`) allows duplicate same-slot executions.
   - Risk: replay/idempotency abuse
   - Mitigation: switch to atomic `SETNX`/CAS-style lock with TTL and run-id dedup in enqueuer path.

2. `raw-local` command execution is host-process based and not a true sandbox.
   - Risk: privilege escalation and data leakage via filesystem/process access
   - Mitigation: keep `raw-local` dev-only, add production-grade isolation (container/VM/seccomp), tighten arg/path policy.

3. High/critical risk approvals depend on caller-provided `AllowExplicitRisk`.
   - Risk: consent/risk bypass
   - Mitigation: move approval to server-side policy decision and reject caller-controlled bypass flags.

4. Side-effect ordering in news assistant (`send` before dedup state write) enables resend on retry/failure.
   - Risk: replay/idempotency abuse
   - Mitigation: idempotent send key (deterministic message id), or transactional outbox/marker before side effects.

5. Scheduler run path triggers plugin actions without explicit consent context.
   - Risk: consent bypass
   - Mitigation: require per-tenant/plugin consent state and enforce in gateway policy before action execution.

### Non-blockers (track and harden)

1. Dedup based on LLM summary hash can drift with nondeterministic outputs.
   - Mitigation: dedup on stable article IDs or canonicalized source content hash.

2. DLQ/logging may persist raw error strings containing sensitive content.
   - Mitigation: structured error codes + redaction before storage/logging.

3. Dotenv auto-loading from working directory can introduce config override risk.
   - Mitigation: disable dotenv loading in production and constrain env file source.

4. Demo gateway currently overwrites dedup hashes with incremental subset.
   - Mitigation: merge historic + new hashes or use set semantics in storage backend.

## Status

- Mitigation checklist: tracked with explicit blockers.
- Rollout gating: blockers above must be resolved before moving beyond allowlist beta.
