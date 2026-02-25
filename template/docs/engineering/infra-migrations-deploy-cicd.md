# DoorX: Schema 迁移与部署（Dev/Test/Prod + CI/CD）

本文档回答这些问题：
- 本地数据库怎么初始化
- 新增表/字段/索引、修改字段定义怎么做
- “新增 key 值”（例如系统配置/字典初始数据）怎么做
- 测试环境/生产环境分别怎么部署
- CI/CD 用到哪些步骤，怎么跑 smoke 验证

## 1. 本地初始化（Dev）

### 1.1 启动基础设施

```bash
cd /Users/dylan/item/projecttemplate/projecttemplate
make infra-up
make infra-ps
```

默认端口与账号：
- Postgres: `127.0.0.1:5432`，用户/密码/库：`projecttemplate/projecttemplate/projecttemplate`
- Redis: `127.0.0.1:6379`
- NATS: `127.0.0.1:4222`（JetStream enabled），监控端口 `127.0.0.1:8222`

### 1.2 初始化/应用迁移（Atlas）

推荐显式设置 `DATABASE_URL`（CI 也是用这个）：

```bash
export DATABASE_URL='postgres://projecttemplate:projecttemplate@127.0.0.1:5432/projecttemplate?sslmode=disable'
make migrate-dev
```

说明：
- 空库：会直接 `atlas migrate apply`
- 非空库：会检查 pending migrations，有则 apply；无则退出（幂等）

### 1.3 启动服务与健康检查

```bash
make run
curl -sS http://127.0.0.1:9988/healthz
curl -sS http://127.0.0.1:9988/readyz
```

### 1.4 (可选) 本地测试用初始化数据

Schema 迁移默认只创建表结构，不会自动创建可登录的管理员账号。

本仓库提供一个最小化的 dev seed 工具（创建 `admin` 用户、`admin` 角色，并写入 `casbin_rule(ptype=g)` 绑定关系）：

```bash
make seed-dev
```

默认用户名/密码：`admin` / `admin123456`。

如果你希望“从干净库开始”，可以先（危险）：

```bash
make migrate-reset
make seed-dev
```

`/readyz` 会检查：
- Postgres ping
- Redis ping
- NATS 连接状态
- JetStream（开启时会做 `AccountInfo` 验证）

## 2. Schema 事实来源与目录约定

### 2.1 事实来源（Source of Truth）

- **表结构的事实来源**：`internal/data/dal/model/*.go`
- **迁移文件**：`migrations/*.sql`（Atlas 生成或手写补充）

典型工作流是：
1. 改 `model/*.go`
2. 生成迁移：`make migrate-diff NAME=...`
3. 应用迁移：`make migrate-dev`

### 2.2 不要把“非迁移文件”放在 `migrations/` 根目录

Atlas 会对迁移目录做 checksum 管理。任何不参与迁移的 SQL/草稿文件都应放到：
- `migrations/_scratch/`

（仓库已把 `desired_schema.sql` / `manual_alter.sql` 移到 `_scratch`）

## 3. 变更示例：新增表 / 新增字段 / 修改字段定义

以下示例都以 PostgreSQL 为主。

### 3.1 新增表（新增一个 Model）

1) 在 `internal/data/dal/model/` 新增一个文件，例如 `sys_feature_flag.go`：

```go
package model

const TableNameSysFeatureFlag = "sys_feature_flag"

type SysFeatureFlag struct {
	BaseModelNoSoftDelete
	Key    string `gorm:"size:128;not null;uniqueIndex;comment:开关Key" json:"key"`
	Value  string `gorm:"type:text;not null;comment:开关值" json:"value"`
	Status int32  `gorm:"type:smallint;not null;default:1;comment:状态 0禁用 1启用" json:"status"`
}

func (SysFeatureFlag) TableName() string { return TableNameSysFeatureFlag }
```

2) 把新 model 加入 `hack/gendb/gen.go` 的 `g.ApplyBasic(...)`（否则不会生成 query 代码）。

3) 生成迁移：

```bash
export DATABASE_URL='postgres://projecttemplate:projecttemplate@127.0.0.1:5432/projecttemplate?sslmode=disable'
make migrate-diff NAME=add_sys_feature_flag
```

4) 应用迁移：

```bash
make migrate-dev
```

### 3.2 新增字段（给已有表加列）

