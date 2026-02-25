# DoorX Project Template

Go backend project template built on Kratos v2 + uber/fx, with PostgreSQL (Atlas migrations) and Redis.

## Quick Start

```bash
# 克隆模板仓库
git clone https://github.com/xxx/project-template.git
cd project-template

# 生成新项目（交互式）
go run init.go myproject

# 或使用非交互模式
go run init.go -o /path/to/output -y myproject

# 进入新项目目录
cd myproject

# 启动开发环境
make dev-run
```

> **注意**: Go flag 包要求选项放在参数之前。正确顺序：`go run init.go [选项] <项目名>`

## Template Variables

生成项目时可配置以下变量：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| ProjectName | 项目名称 | myproject |
| ModuleName | Go module 名 | github.com/user/myproject |
| DBName | 数据库名 | myproject |
| DBUser | 数据库用户 | myproject |
| DBPassword | 数据库密码 | myproject |
| DBPort | 数据库端口 | 5432 |
| RedisPort | Redis 端口 | 6379 |
| BackendPort | 后端端口 | 8080 |
| FrontendPort | 前端端口 | 9001 |
| Description | 项目描述 | My awesome project |
| Year | 年份 | 2025 |
| Version | 版本 | 1.0.0 |

### 环境变量配置

可以使用环境变量设置默认值：

```bash
PROJECT_NAME=myapp \
MODULE_NAME=github.com/myuser/myapp \
DB_NAME=myapp \
go run init.go myapp -y
```

## Generated Project Features

- **User Management**: Complete user authentication and authorization system
- **Role-Based Access Control (RBAC)**: Casbin-based permissions management
- **Department & Post Management**: Organizational structure management
- **Dictionary Management**: System configuration and data dictionaries
- **File Management**: Support for multiple storage backends (local, S3, Aliyun OSS, Tencent COS)
- **Message System**: Internal messaging and notifications
- **Audit Logging**: Comprehensive operation logs and audit trails
- **Multi-tenancy Ready**: Tenant-aware architecture
- **Internationalization**: Built-in i18n support

## Development Commands

生成项目后可用的命令：

```bash
# 启动基础设施（PostgreSQL + Redis）
make infra-up

# 运行数据库迁移
make migrate-dev

# 初始化种子数据
make seed-dev

# 启动开发服务器
make dev-run

# 运行测试
make test

# 生成代码
make all
```

## Architecture

- **Framework**: Kratos v2
- **Dependency Injection**: uber/fx
- **Database**: PostgreSQL with GORM
- **Migrations**: Atlas
- **Cache**: Redis
- **Authentication**: JWT + Casbin RBAC
- **Storage**: Pluggable backends (local, S3, Aliyun, Tencent)
- **Tracing**: OpenTelemetry
- **Metrics**: Prometheus

## Project Structure

```
project-template/
├── init.go              # 模板生成工具
├── template/            # 项目模板目录
│   ├── cmd/             # 命令行入口
│   ├── internal/        # 内部代码
│   │   ├── biz/         # 业务逻辑
│   │   ├── data/        # 数据访问
│   │   └── server/      # 服务器
│   ├── api/             # API 定义
│   ├── pkg/             # 公共包
│   ├── docker/          # Docker 配置
│   ├── migrations/      # 数据库迁移
│   └── Makefile
├── .template-vars.yaml  # 模板变量配置
└── README.md            # 本文件
```

## License

MIT
