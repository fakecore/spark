# DoorX Backend Simplification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Simplify DoorX from AI tools platform to standard Go backend scaffold, removing Change DAG, NATS, AI business logic while keeping Kratos framework, PostgreSQL, Redis, and system modules.

**Architecture:** Progressive deletion approach - remove unused code systematically while ensuring compilation after each step. Keep standard three-layer architecture (Interface/Service/Biz/Data) with system modules only.

**Tech Stack:** Go 1.21+, Kratos v2, PostgreSQL 16, Redis 7, GORM, Atlas migrations

---

## Prerequisites

**Required reading:**
- `docs/plans/2026-02-26-simplify-backend-design.md` - Full design context
- `internal/conf/conf.proto` - Current configuration structure
- `internal/data/data.go` - Data layer initialization

**Environment:**
- Go 1.21 or later installed
- Docker and docker-compose available
- Make available

---

## Task 1: Delete NATS and Change DAG Business Logic

**Files:**
- Delete: `internal/biz/action/` (entire directory)
- Delete: `internal/biz/execution/` (entire directory)
- Delete: `internal/biz/changedag/` (entire directory)

**Step 1: Delete biz/action directory**

```bash
rm -rf internal/biz/action
```

**Step 2: Delete biz/execution directory**

```bash
rm -rf internal/biz/execution
```

**Step 3: Delete biz/changedag directory**

```bash
rm -rf internal/biz/changedag
```

**Step 4: Verify deletion**

```bash
ls internal/biz/
```

Expected output: Only `biz.go` and `system/` remain

**Step 5: Commit**

```bash
git add -A
git commit -m "refactor: delete NATS and Change DAG biz layer

- Remove internal/biz/action/
- Remove internal/biz/execution/
- Remove internal/biz/changedag/"
```

---

## Task 2: Delete Workflow and Plugin Systems

**Files:**
- Delete: `internal/workflow/` (entire directory)
- Delete: `internal/plugin/` (entire directory)

**Step 1: Delete workflow directory**

```bash
rm -rf internal/workflow
```

**Step 2: Delete plugin directory**

```bash
rm -rf internal/plugin
```

**Step 3: Verify deletion**

```bash
ls internal/
```

Expected output: No `workflow/` or `plugin/` directories

**Step 4: Commit**

```bash
git add -A
git commit -m "refactor: delete workflow and plugin systems

- Remove internal/workflow/
- Remove internal/plugin/"
```

---

## Task 3: Delete Service Layer - Action and Sync

**Files:**
- Delete: `internal/service/action/` (entire directory)
- Delete: `internal/service/sync/` (entire directory)

**Step 1: Delete service/action directory**

```bash
rm -rf internal/service/action
```

**Step 2: Delete service/sync directory**

```bash
rm -rf internal/service/sync
```

**Step 3: Verify deletion**

```bash
ls internal/service/
```

Expected output: Only `system/` remains (and potentially other non-removed services)

**Step 4: Commit**

```bash
git add -A
git commit -m "refactor: delete action and sync services

- Remove internal/service/action/
- Remove internal/service/sync/"
```

---

## Task 4: Delete Infrastructure and Endpoints

**Files:**
- Delete: `internal/infrastructure/` (entire directory)
- Delete: `internal/server/ws_endpoints.go`
- Delete: `internal/server/hub_lease_endpoints.go`
- Delete: `internal/server/infra_sync_endpoints.go`
- Delete: `internal/server/infra_exec_endpoints.go`

**Step 1: Delete infrastructure directory**

```bash
rm -rf internal/infrastructure
```

**Step 2: Delete WebSocket endpoints**

```bash
rm -f internal/server/ws_endpoints.go internal/server/ws_endpoints_test.go
```

**Step 3: Delete hub lease endpoints**

```bash
rm -f internal/server/hub_lease_endpoints.go
```

**Step 4: Delete infra sync endpoints**

```bash
rm -f internal/server/infra_sync_endpoints.go
```

**Step 5: Delete infra exec endpoints**

```bash
rm -f internal/server/infra_exec_endpoints.go
```

**Step 6: Verify deletion**

```bash
ls internal/server/*.go
```

Expected output: No `ws_endpoints.go`, `hub_lease_endpoints.go`, `infra_sync_endpoints.go`, or `infra_exec_endpoints.go`

**Step 7: Commit**

```bash
git add -A
git commit -m "refactor: delete infrastructure and special endpoints

- Remove internal/infrastructure/
- Remove WebSocket, hub lease, sync, and exec endpoints"
```

---

## Task 5: Delete Proto Definitions - Action and Sync

**Files:**
- Delete: `api/action/v1/action.proto`
- Delete: `api/sync/v1/change_sync.proto`
- Delete: `api/proto/plugin/v1/` (entire directory)

