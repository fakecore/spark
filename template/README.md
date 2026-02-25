# Spark

Go (Kratos) + React 19 + TanStack + shadcn/ui 全栈脚手架

## 技术栈

### 后端
- **框架**: Kratos v2
- **数据库**: PostgreSQL + GORM
- **缓存**: Redis
- **认证**: JWT + Casbin RBAC
- **迁移**: Atlas

### 前端
- **框架**: React 19 + TypeScript
- **构建**: Vite 5
- **UI**: shadcn/ui + Tailwind CSS v4
- **数据**: TanStack Query + TanStack Router
- **状态**: Zustand

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/fakecore/spark.git my-app
cd my-app
```

### 2. 启动基础设施

```bash
make infra-up
```

### 3. 运行数据库迁移

```bash
export DATABASE_URL='postgres://projecttemplate:projecttemplate@127.0.0.1:5432/projecttemplate?sslmode=disable'
make migrate-dev
```

### 4. 启动后端

```bash
make run
```

### 5. 启动前端

```bash
cd web
pnpm install
pnpm dev
```

访问 http://localhost:5173

## 常用命令

```bash
# 后端
make run              # 启动后端
make build            # 编译
make test             # 运行测试
make migrate-dev      # 运行迁移

# 前端
cd web
pnpm dev              # 开发服务器
pnpm build            # 构建
pnpm lint             # 代码检查
```

## 项目结构

```
├── cmd/server/         # 后端入口
├── internal/           # 私有代码
│   ├── service/        # 业务逻辑
│   └── data/           # 数据访问
├── pkg/                # 公共代码
├── api/                # API 定义
├── migrations/         # 数据库迁移
├── web/                # 前端代码
│   ├── src/
│   │   ├── pages/      # 页面
│   │   ├── components/ # 组件
│   │   └── lib/        # 工具
│   └── package.json
├── docker/             # Docker 配置
├── Makefile            # 常用命令
└── README.md
```

## 使用此模板

### 方式一：一键初始化（推荐）

```bash
curl -fsSL https://raw.githubusercontent.com/fakecore/spark/main/scripts/setup.sh | bash -s my-app
```

### 方式二：手动克隆 + 初始化

```bash
git clone https://github.com/fakecore/spark.git my-app
cd my-app
./scripts/init.sh my-app
```

### 开始开发

```bash
cd my-app
vim docker/backend/config/config.yaml  # 修改配置
make infra-up                           # 启动基础设施
make migrate-dev                        # 初始化数据库
make run                                # 启动后端
cd web && pnpm install && pnpm dev      # 启动前端
```

## License

MIT
