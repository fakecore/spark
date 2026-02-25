# Sync Recovery Runbook

## Scope
- Edge push backlog grows (`order_id IS NULL`).
- Edge cannot converge with hub after transient network failures.
- Pull path stalls after cursor inconsistency.

## Fast Checks
1. Check `/readyz` for `postgres`, `nats`, `jetstream`.
2. Check `/metrics`:
   - `projecttemplate_edge_push_backlog`
   - `projecttemplate_edge_sync_ops_total`
   - `projecttemplate_sync_requests_total`
3. Check edge logs for:
   - `push object failed`
   - `pull object failed`
   - `sync token mismatch`

## Recovery Procedure
1. Validate auth path:
   - edge has `x-sync-token` configured for workspace.
   - hub token map/default matches edge token.
2. Validate edge cursor monotonicity:
   - `sync_cursors.after_order_id` should not regress.
   - if regressed manually, set to last known good order_id and restart edge worker.
3. Force reconcile:
   - keep edge running; reconcile ticker should mark dirty objects.
   - if needed, restart edge process once to trigger full discovery path.
4. Validate convergence:
   - hub and edge `changes` rows for target object have same max `order_id`.
   - `object_state` can be rebuilt without missing parent errors.

## Verification SQL
```sql
-- edge backlog
SELECT workspace_id, count(*) AS pending
FROM changes
WHERE order_id IS NULL
GROUP BY workspace_id;

-- cursor drift
SELECT workspace_id, object_id, after_order_id, updated_at
FROM sync_cursors
ORDER BY updated_at DESC
LIMIT 50;
```