**Step 1: Delete action proto**

```bash
rm -f api/action/v1/action.proto
rm -f api/action/v1/action.pb.go
```

**Step 2: Delete sync proto**

```bash
rm -f api/sync/v1/change_sync.proto
rm -f api/sync/v1/change_sync.pb.go
```

**Step 3: Delete plugin proto directory**

```bash
rm -rf api/proto/plugin
```

**Step 4: Verify deletion**

```bash
ls api/ && ls api/proto/
```

Expected output: No `action/` or `sync/` in `api/`, no `plugin/` in `api/proto/`

**Step 5: Commit**

```bash
git add -A
git commit -m "refactor: delete action, sync, and plugin proto definitions

- Remove api/action/v1/
- Remove api/sync/v1/
- Remove api/proto/plugin/"
```

---

## Task 6: Delete Data Layer Repositories

**Files:**
- Delete: `internal/data/repository/changedag/` (entire directory)
- Delete: `internal/data/repository/execution/` (entire directory)

**Step 1: Delete changedag repository**

```bash
rm -rf internal/data/repository/changedag
```

**Step 2: Delete execution repository**

```bash
rm -rf internal/data/repository/execution
```

**Step 3: Verify deletion**

```bash
ls internal/data/repository/
```

Expected output: Only `system/` remains

**Step 4: Commit**

```bash
git add -A
git commit -m "refactor: delete changedag and execution repositories

- Remove internal/data/repository/changedag/
- Remove internal/data/repository/execution/"
```

---

## Task 7: Delete Test Utilities

**Files:**
- Delete: `internal/testutil/natstest/` (entire directory)

**Step 1: Delete natstest directory**

```bash
rm -rf internal/testutil/natstest
```

**Step 2: Verify deletion**

```bash
ls internal/testutil/
```

Expected output: No `natstest/` directory

**Step 3: Commit**

```bash
git add -A
git commit -m "refactor: delete NATS test utilities

- Remove internal/testutil/natstest/"
```

---

## Task 8: Modify internal/data/data.go - Remove NATS Dependencies

**Files:**
- Modify: `internal/data/data.go`

**Step 1: Read current data.go**

```bash
cat internal/data/data.go
```

**Step 2: Remove NATS related imports and fields**

Edit `internal/data/data.go`:
- Remove NATS imports
- Remove NATS struct fields from Data struct
- Remove NATS initialization in NewData()

**Step 3: Verify edit**

```bash
grep -i "nats" internal/data/data.go
```

Expected output: No results (or only in comments if保留注释)

**Step 4: Attempt build to identify other dependencies**

```bash
go build ./internal/data/...
```

Expected: May have errors due to other files still referencing deleted code

**Step 5: Commit**

```bash
git add internal/data/data.go
git commit -m "refactor: remove NATS dependencies from data.go"
```

---

## Task 9: Modify internal/conf/conf.proto - Remove NATS and Sync

**Files:**
- Modify: `internal/conf/conf.proto`
- Regenerate: `internal/conf/conf.pb.go`

**Step 1: Read current conf.proto**

```bash
cat internal/conf/conf.proto
```

**Step 2: Remove NATS and Sync message definitions**

Edit `internal/conf/conf.proto`, remove:
```protobuf
// Remove these message definitions
message NATS { ... }
message Sync { ... }

// Remove from Data message
message Data {
  Database database = 1;
  Redis redis = 2;
  NATS nats = 3;      // DELETE THIS LINE
  Sync sync = 4;      // DELETE THIS LINE
  // Keep rest...
}
```

**Step 3: Regenerate protobuf**

```bash
make proto
```

Expected: Success, no errors

**Step 4: Verify generated file**

```bash
grep -i "nats\|sync" internal/conf/conf.pb.go | grep -v "^//" | head -5
```

Expected output: No NATS/Sync struct references (except in comments)

**Step 5: Commit**

```bash
git add internal/conf/conf.proto internal/conf/conf.pb.go
git commit -m "refactor: remove NATS and Sync from configuration"
```

---

## Task 10: Modify internal/server/http.go - Remove Special Endpoints

**Files:**
- Modify: `internal/server/http.go`

**Step 1: Read current http.go**

```bash
cat internal/server/http.go
```

**Step 2: Remove endpoint registrations for deleted services**

Edit `internal/server/http.go`:
- Remove `wsServer` registration
- Remove `hubLeaseServer` registration
- Remove `infraSyncServer` registration
- Remove `infraExecServer` registration

**Step 3: Verify build**

```bash
go build ./internal/server/...
```

Expected: May have errors from removed dependencies

**Step 4: Commit**

