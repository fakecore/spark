# Config And Secrets Policy

## Priority
Runtime config/secret resolution order:
1. Environment variables
2. Secrets manager values
3. Static config file

## Sync Token Rules
1. `default_token` cannot be mixed with `default_read_token/default_write_token`.
2. For the same workspace, do not set both:
   - `workspace_tokens`
   - `workspace_read_tokens` / `workspace_write_tokens`
3. Edge role requires:
   - `sync.edge_id`
   - `sync.edge_workspace_id`
   - `sync.hub_grpc_addr`
   - `data.nats.mode=external` and `data.nats.url`

## Logging Policy
1. Never print raw DSN password/token in logs.
2. Redact auth-related values in startup logs.
3. Use metrics for failure counting instead of logging sensitive payloads.
