# Plugin DB Schema and Migration Plan

Last updated: 2026-02-19

## Tables

### `plugin_registry_snapshots`

- `id` (uuid, pk)
- `workspace_id` (varchar, indexed)
- `version` (bigint)
- `manifest_bundle` (jsonb)
- `status` (varchar: active/failed/rollback)
- `created_at` (bigint)
- `updated_at` (bigint)

Indexes:

- `(workspace_id, version desc)`
- partial index for active snapshot (`status='active'`)

### `plugin_instances`

- `id` (uuid, pk)
- `workspace_id` (varchar, indexed)
- `plugin_name` (varchar)
- `plugin_version` (varchar)
- `enabled` (boolean)
- `consent_state` (varchar)
- `permissions` (jsonb)
- `created_at`, `updated_at` (bigint)

Indexes:

- unique `(workspace_id, plugin_name)`
- `(workspace_id, enabled)`

### `plugin_grant_policies`

- `id` (uuid, pk)
- `workspace_id` (varchar)
- `scope` (varchar: org/workspace/instance/consent/default)
- `effect` (varchar: allow/deny)
- `capability` (varchar)
- `ttl_seconds` (bigint nullable)
- `reason` (text)
- `created_at`, `updated_at` (bigint)

Indexes:

- `(workspace_id, scope, capability)`

### `plugin_invocation_audit`

- `id` (uuid, pk)
- `workspace_id` (varchar)
- `plugin_instance_id` (uuid)
- `run_id` (varchar)
- `capability` (varchar)
- `allowed` (boolean)
- `policy_scope` (varchar)
- `reason` (text)
- `duration_ms` (bigint)
- `created_at` (bigint)

Indexes:

- `(workspace_id, plugin_instance_id, created_at desc)`
- `(run_id, created_at asc)`

### `plugin_delegation_records`

- `id` (uuid, pk)
- `workspace_id` (varchar)
- `parent_run_id` (varchar)
- `child_run_id` (varchar)
- `from_plugin_instance_id` (uuid)
- `to_plugin_instance_id` (uuid)
- `depth` (int)
- `capability_intersection` (jsonb)
- `created_at` (bigint)

Indexes:

- `(workspace_id, parent_run_id)`
- `(workspace_id, child_run_id)`

## Migration Sequence

1. Create new tables without foreign key hard-coupling to avoid bootstrap lockstep.
2. Backfill optional metadata for existing plugin instances (if any).
3. Add indexes after bulk backfill.
4. Turn on write-path feature flag.
5. Enable read-path feature flag after validation.

## Rollback Plan

- Keep old read path available behind feature flag.
- On rollback:
  1. disable write/read feature flags,
  2. keep new tables (no destructive rollback in hot path),
  3. archive partial records and re-run migration in next release.