```bash
git add internal/server/http.go
git commit -m "refactor: remove deleted service endpoints from HTTP server"
```

---

## Task 11: Modify internal/server/grpc.go - Remove Special Services

**Files:**
- Modify: `internal/server/grpc.go`

**Step 1: Read current grpc.go**

```bash
cat internal/server/grpc.go
```

**Step 2: Remove gRPC service registrations**

Edit `internal/server/grpc.go`:
- Remove deleted service registrations (action, sync, plugin, etc.)

**Step 3: Verify build**

```bash
go build ./internal/server/...
```

Expected: May have errors from removed dependencies

**Step 4: Commit**

```bash
git add internal/server/grpc.go
git commit -m "refactor: remove deleted services from gRPC server"
```

---

## Task 12: Modify cmd/server/main.go - Remove Deleted Components

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: Read current main.go**

```bash
cat cmd/server/main.go
```

**Step 2: Remove initialization for deleted components**

Edit `cmd/server/main.go`:
- Remove NATS JetStream initialization
- Remove ChangeStore initialization
- Remove Sync Service initialization
- Remove WebSocket initialization
- Remove Plugin System initialization
- Remove deleted service wire registrations

**Step 3: Verify build**

```bash
go build ./cmd/server
```

Expected: May have errors from removed dependencies

**Step 4: Commit**

```bash
git add cmd/server/main.go
git/server/main.go
git commit -m "refactor: remove deleted component initialization from main"
```

---

## Task 13: Modify internal/biz/biz.go - Remove Deleted UseCases

**Files:**
- Modify: `internal/biz/biz.go`

**Step 1: Read current biz.go**

```bash
cat internal/biz/biz.go
```

**Step 2: Remove deleted use case registrations**

Edit `internal/biz/biz.go`:
- Remove Action use case references
- Remove Execution use case references
- Remove ChangeDAG use case references
- Remove Sync use case references

**Step 3: Verify build**

```bash
go build ./internal/biz/...
```

Expected: May have errors from removed dependencies

**Step 4: Commit**

```bash
git add internal/biz/biz.go
git commit -m "refactor: remove deleted use cases from biz layer"
```

---

## Task 14: Clean up go.mod Dependencies

**Files:**
- Modify: `go.mod` and `go.sum`

**Step 1: Run go mod tidy**

```bash
go mod tidy
```

Expected: Removes unused dependencies, updates go.sum

**Step 2: Check for remaining NATS dependencies**

```bash
grep "nats" go.mod
```