以 `internal/data/dal/model/sys_config.go` 为例，新增字段 `Group`：

```go
Group *string `gorm:"size:64;comment:配置分组" json:"group"`
```

然后：

```bash
export DATABASE_URL='postgres://projecttemplate:projecttemplate@127.0.0.1:5432/projecttemplate?sslmode=disable'
make migrate-diff NAME=add_sys_config_group
make migrate-dev
```

### 3.3 修改字段定义（类型/长度/默认值/可空性）

直接改 `model` 的 gorm tag，例如：
- `size`：`size:64` -> `size:128`
- 可空：`string` -> `*string`（或反过来）
- 类型：`type:smallint` -> `type:integer`
- 默认值：`default:1`
- 索引/唯一：`index` / `uniqueIndex`

然后用 `make migrate-diff` 生成 `ALTER TABLE ...` 迁移，再 `make migrate-dev` 应用。

#### 重要：非空列（NOT NULL）对已有数据的风险

如果表里已经有数据，直接加 `NOT NULL` 且无默认值可能会失败。推荐用“分步迁移”：
1) 先加 nullable 列
2) 回填数据（UPDATE）
3) 再改成 NOT NULL

这种场景可以用“手写迁移 SQL”（见下一节）。

## 4. “新增 key 值”（初始化数据/字典/系统配置）

这里的 “key” 常见有两类：
- `sys_config.key`（系统配置）
- `sys_dict_type.type_code` / `sys_dict_value.dict_code`（字典）

推荐把“初始化数据”做成**幂等**的 migration（可重复执行不报错），典型做法：
- 使用 `ON CONFLICT (...) DO NOTHING`
- 或先 `SELECT` 判断再插入

### 4.1 示例：新增一个系统配置 key（sys_config）

新建一个迁移文件（手写 SQL），例如：
- `migrations/20260207000000_seed_sys_config.sql`

内容示例：

```sql
-- Seed sys_config
INSERT INTO sys_config (created_by, created_at, name, key, value, kind, status, remark)
VALUES (
  0,
  floor(extract(epoch from now()) * 1000)::bigint,
  '最大上传大小(MB)',
  'upload.max_mb',
  '50',
  0,
  1,
  'seed by migration'
)
ON CONFLICT (key) DO NOTHING;
```

然后更新 atlas checksum 并应用：

```bash
export DATABASE_URL='postgres://projecttemplate:projecttemplate@127.0.0.1:5432/projecttemplate?sslmode=disable'
cd migrations && atlas migrate hash --env local
cd .. && make migrate-dev
```

说明：
- `atlas migrate diff` 主要用于 schema diff；“数据 seed”更适合手写 migration。
- 手写 migration 之后一定要 `atlas migrate hash`，否则会 checksum mismatch。

## 5. 测试环境与生产环境如何部署

这里给出两套“可落地”的推荐方式，先按简单可靠来。

### 5.1 测试环境（Staging/Test）推荐

目标：可快速拉起一套 DoorX（依赖+服务），方便 smoke/联调。

建议：
1) 依赖（PG/Redis/NATS）用独立服务或同机 docker（但请隔离 DB/账号/stream）
2) DoorX 用容器运行（推荐 `docker/docker-compose.yml`），镜像来源于 CI push
3) 迁移由 CI 或发布流程执行（推荐使用 `make migrate-prod` 指向 staging 的 `DATABASE_URL`）

典型流程（在测试机上，手动部署）：

```bash
cd /opt/projecttemplate-staging

# .env 参考：`docker/.env.example`
docker compose pull

# 注意：迁移建议在部署前完成（CI 里跑 make migrate-prod）
docker compose up -d

curl http://127.0.0.1:8080/readyz
```

### 5.2 生产环境（Prod）推荐

目标：可靠、可观测、可回滚，避免把 DB/NATS 跟业务容器绑死。

建议：
- Postgres/Redis/NATS 使用独立服务（云托管或自建集群）
- DoorX 只作为应用进程（容器或 systemd）
- 迁移流程与发布解耦：上线前先跑 `make migrate-prod`（必须提供 `DATABASE_URL`）

生产迁移：

```bash
export DATABASE_URL='postgres://user:pass@prod-host:5432/projecttemplate?sslmode=require'
make migrate-prod
```

生产配置建议：
- 不要把明文密钥写进 repo 的 yaml
- 用环境变量覆盖（例如 config.yaml 对应字段 / config.yaml 对应字段 等）

