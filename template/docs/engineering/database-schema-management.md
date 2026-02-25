# 数据库 Schema 管理规范

## 架构总览

采用 **Code-First + Atlas** 架构：大多数**业务实体表**以 Go struct 作为事实来源，Atlas 负责生成迁移与校验。

### 例外（允许手写 SQL / Atlas HCL 作为权威）

以下能力很难（或不建议）仅用 Go struct/GORM tag 表达，允许以 `migrations/*.sql`（或 Atlas HCL）为事实来源：

- **表分区**：`PARTITION BY RANGE/HASH`、分区创建/挂载/删除、分区索引策略
- **高级索引**：部分索引（`WHERE ...`）、表达式索引、`GIN/GIST`、`INCLUDE`、`CONCURRENTLY`
- **触发器/函数/生成列**：Outbox 发布触发、审计、computed/generated column
- **视图/物化视图 / RLS / 扩展**：如 `pgcrypto`、Row-Level Security policy
- **在线迁移过程**：大表加非空列、分批 backfill、双写切换等（“过程”本身不是 struct 能表达的）

同步内核/基础设施表（例如 ChangeStore/Outbox/分区相关表）通常会用到以上能力，因此也属于例外：Go struct 仅作为 ORM 映射，不再宣称“唯一事实来源”。

```
┌─────────────────────────────────────────────────────────────────┐
│                        开发工作流                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   internal/data/dal/model/*.go    (手写实体，唯一事实来源)         │
│              │                                                  │
│              ├──────────────────┬───────────────────┐           │
│              ▼                  ▼                   ▼           │
│     Atlas migrate diff    GORM Gen generate    直接使用          │
│              │                  │                   │           │
│              ▼                  ▼                   ▼           │
│     migrations/*.sql    query/*.gen.go      Repository 层       │
│              │                  │                   │           │
│              ▼                  └───────────────────┤           │
│       PostgreSQL DB ◄───────────────────────────────┘           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 目录结构

```
internal/data/dal/
├── model/                    # 手写实体定义 (Source of Truth)
│   ├── base.go              # 公共字段
│   ├── sys_user.go          # 系统用户
│   ├── sys_role.go          # 系统角色
│   ├── sys_dept.go          # 部门
│   └── ...
└── query/                    # GORM Gen 生成的查询接口
    ├── gen.go
    ├── sys_user.gen.go
    └── ...

hack/gendb/
├── gen.go                    # GORM Gen 配置 (生成 Query 接口)
└── querier/                  # 自定义 SQL 查询接口 (可选)
    └── sys_dept.go

migrations/                   # Atlas 生成的迁移文件
├── atlas.hcl                 # Atlas 配置
├── 20240101000000_init.sql
└── atlas.sum
```

## 实体定义规范

### 基础模型

```go
// internal/data/dal/model/base.go
package model

import "gorm.io/plugin/soft_delete"

// BaseModel 所有需要软删除和审计的实体继承此结构
type BaseModel struct {
    ID        int64                 `gorm:"primaryKey;autoIncrement" json:"id"`
    CreatedBy *int64                `gorm:"comment:创建者" json:"created_by"`
    UpdatedBy *int64                `gorm:"comment:更新者" json:"updated_by"`
    DeletedBy *int64                `gorm:"comment:删除者" json:"deleted_by"`
    CreatedAt int64                 `gorm:"autoCreateTime:milli;comment:创建时间" json:"created_at"`
    UpdatedAt int64                 `gorm:"autoUpdateTime:milli;comment:更新时间" json:"updated_at"`
    DeletedAt soft_delete.DeletedAt `gorm:"softDelete:milli;comment:删除时间" json:"deleted_at"`
}

// BaseModelNoSoftDelete 不需要软删除的实体继承此结构
type BaseModelNoSoftDelete struct {
    ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
    CreatedBy *int64 `gorm:"comment:创建者" json:"created_by"`
    UpdatedBy *int64 `gorm:"comment:更新者" json:"updated_by"`
    CreatedAt int64  `gorm:"autoCreateTime:milli;comment:创建时间" json:"created_at"`
    UpdatedAt int64  `gorm:"autoUpdateTime:milli;comment:更新时间" json:"updated_at"`
}

// BaseModelCreateOnly 只有创建时间的实体继承此结构
type BaseModelCreateOnly struct {
    ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
    CreatedBy *int64 `gorm:"comment:创建者" json:"created_by"`
    CreatedAt int64  `gorm:"autoCreateTime:milli;comment:创建时间" json:"created_at"`
}
```

### 实体定义示例

```go
// internal/data/dal/model/sys_user.go
package model

const TableNameSysUser = "sys_user"

