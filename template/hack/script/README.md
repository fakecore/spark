# Docker Build Script

这个脚本用于构建和打包Docker镜像，支持生产环境和开发环境。

## 使用方法

### 直接使用脚本

#### 构建生产环境
```bash
./hack/script/build-docker.sh prod
```

#### 构建开发环境
```bash
./hack/script/build-docker.sh dev
```

#### 错误用法
```bash
# 错误：必须指定环境参数
./hack/script/build-docker.sh

# 错误：环境参数不正确
./hack/script/build-docker.sh test
```

### 使用Make命令（推荐）

#### 构建生产环境
```bash
make docker-build-prod
# 或者
make docker-build ENV=prod
```

#### 构建开发环境
```bash
make docker-build-dev
# 或者
make docker-build ENV=dev
```

#### 查看Docker构建帮助
```bash
make docker-help
```

#### 清理构建产物
```bash
make docker-clean
```

#### 单独构建（支持环境参数）
```bash
make buildall ENV=prod  # 构建生产环境
make buildall ENV=dev   # 构建开发环境
make buildall           # 默认构建生产环境
```

## 功能说明

1. **环境支持**: 支持 `prod` 和 `dev` 两种环境（用于打包标识与默认 `.env` 约定）
   - 运行方案统一：使用同一份 `docker/docker-compose.yml` + 环境变量区分环境

2. **自动构建**: 执行 `make buildall` 构建所有组件
   - 支持环境参数：`make buildall ENV=prod` 或 `make buildall ENV=dev`
   - 默认使用生产环境配置

3. **镜像打包**: 导出 backend 和 frontend Docker 镜像

4. **配置打包**: 复制 `docker/backend/config/` 下的配置文件

5. **管理脚本**: 包含 `manage.sh` 脚本用于部署后的服务管理

6. **文件重命名**: docker-compose文件在部署包中统一重命名为 `docker-compose.yml`

## 配置文件

脚本会根据环境选择性地复制配置文件到部署包中：

- `casbin_model.conf` - 权限模型配置（所有环境都使用）
- `config.yaml` - 统一配置文件（通过环境变量覆盖区分环境）

**注意**：部署包中的配置文件会被重命名为 `config.yaml`，这样应用程序可以直接使用默认配置文件名。docker-compose文件中的command配置也相应更新为使用 `config.yaml`。

## 输出文件

脚本会在 `./dist/` 目录下创建以下文件：
- `${PRODUCT_NAME}-${VERSION}-${ENVIRONMENT}/` - 部署包目录
  - `docker-compose.yml` - 统一命名的docker-compose文件
  - `backend/config/` - 配置文件目录
    - `config.yaml` - 统一配置文件（可通过 `.env` / 环境变量覆盖）
    - `casbin_model.conf` - 权限模型配置
  - `images/` - Docker镜像文件
  - `manage.sh` - 服务管理脚本
- `${PRODUCT_NAME}-${VERSION}-${ENVIRONMENT}.tar.gz` - 压缩包

## 部署后管理

使用 `manage.sh` 脚本管理服务：

```bash
# 加载镜像
./manage.sh load

# 运行数据库迁移（推荐升级时先跑；需要提供 DATABASE_URL）
export DATABASE_URL='postgres://user:pass@127.0.0.1:5432/projecttemplate?sslmode=disable'
./manage.sh migrate

# 启动服务
./manage.sh start

# 停止服务
./manage.sh stop

# 重启服务
./manage.sh restart

# Smoke 检查（轮询 /readyz）
./manage.sh smoke

# 一键离线升级（load -> migrate -> start -> smoke）
export DATABASE_URL='postgres://user:pass@127.0.0.1:5432/projecttemplate?sslmode=disable'
./manage.sh upgrade
```

## 环境差异

### 生产环境 (prod)
- 使用生产docker-compose配置
- 默认只包含应用（backend/frontend）；数据库/redis/nats 由外部提供或由你自行编排
- 升级推荐走 release 包（含 images + migrations），用 `./manage.sh upgrade` 完成自动迁移与重启

### 开发环境 (dev)
- 使用开发docker-compose配置
- 启用调试端口
- 更详细的日志输出
- 本地存储配置