Expected: No NATS-related imports (or only if still used elsewhere)

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "refactor: clean up unused dependencies with go mod tidy"
```

---

## Task 15: Update docker-compose.yml - Remove NATS Service

**Files:**
- Modify: `docker/docker-compose.yml`

**Step 1: Read current docker-compose.yml**

```bash
cat docker/docker-compose.yml
```

**Step 2: Remove NATS service**

Edit `docker/docker-compose.yml`:
- Remove entire `nats:` service block

**Step 3: Verify services**

```bash
grep "^  [a-z]" docker/docker-compose.yml
```

Expected output: Only `postgres:` and `redis:` services

**Step 4: Commit**

```bash
git add docker/docker-compose.yml
git commit -m "refactor: remove NATS from docker-compose"
```

---

## Task 16: Update config.yaml - Remove NATS and Sync Configuration

**Files:**
- Modify: `docker/backend/config/config.yaml`
- Modify: `docker/backend/config/config.yaml.example`

**Step 1: Edit config.yaml**

Remove `nats:` and `sync:` sections from `docker/backend/config/config.yaml`

**Step 2: Edit config.yaml.example**

Remove `nats:` and `sync:` sections from `docker/backend/config/config.yaml.example`

**Step 3: Verify config files**

```bash
grep -E "^data:|^  nats:|^  sync:" docker/backend/config/config.yaml
```

Expected output: Only shows `data:`, no `nats:` or `sync:` subsections

**Step 4: Commit**

```bash
git add docker/backend/config/
git commit -m "refactor: remove NATS and Sync from configuration files"
```

---

## Task 17: Update Makefile - Remove NATS Targets

**Files:**
- Modify: `Makefile`

**Step 1: Read current Makefile**

```bash
cat Makefile
```

**Step 2: Remove NATS-related targets**

Edit `Makefile`:
- Remove `nats-setup` target
- Remove `nats-streams-create` target
- Remove any other NATS-specific targets

**Step 3: Verify Makefile syntax**

```bash
make -n help  # dry-run, check syntax
```

Expected: No errors

**Step 4: Commit**

```bash
git add Makefile
git commit -m "refactor: remove NATS targets from Makefile"
```

---

## Task 18: Verify Compilation

**Files:**
- Build: `cmd/server/main.go`

**Step 1: Clean build**

```bash
go clean -cache
go build ./cmd/server
```

Expected: Successful build, binary created

**Step 2: Verify binary exists**

```bash
ls -lh ./server
```

Expected output: Binary file exists with reasonable size

**Step 3: Test basic compilation**

```bash
go build ./...
```

Expected: All packages compile successfully

**Step 4: If compilation fails, fix errors**

For each compilation error:
1. Read the error message carefully
2. Identify the file and line with error
3. Remove or fix the problematic code (likely references to deleted types)
4. Re-run build
5. Repeat until build succeeds

**Step 5: Commit any fixes**

```bash
git add -A
git commit -m "fix: resolve compilation errors after simplification"
```

---

## Task 19: Run Unit Tests

**Files:**
- Test: All remaining test files

**Step 1: Run tests with coverage**

```bash
go test -v -cover ./...
```

Expected: Tests pass (some may fail if they reference deleted code)

**Step 2: Fix failing tests**

For each failing test:
1. Read the test file
2. Delete or update the test if it references deleted components
3. Re-run tests

**Step 3: Commit test fixes**

```bash
git add -A
git commit -m "test: update tests after simplification"
```

---

## Task 20: Update Documentation

**Files:**
- Modify: `README.md`
- Delete: `docs/design/architecture.md` (or update)
- Delete: `docs/design/implementation-roadmap.md` (or update)

**Step 1: Update README.md**

Edit `README.md` to reflect simplified project:
- Remove references to AI features
- Remove references to NATS
- Update project description
- Update quick start instructions (no NATS)

**Step 2: Decide on design docs**

Either delete or update architecture docs to match simplified version

**Step 3: Commit documentation updates**

```bash
git add README.md docs/
git commit -m "docs: update documentation for simplified backend"
```

---

## Task 21: Integration Test - Full Stack

**Files:**
- Test: Full application stack

**Step 1: Start infrastructure**

```bash
make dev-up
```

Expected: PostgreSQL and Redis containers start successfully

**Step 2: Run database migrations**

```bash
make migrate-up
```

Expected: Migrations apply successfully (only system tables)

**Step 3: Start server**

```bash
make run
```

Expected: Server starts without errors

**Step 4: Check health endpoint**

```bash
curl -s http://127.0.0.1:9988/healthz
```

Expected: `{"status":"ok"}` or similar

**Step 5: Test system API**

```bash
curl -s http://127.0.0.1:9988/api/v1/system/config/list
```

Expected: Valid JSON response

**Step 6: Stop infrastructure**

```bash
make dev-down
```

**Step 7: Commit any fixes**

```bash
git add -A
git commit -m "fix: resolve integration test issues"
```

---

## Task 22: Final Verification and Cleanup

**Files:**
- All project files

**Step 1: Verify no leftover references**

```bash
grep -r "ChangeDAG\|ChangeStore\|NATS\|JetStream" internal/ --include="*.go" | grep -v "^//" | head -10
```

Expected: No results (or only in comments)

**Step 2: Check for deleted proto imports**

```bash
grep -r "action/v1\|sync/v1\|plugin/v1" internal/ --include="*.go"
```

Expected: No results

**Step 3: Final build test**

```bash
make build
```

Expected: Clean build

**Step 4: Final test run**

```bash
make test
```

Expected: All tests pass

**Step 5: Create summary commit**

```bash
git add -A
git commit -m "chore: final cleanup after backend simplification

Completed simplification from AI tools platform to standard Go backend:
- Removed Change DAG sync system
- Removed NATS JetStream message queue
- Removed AI business logic (Action/Chat)
- Removed WebSocket real-time communication
- Removed plugin system and workflow engine

Retained:
- Kratos v2 framework (HTTP/gRPC)
- PostgreSQL + GORM + Atlas migrations
- Redis for caching
- System modules (User/Role/Menu/Permission)
- Standard three-layer architecture"
```

---

## Verification Checklist

After completing all tasks, verify:

- [ ] `make build` compiles successfully
- [ ] `make dev-up` starts only PostgreSQL and Redis
- [ ] `make migrate-up` applies migrations
- [ ] `make run` starts server without errors
- [ ] `curl http://127.0.0.1:9988/healthz` returns OK
- [ ] System APIs (user, role, menu) are accessible
- [ ] `make test` passes all tests
- [ ] No references to deleted code in `internal/`
- [ ] Documentation is updated

---

## Notes

- Each task includes commit points for easy rollback
- Build verification steps are included throughout
- Tasks are ordered to minimize cascading errors
- Some tasks may need iteration if compilation errors persist
- Keep all commits atomic (one logical change per commit)