type SysUser struct {
    BaseModel
    Name     string  `gorm:"size:64;not null;comment:用户名" json:"name"`
    Nickname string  `gorm:"size:64;not null;comment:用户昵称" json:"nickname"`
    Email    string  `gorm:"size:128;not null;comment:邮箱" json:"email"`
    Password string  `gorm:"size:128;not null;comment:密码" json:"password"`
    Status   int32   `gorm:"type:smallint;not null;default:1;comment:状态 0禁用 1正常" json:"status"`
    Sex      int32   `gorm:"type:smallint;not null;default:0;comment:性别" json:"sex"`
    IsAdmin  int32   `gorm:"type:smallint;not null;default:0" json:"is_admin"`
    Avatar   *string `gorm:"size:512" json:"avatar"`
    Remark   *string `gorm:"size:512" json:"remark"`
    DeptID   int64   `gorm:"not null;index" json:"dept_id"`

    // 关系定义
    Roles []SysRole `gorm:"many2many:sys_user_role;joinForeignKey:user_id;joinReferences:role_id" json:"roles"`
    Dept  SysDept   `gorm:"foreignKey:DeptID" json:"dept"`
    Posts []SysPost `gorm:"many2many:sys_user_post;joinForeignKey:user_id;joinReferences:post_id" json:"posts"`
}

func (SysUser) TableName() string {
    return TableNameSysUser
}
```

### GORM Tag 速查表 (PostgreSQL)

| 需求 | Tag | 示例 |
|------|-----|------|
| 字符串长度 | `size:n` | `gorm:"size:64"` |
| 精确类型 | `type:xxx` | `gorm:"type:varchar(64)"` |
| SMALLINT | `type:smallint` | `gorm:"type:smallint"` |
| DECIMAL | `type:decimal(m,n)` | `gorm:"type:decimal(10,2)"` |
| TEXT | `type:text` | `gorm:"type:text"` |
| JSON/JSONB | `type:jsonb` | `gorm:"type:jsonb"` |
| 非空 | `not null` | `gorm:"not null"` |
| 默认值 | `default:值` | `gorm:"default:1"` |
| 注释 | `comment:说明` | `gorm:"comment:状态"` |
| 索引 | `index` | `gorm:"index"` |
| 唯一索引 | `uniqueIndex` | `gorm:"uniqueIndex"` |
| 日期类型 | `type:date` | `gorm:"type:date"` |

### Go 类型与 PostgreSQL 类型映射

| Go 类型 | PostgreSQL 默认 | 精确控制 |
|---------|----------------|----------|
| `string` | varchar(256) | `size:64` 或 `type:varchar(64)` |
| `int32` | integer | `type:smallint` |
| `int64` | bigint | - |
| `float64` | double precision | `type:decimal(10,2)` |
| `bool` | boolean | - |
| `*string` | varchar NULL | 可空字段用指针 |
| `time.Time` | timestamptz | `type:date` / `type:timestamp` |

## gen.go 配置

```go
// hack/gendb/gen.go
package main

import (
    "projecttemplate/internal/data/dal/model"
    "gorm.io/gen"
)

func main() {
    g := gen.NewGenerator(gen.Config{
        OutPath:      "internal/data/dal/query",
        ModelPkgPath: "projecttemplate/internal/data/dal/model",
        Mode:         gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
    })

    // 直接使用手写的 model，不再从数据库生成
    g.ApplyBasic(
        model.SysUser{},
        model.SysRole{},
        model.SysDept{},
        model.SysPost{},
        // ... 添加所有需要的 model
    )

    // 自定义 SQL 查询接口 (可选)
    // g.ApplyInterface(func(querier.SysDeptQuerier) {}, model.SysDept{})

    g.Execute()
}
```

## Atlas 配置

### atlas.hcl

位置: `migrations/atlas.hcl`

```hcl
data "external_schema" "gorm" {
  program = [
    "go", "run", "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "../internal/data/dal/model",
    "--dialect", "postgres",
  ]
}

env "local" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/15/projecttemplate?search_path=public"
  url = "postgres://user:pass@localhost:5432/projecttemplate?sslmode=disable"

  migration {
    dir    = "file://."
    format = atlas
  }
}

env "dev" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/15/projecttemplate?search_path=public"
  url = getenv("DATABASE_URL")

  migration {
    dir = "file://."
  }
}
```

**配置说明:**
- `external_schema.gorm`: 使用 Atlas GORM Provider 从 model 文件加载 schema
- `src`: 期望的 schema 状态（从 model 生成）
- `dev`: 开发数据库（用于计算 diff，可以是本地 Docker）
- `url`: 目标数据库（实际应用迁移的数据库）
- `migration.dir`: 迁移文件存放目录

## 日常工作流（推荐）

### 方案：智能迁移（兼容空库/已有数据）

```bash
# 开发环境 - 自动检测并同步
make migrate-dev

