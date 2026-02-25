# Infra Alerts Baseline

## Recommended Alerts
1. `DLQGrowth`
- Expression: `increase(projecttemplate_jetstream_dlq_total[5m]) > 0`
- Severity: warning

2. `EdgePushBacklogHigh`
- Expression: `max_over_time(projecttemplate_edge_push_backlog[10m]) > 100`
- Severity: warning

3. `MaterializeErrorRateHigh`
- Expression: `sum(increase(projecttemplate_materialize_rebuild_total{result="error"}[5m])) > 0`
- Severity: warning

4. `WSDisconnectSpike`
- Expression: `increase(projecttemplate_ws_disconnects_total[5m]) > 50`
- Severity: warning

5. `SyncPermissionDeniedSpike`
- Expression: `sum(increase(projecttemplate_sync_requests_total{result="PermissionDenied"}[5m])) > 20`
- Severity: warning

## Dashboard Sections
1. Sync API latency and error code heatmap.
2. Edge push backlog + retry/timeout counters.
3. Materializer rebuild duration and error ratio.
4. DLQ reasons by component.
5. WS active connections and disconnect reasons.
