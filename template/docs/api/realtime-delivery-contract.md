# Realtime Delivery Contract (WS)

## Semantics
1. WS channel is notification-only.
2. Delivery is at-least-once and best-effort.
3. Duplicate events are allowed; client should deduplicate by `change_id`/event identity.

## Auth And Boundary
1. Endpoint: `/api/v1/ws?workspace_id=<ws>&token=<jwt>`
2. Missing/invalid token is rejected.
3. Connection only subscribes to `projecttemplate.events.<workspace_id>.>` events.

## Reconnect Strategy
1. Client should use exponential backoff reconnect.
2. After reconnect, client must call existing pull APIs to fill gaps:
   - `GetHeads`
   - `GetChangesAfter`
3. WS is not a source of truth; state convergence is based on pull + replay.