# 修改 model 后，生成迁移文件
make migrate-diff NAME=add_user_phone

# 生产环境 - 安全应用
make migrate-prod

# 查看状态
make migrate-status

# 重置数据库（危险！清空所有数据）
make migrate-reset
```

### 手动步骤（传统方式）

#### 新增表

```bash
# 1. 创建 model 文件
vim internal/data/dal/model/sys_new_table.go
```

```go
package model

const TableNameSysNewTable = "sys_new_table"

type SysNewTable struct {
    BaseModel
    Name   string `gorm:"size:64;not null" json:"name"`
    Status int32  `gorm:"type:smallint;default:1" json:"status"`
}

func (SysNewTable) TableName() string {
    return TableNameSysNewTable
}
```

```bash
# 2. 更新 gen.go，添加新 model
vim hack/gendb/gen.go

# 在 ApplyBasic 中添加：
# model.SysNewTable{}

# 3. 生成迁移（Atlas 自动 diff）
cd migrations && atlas migrate diff create_sys_new_table --env local

# 4. 审查生成的 SQL
cat 20240205_create_sys_new_table.sql

# 5. 应用迁移
atlas migrate apply --env local

# 6. 重新生成 Query 接口
cd .. && go run hack/gendb/gen.go
```

### 修改字段（Update）

```bash
# 1. 修改 model 文件
vim internal/data/dal/model/sys_user.go
```

```go
// 修改前
Name string `gorm:"size:64;not null" json:"name"`

// 修改后 - 扩大字段长度
Name string `gorm:"size:128;not null" json:"name"`  // 64 -> 128
```

```bash
# 2. 生成迁移（Atlas 自动检测变更）
cd migrations && atlas migrate diff update_user_name_length --env local

# 3. 审查 SQL（Atlas 自动生成 ALTER TABLE）
cat 20240205_update_user_name_length.sql
# 输出: ALTER TABLE sys_user ALTER COLUMN name TYPE varchar(128);

# 4. 应用迁移
atlas migrate apply --env local

# 5. 不需要重新生成 query（字段类型变更不影响接口）
```

### 添加字段

```go
// internal/data/dal/model/sys_user.go
type SysUser struct {
    BaseModel
    Name     string  `gorm:"size:64;not null" json:"name"`
    Email    string  `gorm:"size:128;not null" json:"email"`
    Avatar   *string `gorm:"size:512" json:"avatar"`        // 原有
    Phone    *string `gorm:"size:32" json:"phone"`          // 新增字段
}
```

```bash
cd migrations && atlas migrate diff add_user_phone --env local
atlas migrate apply --env local
cd .. && go run hack/gendb/gen.go  # 新增字段需要重新生成 query
```

### 删除字段

```go
// 直接删除字段定义
type SysUser struct {
    BaseModel
    Name     string  `gorm:"size:64;not null" json:"name"`
    Email    string  `gorm:"size:128;not null" json:"email"`
    // Phone    *string `gorm:"size:32" json:"phone"`  // 删除此行
}
```

```bash
cd migrations && atlas migrate diff drop_user_phone --env local
atlas migrate apply --env local
cd .. && go run hack/gendb/gen.go
```

## 查询使用方式

### 生成的 Query 接口 (推荐)

```go
// 类型安全，编译时检查
user, err := query.SysUser.WithContext(ctx).
    Where(query.SysUser.ID.Eq(id)).
    First()

users, err := query.SysUser.WithContext(ctx).
    Where(query.SysUser.Status.Eq(1)).
    Order(query.SysUser.CreatedAt.Desc()).
    Find()
```

### 原生 SQL (复杂报表)

```go
// Repository 层直接使用
var stats DailyStats
err := r.db.WithContext(ctx).Raw(`
    SELECT DATE(TO_TIMESTAMP(created_at/1000)) as date, COUNT(*) as count
    FROM sys_user
    WHERE created_at BETWEEN $1 AND $2
    GROUP BY date
`, startTime, endTime).Scan(&stats).Error
```

## 初始化设置

### 安装依赖

```bash
# 安装 Atlas CLI
curl -sSf https://atlasgo.sh | sh

# 安装 Atlas GORM Provider
go get ariga.io/atlas-provider-gorm

# 安装 PostgreSQL driver
go get gorm.io/driver/postgres
```

### 初始化迁移

```bash
# 1. 确保 migrations 目录存在
mkdir -p migrations

# 2. 创建初始迁移 (将当前 model 作为基准)
cd migrations && atlas migrate diff init --env local

