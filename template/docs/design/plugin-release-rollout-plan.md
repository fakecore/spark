# Plugin Release Rollout Plan

Last updated: 2026-02-19

## Objective

Roll out plugin platform safely from internal alpha to GA with explicit gates, kill switches, and rollback criteria.

## Stages

### Stage 0: Internal Alpha

Entry criteria:

- Sandbox raw-local guardrails enabled
- Scheduler slot idempotency and retry/DLQ verified
- Reference plugin run passes end-to-end in test

Controls:

- Allowlist: internal team only
- High/critical profile requires explicit approval
- Daily manual review of audit traces

Exit criteria:

- 7 consecutive days without security incident
- DLQ rate < 1% per day
- No unauthorized command execution

### Stage 1: Allowlist Beta

Entry criteria:

- Stage 0 complete
- Ops runbook validated with incident drill

Controls:

- Workspace allowlist only
- Feature flag for scheduler-run plugins
- Kill switch tested weekly

Exit criteria:

- 14 consecutive days stable
- Retry recovery success rate >= 99%
- No unresolved high-severity security findings

### Stage 2: General Availability (GA)

Entry criteria:

- Stage 1 complete
- Threat model checklist closed or accepted with owners

Controls:

- Progressive rollout by region/workspace cohorts
- Runtime kill switch remains available
- Observability SLO dashboard active

GA signoff:

- Product owner
- Security owner
- Operations owner

## Kill Switches

- Disable scheduler trigger for plugin runs
- Force deny high/critical capability executions
- Disable plugin module load at service startup

## Rollback Criteria

Immediate rollback when any is true:

- Confirmed privilege escalation or consent bypass
- Repeated unauthorized command execution
- Persistent run duplication for same `instance+slot`
- DLQ backlog growing for 2 consecutive days with no recovery

## Rollback Procedure

1. Flip kill switches (scheduler off + capability deny)
2. Revoke active approvals and block new elevated runs
3. Preserve and export audit artifacts
4. Return traffic to previous stable release
5. Open incident and postmortem timeline

## Go/No-Go Checklist

- [ ] Security review complete
- [ ] Ops runbook complete and linked
- [ ] Unit/integration/e2e suites green
- [ ] Rollback rehearsed on staging
- [ ] On-call owners assigned
