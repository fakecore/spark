# Plugin Implementation Checklist

Last updated: 2026-02-19

Purpose: persistent, session-independent execution checklist for plugin platform rollout.

Status legend:
- `pending`: not started
- `in_progress`: currently being worked on
- `blocked`: waiting on dependency/decision
- `done`: completed and verified

## Phase 0 - Rule Freeze

1) `[pending]` Kickoff freeze matrix
- Agent: `explore`
- Task: Re-read `docs/design/llm-master-agent-architecture.md` and `docs/design/plugin-platform-capability-spec.md`; produce one-page MUST-rule matrix.
- DoD: capability names, risk levels, token semantics, consent flow have no ambiguous wording.

2) `[pending]` Policy precedence and conflict resolution
- Agent: `oracle`
- Task: Finalize rule order `hard-deny > org > workspace > instance > consent > default`; deny-overrides-allow; scope specificity; TTL precedence.
- DoD: explicit decision table approved.

3) `[pending]` Token model freeze
- Agent: `oracle`
- Task: Freeze `ttl` config vs `exp` claim, run-scope vs per-call thresholds by risk, run-end invalidation, in-flight handling, revocation propagation.
- DoD: normative section has zero contradictions.

4) `[pending]` Consent lifecycle freeze
- Agent: `oracle`
- Task: Freeze transitions (`pending_consent/granted/active/revoked/expired`) and `allow_once/allow_until` semantics.
- DoD: state transition table plus examples.

5) `[pending]` Inter-plugin delegation freeze
- Agent: `oracle`
- Task: Freeze Action-first constraints: default deny, export/require matching, max depth, no privilege passthrough, derived token intersection.
- DoD: delegation validation algorithm documented.

## Phase 1 - Contracts and Data Model

6) `[pending]` Admin API contracts (`api/proto/plugin/v1`)
- Agent: `deep`
- Task: Draft install/enable/disable/permissions/consent/revoke/list contracts.
- DoD: proto passes lint/generation and error model is complete.

7) `[pending]` Runtime API contracts
- Agent: `deep`
- Task: Draft Handshake/Health/Invoke/OnEvent/Configure/Shutdown and EventEnvelope schema.
- DoD: proto generated and request/response schemas frozen.

8) `[pending]` Error code catalog
- Agent: `writing`
- Task: Define API errors (PERMISSION_DENIED, CAPABILITY_REQUIRES_CONSENT, TOKEN_EXPIRED, etc.), retryability, and UI behavior.
- DoD: error catalog linked from design index.

9) `[pending]` DB schema and migration design
- Agent: `deep`
- Task: Design `plugin_registry_snapshots`, `plugin_instances`, `plugin_grant_policies`, `plugin_invocation_audit`, delegation records.
- DoD: migration plan includes rollback and indexes.

## Phase 2 - Foundation Skeleton

10) `[pending]` Package scaffolding and wiring points
- Agent: `quick`
- Task: Create `internal/plugin/{core,registry,manager,policy,gateway,runtime,token}` with interface-only stubs and fx module hooks.
- DoD: project builds with stubs.

## Phase 3 - Core Runtime

11) `[pending]` Registry pipeline
- Agent: `deep`
- Task: Implement discover/verify/compile/activate snapshot flow with atomic swap and last-known-good rollback.
- DoD: bad manifest never replaces active snapshot.

12) `[pending]` Policy engine implementation
- Agent: `deep`
- Task: Implement deterministic policy decision service with explainable trace.
- DoD: matrix unit tests cover all branches.

13) `[pending]` Token service implementation
- Agent: `deep`
- Task: Implement mint/validate/revoke with `run_id` binding, derived token support, denylist/version checks, run-end force invalidation.
- DoD: security tests pass replay/reuse/revocation scenarios.

14) `[pending]` Capability gateway implementation
- Agent: `deep`
- Task: Implement enforcement for `action.invoke`, `plugin.invoke`, `llm.chat.invoke`, storage capabilities (authz/quota/audit/errors).
- DoD: unauthorized calls fail closed.

15) `[pending]` Action catalog integration
- Agent: `explore+deep`
- Task: Register `exports.actions[]` into Action Catalog and route via existing action entrypoint.
- DoD: plugin action discoverable and invokable through gateway.

## Phase 3.5 - Sandbox Baseline (Raw Local First)

16) `[pending]` Sandbox abstraction contract
- Agent: `deep`
- Task: Define `SandboxDriver` interface and `SandboxProfile` model (`low/medium/high/critical`), with capability-to-profile mapping.
- DoD: architecture contract is frozen and referenced by gateway/executor.

17) `[pending]` Raw local driver implementation (first)
- Agent: `deep`
- Task: Implement `raw-local` driver as initial backend (host process execution + strict audit hooks), without Docker/VM dependency.
- DoD: all sandboxed execution paths can run through `raw-local` backend in local development.

18) `[pending]` Local policy enforcement and guardrails
- Agent: `deep`
- Task: Enforce profile-based limits in `raw-local` mode (command allowlist, cwd scope, timeout, env filtering, output caps).
- DoD: high-risk capabilities are blocked or require explicit policy gates even in local mode.

Note: Execute this phase before Workflow integration.

## Phase 4 - Workflow and E2E Example

19) `[pending]` Workflow/scheduler integration
- Agent: `deep`
- Task: Integrate plugin runs with scheduler (08:00 cron), slot idempotency lock, retry/DLQ behavior.
- DoD: same `plugin_instance+slot` never executes twice.

20) `[pending]` Reference plugin: news assistant
- Agent: `deep`
- Task: Build demo flow (fetch -> summarize -> dedup -> send) using gateway-only capabilities.
- DoD: daily end-to-end run works and audit trace is complete.

## Phase 5 - Security, Testing, Operations

21) `[pending]` Threat-model review
- Agent: `oracle`
- Task: Review leakage, privilege escalation, delegation abuse, consent bypass.
- DoD: mitigation checklist closed or tracked as blockers.

22) `[pending]` Test suite layers
- Agent: `deep`
- Task: Add unit (policy/token), integration (gateway+registry), e2e (consent/revoke/run-end invalidation/inter-plugin depth).
- DoD: CI gate includes suites.

23) `[pending]` Ops runbook and docs
- Agent: `writing`
- Task: Produce onboarding, risk prompts, revoke/denylist incident response, observability dashboard guidance.
- DoD: docs linked in `docs/design/README.md`.

24) `[pending]` Release rollout plan
- Agent: `oracle`
- Task: Define internal alpha -> allowlist beta -> GA with kill switches and rollback criteria.
- DoD: go/no-go checklist signed.

---

## New Session Start Prompt

Use this in a new session:

"Load `docs/design/plugin-implementation-checklist.md` and `docs/design/plugin-implementation-checklist.yaml`, set all tasks to pending in todo list, then start from item 1 and execute phase by phase."