## 6. CI/CD 怎么跑（GitHub Actions 为主，Gitea 可选推送）

### 6.1 GitHub Actions（权威）

文件：`.github/workflows/ci.yml`

关键点：
- `test` job：`make fmt` / `make vet` / `make all` / `make build`
- `smoke` job：
  - 起 postgres/redis（actions services）
  - `docker run` 起 nats（JetStream enabled）
  - 设置 `DATABASE_URL`
  - `make migrate-dev`
  - `go run cmd/server/main.go ...`
  - 轮询 `GET /readyz` 直到成功或超时

这能保证：依赖 + 迁移 + 服务启动 + readiness 全链路都可重复验证。

### 6.2 Gitea Actions（可选推送到“本地测试 Registry”）

文件：`.gitea/workflows/ci.yml`

如果要开启镜像推送，需要在 Gitea 的 secrets 里配置：
- `PROJECT_TEMPLATE_REGISTRY`
- `PROJECT_TEMPLATE_REGISTRY_USERNAME`
- `PROJECT_TEMPLATE_REGISTRY_PASSWORD`

未配置 secrets 时会自动跳过 login/push，不会因为 registry 缺失导致失败。

#### ash_nyx（192.168.31.230）上启用 Gitea Container Registry 的注意事项

当前 `ash_nyx` 已做两件事（已验证）：
- Caddy 已将 `https://192.168.31.230/` 反代到 Gitea（原 9998 端口）
- Gitea 已启用 `[packages]`（因此 registry 的 `/v2/` 可用）

你可以用下面命令验证（任意一台机器上都行，但可能需要跳过证书校验）：

```bash
curl -k -sS -D- https://192.168.31.230/v2/ -o /dev/null | sed -n '1,20p'
```

会看到类似：
- `HTTP/2 401`
- `docker-distribution-api-version: registry/2.0`
- `www-authenticate: Bearer realm="https://192.168.31.230/v2/token"...`

**Docker 证书信任（关键，否则 CI/机器上 push/pull 会报 x509）**

因为这里的 HTTPS 证书由 homelab 的 Step-CA 签发，Docker daemon 默认不信任，需要在每台需要 push/pull 的机器上安装 CA：

```bash
# 1) 拉取 CA 证书（在 ash_nyx 上执行）
docker exec caddy sh -c 'cat /ca/certs/root_ca.crt' > /tmp/ash-homelab-root-ca.crt

# 2) 安装到 Docker 的 registry 信任目录（需要 sudo）
sudo mkdir -p /etc/docker/certs.d/192.168.31.230
sudo cp /tmp/ash-homelab-root-ca.crt /etc/docker/certs.d/192.168.31.230/ca.crt

# 3) 重启 docker（不同发行版命令可能不同）
sudo systemctl restart docker || sudo service docker restart
```

然后测试：

```bash
docker login 192.168.31.230
```

认证建议使用：
- 用户名：你的 Gitea 用户名（例如 `ash`）
- 密码：Gitea Personal Access Token（需要包含 packages 读写权限）

#### （推荐）自动部署到 Staging

建议让 CI 在发布 tag（例如 `v1.0.1`）后自动：
1) push 镜像到你的 registry
2) 对 staging DB 执行迁移（推荐在 staging 机器上执行，避免 CI runner 必须直连 DB）
3) SSH 到 staging 机器执行 `docker compose pull && docker compose up -d`
4) SSH 上本机 `curl http://127.0.0.1:8080/readyz` 做 smoke

推荐 secrets（Gitea）：
- `STAGING_SSH_HOST`
- `STAGING_SSH_USER`
- `STAGING_SSH_KEY`（私钥内容）
- `STAGING_DATABASE_URL`（例如：`postgres://projecttemplate:projecttemplate123@127.0.0.1:5432/projecttemplate?sslmode=disable`，在 staging 机本地可达即可）
- `STAGING_DATABASE_SOURCE`（DoorX 容器连接 DB 的 DSN，例如：`host=host.docker.internal user=projecttemplate password=projecttemplate123 dbname=projecttemplate port=5432 sslmode=disable TimeZone=Asia/Shanghai`）
- `PROJECT_TEMPLATE_REGISTRY`
- `PROJECT_TEMPLATE_REGISTRY_USERNAME`
- `PROJECT_TEMPLATE_REGISTRY_PASSWORD`

