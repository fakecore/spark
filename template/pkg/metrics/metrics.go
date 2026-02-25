package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	once sync.Once

	HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_http_requests_total",
		Help: "Total number of HTTP requests handled by the server (Kratos operation label).",
	}, []string{"operation", "code"})
	HTTPRequestDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "projecttemplate_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds (Kratos operation label).",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})

	GRPCRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_grpc_requests_total",
		Help: "Total number of gRPC requests handled by the server.",
	}, []string{"method", "code"})
	GRPCRequestDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "projecttemplate_grpc_request_duration_seconds",
		Help:    "gRPC request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method"})
	SyncRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_sync_requests_total",
		Help: "Total sync API requests handled by role/workspace/result.",
	}, []string{"method", "node_role", "workspace_id", "result"})
	SyncRequestDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "projecttemplate_sync_request_duration_seconds",
		Help:    "Sync API request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "node_role"})

	JobRunsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_job_runs_total",
		Help: "Total number of background job runs (best-effort; per-process).",
	}, []string{"job", "result"})
	JobRunDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "projecttemplate_job_run_duration_seconds",
		Help:    "Background job run latency in seconds (best-effort; per-process).",
		Buckets: prometheus.DefBuckets,
	}, []string{"job"})

	LeaseOpsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_hub_lease_ops_total",
		Help: "Hub lease operations (acquire/renew/status).",
	}, []string{"op", "result"})

	WSConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "projecttemplate_ws_connections",
		Help: "Number of active WebSocket connections.",
	})
	WSMessagesSentTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_ws_messages_sent_total",
		Help: "Total number of WebSocket messages sent (best-effort).",
	}, []string{"event_type"})
	WSDisconnectsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_ws_disconnects_total",
		Help: "Total number of websocket disconnects by reason.",
	}, []string{"reason"})
	WSAuthFailuresTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_ws_auth_failures_total",
		Help: "Total number of websocket auth failures by reason.",
	}, []string{"reason"})

	ExecTaskOpsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_exec_task_ops_total",
		Help: "Execution task operations (create/claim/report).",
	}, []string{"op", "result"})

	SecretsOpsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_secrets_ops_total",
		Help: "Secrets manager operations (encrypt/decrypt/rotate/reencrypt).",
	}, []string{"op", "result"})
	EdgeSyncOpsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_edge_sync_ops_total",
		Help: "Edge sync operation counts by op/result.",
	}, []string{"op", "result"})
	EdgeSyncDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "projecttemplate_edge_sync_duration_seconds",
		Help:    "Edge sync operation latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"op"})
	EdgePushBacklog = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "projecttemplate_edge_push_backlog",
		Help: "Current edge push backlog (changes with null order_id).",
	}, []string{"workspace_id"})
	MaterializeRebuildTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_materialize_rebuild_total",
		Help: "Materialize rebuild attempts by result.",
	}, []string{"result"})
	MaterializeRebuildDurationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "projecttemplate_materialize_rebuild_duration_seconds",
		Help:    "Materialize rebuild latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"result"})
	JetStreamDLQTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "projecttemplate_jetstream_dlq_total",
		Help: "JetStream DLQ published messages by component and reason.",
	}, []string{"component", "reason"})
)

// Init registers all DoorX metrics with the default Prometheus registry.
// Safe to call multiple times.
func Init() {
	once.Do(func() {
		prometheus.MustRegister(
			HTTPRequestsTotal,
			HTTPRequestDurationSeconds,
			GRPCRequestsTotal,
			GRPCRequestDurationSeconds,
			SyncRequestsTotal,
			SyncRequestDurationSeconds,
			JobRunsTotal,
			JobRunDurationSeconds,
			LeaseOpsTotal,
			WSConnections,
			WSMessagesSentTotal,
			WSDisconnectsTotal,
			WSAuthFailuresTotal,
			ExecTaskOpsTotal,
			SecretsOpsTotal,
			EdgeSyncOpsTotal,
			EdgeSyncDurationSeconds,
			EdgePushBacklog,
			MaterializeRebuildTotal,
			MaterializeRebuildDurationSeconds,
			JetStreamDLQTotal,
		)
	})
}
