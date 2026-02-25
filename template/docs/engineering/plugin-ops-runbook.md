# Plugin Ops Runbook

Last updated: 2026-02-19

## Scope

This runbook covers plugin platform operations for:

- Sandbox execution (`raw-local` backend)
- Scheduler-driven plugin runs (08:00 UTC slot)
- Reference plugin (`news-assistant`) audit tracing

## Operator Onboarding

1. Verify service boots and plugin module wiring is loaded.
   - `go build ./cmd/server`
2. Verify plugin tests pass locally.
   - `go test ./internal/plugin/...`
3. Verify scheduler idempotency behavior.
   - Run `scheduler.RunSlot` twice with same slot in tests and confirm one enqueue only.
4. Verify sandbox guardrails.
   - `raw-local` allowlist blocks commands outside policy.
   - High/critical profiles require explicit risk approval.

## Risk Prompt Templates

Use these prompts when approving/denying elevated plugin operations:

- High-risk command request:
  - "This plugin requests high-risk capability execution. Confirm explicit approval for this run only, or deny."
- Critical-risk command request:
  - "This plugin requests critical-risk capability execution. Confirm explicit approval with incident ticket ID, or deny."
- Repeat request after prior deny:
  - "This request was denied previously for this slot. Confirm override with reason, or keep denied."

## Revoke and Denylist Incident Response

### Trigger Conditions

- Suspicious command payload or unexpected capability escalation
- Repeated scheduler failures reaching DLQ threshold
- Unauthorized command invocation attempts in audit logs

### Immediate Actions (First 15 minutes)

1. Stop new plugin runs by disabling scheduler trigger or pausing worker node.
2. Revoke active high/critical run approvals.
3. Add offending command or plugin instance to denylist policy.
4. Capture incident timeline and impacted run IDs.

### Containment and Recovery

1. Confirm no additional runs for impacted `instance+slot` keys.
2. Purge or quarantine DLQ payloads that contain sensitive data.
3. Roll forward patched policy and re-enable scheduler in controlled mode.
4. Re-run a single canary plugin slot and validate audit trace completeness.

## Observability Dashboard Guidance

Track these baseline indicators:

- Scheduler:
  - slot attempts
  - slot idempotent skips
  - retries and DLQ writes
- Sandbox:
  - command allowed/denied counts
  - timeout counts
  - output truncation counts
- Plugin flow:
  - fetch/summarize/send latency
  - per-step error rate
  - dedup hit rate

## Verification Checklist

- [ ] Plugin module compiles and is loaded by server
- [ ] Unit/integration/e2e suites for plugin pass
- [ ] Scheduler idempotency is verified for same `instance+slot`
- [ ] Sandbox high-risk deny path and explicit-approval path are verified
- [ ] Incident response playbook dry-run completed
