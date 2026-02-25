# Plugin Rule Freeze Matrix

Last updated: 2026-02-19

This document freezes rule semantics for checklist items 1-5.

## 1) MUST-rule Matrix

| Domain | MUST Rule | Notes |
|---|---|---|
| Capability naming | Canonical names only: `action.invoke`, `plugin.invoke`, `llm.chat.invoke`, `storage.read`, `storage.write`, `storage.delete`, `storage.list` | Unknown capability fails closed |
| Risk levels | `low`, `medium`, `high`, `critical` | `high/critical` require explicit approval policy |
| Consent states | `pending_consent`, `granted`, `active`, `revoked`, `expired` | `revoked/expired` always deny |
| Token model | `exp` claim is enforcement source, `ttl` config is minting input | `exp` mismatch with policy window => deny |
| Delegation | default deny + intersection-only derived capability set | no privilege amplification |

## 2) Policy Precedence Decision Table

Precedence order (highest first):

1. `hard-deny`
2. `org`
3. `workspace`
4. `instance`
5. `consent`
6. `default`

Conflict resolution:

- Same scope: `deny` overrides `allow`
- More specific scope overrides less specific scope only when higher precedence does not already deny
- TTL precedence: shorter valid window wins for allow decisions

## 3) Token Freeze (normative)

- Mint input uses `ttl`; validation enforces `exp`
- Token must bind to `run_id`
- Per-call validation checks:
  - signature valid
  - not expired
  - not denylisted/revoked
  - requested capability in token claims
- Run-end invalidation revokes all tokens for that `run_id`
- In-flight handling: validation failure after revocation fails next guarded capability call

## 4) Consent Lifecycle Freeze

Allowed transitions:

- `pending_consent -> granted`
- `granted -> active`
- `active -> revoked`
- `active -> expired`
- `granted -> revoked`

Terminal states:

- `revoked` terminal unless explicit new consent workflow resets to `pending_consent`
- `expired` requires renewal path (`pending_consent -> granted -> active`)

Semantics:

- `allow_once`: applies to one invocation only and is consumed immediately
- `allow_until`: time-bounded allow, transitions to `expired` after deadline

## 5) Inter-plugin Delegation Freeze

Rules:

- Action-first default deny
- `exports.actions[]` and `requires[]` must match explicit contract
- Max delegation depth: 3
- No privilege passthrough: delegated call cannot gain capabilities absent in caller token
- Derived token capability set is strict intersection(parent, requested)

Validation algorithm:

1. Validate parent token and run binding
2. Check depth budget (`depth < 3`)
3. Confirm target action is exported by target plugin
4. Compute capability intersection
5. Deny if intersection empty
6. Mint derived token and execute delegated call
