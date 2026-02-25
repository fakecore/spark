# JetStream DLQ Runbook

## Trigger
- `projecttemplate_jetstream_dlq_total` increases.
- Ingestor logs show `decode_json`, `validate_task`, or `max_deliver`.

## Immediate Actions
1. Identify DLQ reason distribution from metrics:
   - `component=change_ingestor`
   - `reason=decode_json|validate_task|max_deliver`
2. Inspect recent DLQ payload from `projecttemplate_dlq` stream.
3. Confirm upstream producer format (`ChangeTask`) and schema compatibility.

## Reason Playbook
1. `decode_json`:
   - Producer emitted invalid JSON.
   - Fix producer serialization and replay from source if needed.
2. `validate_task`:
   - Required field missing (`workspace_id`, `object_id`, `change_id`, etc.).
   - Fix producer validation before publish.
3. `max_deliver`:
   - Persistent processing failure.
   - Check hub DB health, unique constraints, and immutable conflict logs.

## Post-Recovery
1. Ensure new ingest succeeds and DLQ growth stops.
2. Requeue DLQ messages only after producer/consumer fix is verified.
3. Track `projecttemplate_materialize_rebuild_total{result="error"}` for downstream impact.