# 3. 查看生成的迁移文件
ls
# 输出: 20240205120000_init.sql  atlas.sum

# 4. 应用到数据库
atlas migrate apply --env local

# 5. 验证迁移状态
atlas migrate status --env local
```

## Atlas 命令速查

| 命令 | 说明 |
|------|------|
| `atlas migrate diff <name> --env local` | 生成新迁移文件 |
| `atlas migrate apply --env local` | 应用待执行的迁移 |
| `atlas migrate status --env local` | 查看迁移状态 |
| `atlas migrate lint --env local` | 检查迁移文件 |
| `atlas schema inspect --env local` | 查看数据库当前 schema |

## 常见问题与解决方案

### 1. 数据库中已有表，无法应用迁移

**错误**: `pq: relation "xxx" already exists`

**原因**: Atlas 生成的迁移是 `CREATE TABLE`，但数据库中已有部分表。

**解决方案**: 使用 GORM 直接执行 ALTER 语句：

```go
package main

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "postgres://user:pass@host:5432/db?sslmode=disable"
	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	// 修改字段长度
	db.Exec("ALTER TABLE sys_user ALTER COLUMN password TYPE varchar(255)")
	db.Exec("ALTER TABLE sys_dept ALTER COLUMN ancestors TYPE varchar(512)")

	// 添加字段
	db.Exec("ALTER TABLE sys_oper_log ADD COLUMN IF NOT EXISTS created_by bigint")
}
```

### 2. 生成 ALTER 迁移而非 CREATE TABLE

当数据库已有数据时，需要生成 `ALTER TABLE` 而非 `CREATE TABLE`：

```bash
# 1. 导出当前 model 的期望 schema
go run -mod=mod ariga.io/atlas-provider-gorm load \
  --path ./internal/data/dal/model \
  --dialect postgres > desired_schema.sql

# 2. 对比并生成 diff（需要 Atlas Pro 或使用 schema apply）
atlas schema apply \
  --url "postgres://user:pass@host/db?sslmode=disable" \
  --to "file://desired_schema.sql" \
  --dev-url "docker://postgres/15/db?search_path=public"
```

### 3. 迁移状态混乱

**检查状态**:
```bash
atlas migrate status --env local
```

**重置迁移状态**（谨慎操作）:
```bash
# 直接操作 atlas_schema_revisions 表
psql $DATABASE_URL -c "TRUNCATE atlas_schema_revisions;"
```

### 4. 常用 PostgreSQL ALTER 语句

```sql
-- 修改字段长度
ALTER TABLE table_name ALTER COLUMN column_name TYPE varchar(new_size);

-- 添加字段
ALTER TABLE table_name ADD COLUMN IF NOT EXISTS column_name bigint;

-- 添加字段（带默认值）
ALTER TABLE table_name ADD COLUMN column_name varchar(100) DEFAULT '';

-- 修改字段可空性
ALTER TABLE table_name ALTER COLUMN column_name SET NOT NULL;
ALTER TABLE table_name ALTER COLUMN column_name DROP NOT NULL;

-- 删除字段
ALTER TABLE table_name DROP COLUMN IF EXISTS column_name;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_name ON table_name(column_name);

-- 添加唯一索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_name ON table_name(column_name);
```

## 当前 Model 列表

| Model | 表名 | 软删除 | 说明 |
|-------|------|--------|------|
| SysUser | sys_user | ✓ | 用户表 |
| SysRole | sys_role | ✗ | 角色表 |
| SysDept | sys_dept | ✓ | 部门表 |
| SysPost | sys_post | ✓ | 岗位表 |
| SysMenu | sys_menu | ✗ | 菜单表 |
| SysConfig | sys_config | ✗ | 配置表 |
| SysDictType | sys_dict_type | ✗ | 字典类型表 |
| SysDictValue | sys_dict_value | ✓ | 字典值表 |
| SysFile | sys_file | ✓ | 文件表 |
| SysLoginLog | sys_login_log | ✗ | 登录日志 |
| SysOperLog | sys_oper_log | ✗ | 操作日志 |
| SysAnnouncement | sys_announcement | ✗ | 公告表 |
| SysMessage | sys_message | ✗ | 站内信表 |
| SysMessageText | sys_message_text | ✗ | 站内信内容表 |
| SysMessageUser | sys_message_user | ✗ | 站内信用户表 |
| SysRoleDept | sys_role_dept | ✗ | 角色部门关联表 |
| SysUserOAuth | sys_user_o_auth | ✗ | OAuth表 |
| SysUserOnline | sys_user_online | ✗ | 在线用户表 |
| SysUserPost | sys_user_post | ✗ | 用户岗位关联表 |
| CasbinRule | casbin_rule | ✗ | Casbin规则表 |