## 6.3 To-C/离线部署：下载新版本后“自动迁移 + 升级”

如果你的目标用户是单机部署（例如家用 NAS/homelab），推荐发 release 包：
- `dist/<product>-<version>-prod.tar.gz`

release 包里包含：
- `docker-compose.yml`
- `images/backend.tar` / `images/frontend.tar`（离线镜像）
- `migrations/*.sql` + `migrations/atlas.sum`
- `manage.sh`（支持一键升级）

典型升级流程（用户侧，不需要 git，不需要 build）：

```bash
tar -xzf projecttemplate-<version>-prod.tar.gz
cd projecttemplate-<version>-prod

# 只在 migrate/upgrade 时需要 DATABASE_URL
export DATABASE_URL='postgres://user:pass@127.0.0.1:5432/projecttemplate?sslmode=disable'

./manage.sh upgrade
```

为什么不推荐“后端进程启动时自动迁移”：
- 多副本/并发启动时容易出现竞争（多个实例同时跑 migrate）
- 迁移失败会导致服务启动失败，回滚/可观测性也更差

因此这里把迁移放在“升级流程”中显式执行（`manage.sh upgrade` / CI deploy job）。


## 7. 常用命令速查

```bash
# infra
make infra-up
make infra-down

# migrate
export DATABASE_URL='postgres://projecttemplate:projecttemplate@127.0.0.1:5432/projecttemplate?sslmode=disable'
make migrate-dev
make migrate-diff NAME=add_xxx
make migrate-status

# run
make run
curl -sS http://127.0.0.1:9988/readyz
```

## 8. NATS 模式切换（external/embedded/disabled）

DoorX 支持三种 NATS 模式，方便 To-C 降低部署复杂度，同时也能随时切换到 hub/服务器外部 NATS：

- `external`（默认）：连接到 `data.nats.url`
- `embedded`：进程内启动 NATS Server（可启用 JetStream 持久化）
- `disabled`：完全禁用 NATS（依赖消息队列/通知的功能需要同时关闭或降级）

### 8.0 为什么单机也值得用 NATS JetStream

即使只有单机、没有 hub，NATS JetStream 仍然能作为“本机内部队列/通知/任务分发”的基础设施：

- 异步化：把耗时或易失败的工作从 HTTP 请求链路中拆出来
- 可重试：失败任务可以重试，不阻塞主流程
- 解耦：业务逻辑不需要直接耦合到“通知/同步/任务”的实现细节
- 可恢复：JetStream 持久化后，进程重启不丢任务（需要 `store_dir` 持久化）

典型用途：
- 变更通知（例如配置变更 -> 推送在线客户端/刷新缓存）
- 后台任务（导入导出、报表生成、大文件处理）
- 对外动作解耦（写库成功后异步执行 webhook/同步等，可重试）

### 8.1 配置项

配置文件（`docker/backend/config/config.yaml*`）：

```yaml
data:
  nats:
    mode: external # external|embedded|disabled
    url: nats://127.0.0.1:4222
    jetstream:
      enabled: true
      domain: ""
    # mode=embedded 且 jetstream.enabled=true 时生效
    store_dir: ""
```

环境变量覆盖：
- `config.yaml 对应字段=external|embedded|disabled`
- `config.yaml 对应字段=/path/to/store`（仅 embedded+JetStream 用）

### 8.2 “随时切换到 hub”的含义

当你从 `embedded` 切到 `external`：
- 只需要把 `config.yaml 对应字段=external` 并设置 `config.yaml 对应字段=...`，重启即可
- embedded JetStream 的历史消息不会自动迁移到外部 NATS（通常可以接受，从切换点开始走 external 即可）

## 9. Redis 模式（external/auto/memory）

DoorX 的 token/session KV 默认走 Redis，但支持在 To-C 单机部署中降级为内存存储以降低依赖。

- `external`：强依赖 Redis，无法连接则启动失败
- `auto`（默认）：尽力连接 Redis，连不上或运行中 Redis 出错时，降级到内存存储
- `memory`：强制使用内存存储（单进程有效，重启会丢）

配置文件：

```yaml
data:
  redis:
    mode: auto # external|auto|memory
    addr: 127.0.0.1:6379
    password: ""
    database: 1
```

环境变量覆盖：
- `config.yaml 对应字段=external|auto|memory`
