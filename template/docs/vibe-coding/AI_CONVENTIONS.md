# AI 编码工具统一约定

本文档定义了所有 AI 编码工具（Claude Code、Codex、OpenCode 等）在 DoorX 项目中应遵守的统一约定。

## 1. 提交信息规范 (Conventional Commits)

### 格式
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### 类型 (Type)
| 类型 | 用途 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `docs` | 文档更新 |
| `style` | 代码格式（不影响功能） |
| `refactor` | 代码重构 |
| `perf` | 性能优化 |
| `test` | 测试相关 |
| `chore` | 构建/工具/依赖更新 |
| `ci` | CI/CD 配置 |
| `build` | 构建系统 |

### 作用域 (Scope) - 可选但推荐
**后端:**
- `api` - API 定义和接口
- `service` - 服务层
- `biz` - 业务逻辑层
- `data` - 数据访问层
- `infra` - 基础设施层
- `db` - 数据库相关
- `docker` - Docker 配置

**前端:**
- `web` - 前端整体
- `ui` - UI 组件
- `api-client` - API 客户端

**通用:**
- `deps` - 依赖更新
- `config` - 配置文件
- `docs` - 文档

### 示例
```bash
feat(api): add user authentication endpoint
fix(db): resolve connection pool exhaustion issue
docs(readme): update deployment instructions
refactor(biz): simplify order processing logic
chore(deps): update go.mod dependencies
style(web): format with prettier
test(service): add unit tests for user service
ci(github): add automated release workflow
```

## 2. 代码规范

### Go (后端)
```bash
# 提交前必须运行
make fmt          # 格式化代码
go vet ./...      # 静态检查
```

- 使用 Uber FX 进行依赖注入
- 遵循分层架构：API → Service → Biz → Data → Infrastructure
- 接口定义在使用方，不在实现方
- 错误处理使用 `github.com/go-kratos/kratos/v2/errors`

### TypeScript/React (前端)
```bash
cd web && npm run lint    # 检查代码
```

- 遵循项目 ESLint 配置
- 使用 Prettier 格式化（单引号，自动组织 imports）
- React 组件使用默认导出

## 3. 代码生成工作流

### Protocol Buffers
修改 `.proto` 文件后：
```bash
make api    # 生成 Go 代码
```

### 数据库迁移
修改 GORM 模型后：
```bash
make migrate-diff    # 生成迁移文件
```

## 4. 禁止事项

- ❌ 提交二进制文件
- ❌ 提交 `.env` 文件或敏感信息
- ❌ 提交生成的代码（除非必要且无法自动生成）
- ❌ 提交未格式化的代码
- ❌ 破坏向后兼容的 API 修改（无适当版本控制）

## 5. 项目结构

```
projecttemplate/
├── api/              # Protocol Buffer 定义
├── cmd/              # 应用入口
├── internal/         # 内部实现
│   ├── biz/          # 业务逻辑层
│   ├── data/         # 数据访问层
│   ├── service/      # 服务层
│   └── infra/        # 基础设施层
├── web/              # React 前端
├── docs/             # 文档
├── Makefile          # 构建脚本
└── go.mod            # Go 依赖
```

## 6. 工具配置位置

| 工具 | 配置位置 |
|------|----------|
| Claude Code | `.claude/CLAUDE.md` |
| Codex | `.claude/codex.md` |
| OpenCode | `.opencode/opencode.md` |
| 通用约定 | `AI_CONVENTIONS.md` (本文档) |

## 7. 快速检查清单

提交代码前：
- [ ] 代码已格式化 (`make fmt`)
- [ ] 静态检查通过 (`go vet ./...`)
- [ ] 提交信息符合 Conventional Commits
- [ ] 未提交敏感信息
- [ ] 生成的代码已更新（如果修改了 proto 或模型）
