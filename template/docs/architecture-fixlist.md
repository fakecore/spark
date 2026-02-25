# DoorX Architecture Fix List

> 基于 `docs/architecture-spec.md` 规范的代码审查结果。
> 按优先级排序，每条包含：问题描述、涉及文件/行号、修复方向、预估影响范围。

---

## 优先级说明

| 级别 | 含义 | 处理策略 |
|------|------|---------|
| P0   | 架构性问题，阻碍后续开发 | 优先修复 |
| P1   | 层级违规，增加维护成本 | 本迭代内修复 |
| P2   | 设计味道，可逐步改善 | 碰到时修复 (Boy Scout Rule) |

---

## P0: Repo 之间互相依赖

### FIX-001: UserRepo 依赖 RoleRepo

**问题**: `systemUserRepo` 持有 `SystemRoleRepo` 引用，违反 "Repo 之间不得互相调用" 原则。

**文件及行号**:
- `internal/data/repository/system/user.go:35` — 字段定义 `roleRepo system.SystemRoleRepo`
- `internal/data/repository/system/user.go:40-46` — 构造函数注入 RoleRepo
- `internal/data/repository/system/user.go:162` — 调用 `r.roleRepo.ListUserPermissionMenus()`

**影响方法**:
- `GetUserProfile()` (L50-171) — 在 repo 层做了跨 repo 聚合

**修复方向**:
1. 从 `systemUserRepo` 中移除 `roleRepo` 字段
2. `GetUserProfile()` 的聚合逻辑上移到 `SystemUserUsecase`
3. UserRepo 只负责查询 `sys_user` 表的数据
4. 角色、岗位、部门、菜单的关联查询由 usecase 编排多个 repo 完成

**新增的 biz 层逻辑草案**:
```go
// internal/biz/system/user.go
func (uc *SystemUserUsecase) GetUserProfile(ctx context.Context, userID int64) (*do.SysUserProfile, error) {
    user, err := uc.userRepo.FindByID(ctx, userID)
    roles, err := uc.roleRepo.ListUserRoles(ctx, userID)
    posts, err := uc.postRepo.ListUserPosts(ctx, userID)
    menus, err := uc.roleRepo.ListUserPermissionMenus(ctx, userID)
    // 组装 profile...
}
```

**预估影响**: UserUsecase 需要新增 RoleRepo 依赖；FX provider 注入调整。

---

### FIX-002: MenuRepo 依赖 RoleRepo

**问题**: `systemMenuRepo` 持有 `SystemRoleRepo` 引用。

**文件及行号**:
- `internal/data/repository/system/menu.go:23` — 字段定义 `roleRepo system.SystemRoleRepo`
- `internal/data/repository/system/menu.go:27-33` — 构造函数注入 RoleRepo
- `internal/data/repository/system/menu.go:276` — `ListUserPermissions()` 调用 `r.roleRepo.ListUserRoles()`

**影响方法**:
- `ListUserPermissions()` (L271-314)
- `CheckMenuPermission()` (L247-267) — 直接用 Casbin Enforce
- `CheckMultiplePermissions()` (L317-357) — 直接用 Casbin Enforce

**修复方向**:
1. 从 `systemMenuRepo` 中移除 `roleRepo` 字段
2. `ListUserPermissions()` 逻辑上移到 `PermissionUsecase`
3. `CheckMenuPermission()` 和 `CheckMultiplePermissions()` 移到 `PermissionUsecase`
4. MenuRepo 只负责 `sys_menu` 表的 CRUD

**预估影响**: MenuRepo interface 需要瘦身；MenuUsecase 和 PermissionUsecase 需要调整。

---

## P0: 业务逻辑下沉到 Data 层

### FIX-003: UserRepo.Create() 包含业务验证

**问题**: 用户名查重是业务规则，不应在 repo 层。

**文件及行号**:
- `internal/data/repository/system/user.go:418-424` — 用户名重复检查
- `internal/data/repository/system/user.go:431-432` — MySQL 错误码判断

**修复方向**:
1. `UserRepo.Create()` 只做数据库插入
2. 用户名查重逻辑移到 `SystemUserUsecase.CreateUser()`
3. 唯一索引冲突的错误转换可保留在 repo 层（属于数据访问层适配）

---

### FIX-004: UserRepo.CreateWithRoles() 包含复合业务操作

**问题**: "创建用户 + 分配角色" 是业务编排，不应在 repo 层。repo 里还包含了角色存在性验证和 casbin 规则创建。

**文件及行号**:
- `internal/data/repository/system/user.go:228-266` — 完整方法
- L236-244 — 角色存在性验证（业务逻辑）
- L253-263 — Casbin 规则创建（应在独立 CasbinRuleRepo）

**修复方向**:
1. 从 `SystemUserRepo` interface 中移除 `CreateWithRoles` 方法
2. 在 `SystemUserUsecase` 中编排: 验证 → 创建用户 → 分配角色 (通过 CasbinRuleRepo)
3. 事务由 usecase 层控制

---

### FIX-005: UserRepo.UpdateUserWithBindings() 是巨型 God Method

**问题**: 120 行的方法包含: 用户存在性验证、角色/岗位存在性验证、事务管理、基本信息更新、角色绑定更新、岗位绑定更新、Casbin 规则重载。这是典型的业务编排逻辑。

**文件及行号**:
- `internal/data/repository/system/user.go:282-408` — 完整方法
- L288-291 — 用户存在性验证（业务逻辑）
- L294-305 — 角色存在性验证（业务逻辑）
- L307-318 — 岗位存在性验证（业务逻辑）
- L320-339 — 事务管理（应在 biz 层）
- L353-373 — Casbin 规则操作（应在 CasbinRuleRepo）
- L376-394 — UserPost 关联操作（应在独立 UserPostRepo 或 biz 编排）
- L402-406 — Casbin LoadPolicy（应在 biz 层事务后）

**修复方向**:
1. 从 `SystemUserRepo` interface 中移除 `UpdateUserWithBindings` 方法
2. 拆分为多个原子 repo 操作:
   - `UserRepo.Update(ctx, user)` — 更新用户基本信息
   - `CasbinRuleRepo.ReplaceUserRoles(ctx, userID, roleIDs)` — 替换用户角色
   - `UserPostRepo.ReplaceUserPosts(ctx, userID, postIDs)` — 替换用户岗位
3. 在 `SystemUserUsecase.UpdateUserWithBindings()` 中用事务编排

---

### FIX-006: RoleRepo.CreateWithMenu() 包含复合业务操作

**问题**: "创建角色 + 分配菜单权限" 是业务编排。

**文件及行号**:
- `internal/data/repository/system/role.go:88-149` — 完整方法
- L111-119 — 菜单存在性验证（业务逻辑）
- L122-132 — Casbin 规则创建（应集中管理）
- L141-146 — Casbin LoadPolicy（应在 biz 层）

**修复方向**:
1. 从 `SystemRoleRepo` interface 中移除 `CreateWithMenu` 方法
2. 在 `SystemRoleUsecase` 中编排: 验证菜单 → 创建角色 → 分配权限
3. Casbin 规则操作委托给 `CasbinRuleRepo`

---

### FIX-007: RoleRepo.Update() 包含菜单权限同步

**问题**: 角色更新方法中包含了菜单权限的验证、删除、创建和 Casbin 重载。

**文件及行号**:
- `internal/data/repository/system/role.go:207-291` — 完整方法
- L245-276 — 菜单权限更新逻辑（应在 biz 层）
- L284-288 — Casbin LoadPolicy（应在 biz 层）

**修复方向**:
1. `RoleRepo.Update()` 只更新 `sys_role` 表基本字段
2. 菜单权限更新由 `PermissionUsecase` 或 `RoleUsecase` 编排

---

### FIX-008: RoleRepo.Delete() 包含级联清理

**问题**: 删除角色时同时清理 casbin 规则（p 规则和 g 规则），这是业务逻辑。

**文件及行号**:
- `internal/data/repository/system/role.go:152-205`
- L181-184 — 删除 p 规则
- L187-190 — 删除 g 规则
- L198-202 — Casbin LoadPolicy

**修复方向**:
1. `RoleRepo.Delete()` 只删除 `sys_role` 记录
2. Casbin 规则清理由 `RoleUsecase.Delete()` 编排，委托 `CasbinRuleRepo`

---

### FIX-009: RoleRepo.UpdateRoleMenus() 包含完整的业务编排

**问题**: 70 行的方法包含验证、差异计算、事务、规则增删。

**文件及行号**:
- `internal/data/repository/system/role.go:627-729`
- L633-639 — 菜单存在性验证
- L642-659 — 获取现有权限并计算差异
- L669-689 — 删除过期权限
- L692-713 — 添加新权限
- L721-726 — Casbin LoadPolicy

**修复方向**:
1. 差异计算和验证上移到 biz 层
2. Repo 只提供 `DeleteRoleMenuPermissions` 和 `CreateRoleMenuPermissions` 原子操作

---

### FIX-010: RoleRepo.UpdateUserRoles() 包含业务编排

**文件及行号**:
- `internal/data/repository/system/role.go:854-915`
- L860-865 — 角色存在性验证
- L876-884 — 删除旧分配
- L887-899 — 创建新分配
- L907-912 — Casbin LoadPolicy

**修复方向**: 同 FIX-009 模式。上移到 biz 层编排。

---

## P0: Casbin 操作散落在各层

### FIX-011: Casbin Enforce 调用散落在 Repo 层

**问题**: 权限检查（`Casbin.Enforce`）是业务逻辑，不应在 data 层。

**文件及行号**:
- `internal/data/repository/system/role.go:805-818` — `CheckPermission()` 直接调用 `s.data.Casbin.Enforce`
- `internal/data/repository/system/menu.go:258-261` — `CheckMenuPermission()` 直接调用 `r.data.Casbin.Enforce`
- `internal/data/repository/system/menu.go:343-346` — `CheckMultiplePermissions()` 直接调用 `r.data.Casbin.Enforce`

**修复方向**:
1. 创建 `internal/biz/system/casbin.go`，定义 `CasbinService` 或扩展 `PermissionUsecase`
2. 所有 `Enforce` 调用集中在 biz 层
3. 从 Repo interface 中移除 `CheckPermission`、`CheckMenuPermission`、`CheckMultiplePermissions` 方法
4. Infrastructure 层的 `*casbin.Enforcer` 通过 FX 注入到 biz 层

---

### FIX-012: Casbin LoadPolicy 散落在 Repo 层

**问题**: 每个修改 casbin_rule 的 repo 方法结尾都调用 `s.data.Casbin.LoadPolicy()`。

**文件及行号**:
- `internal/data/repository/system/user.go:402-406`
- `internal/data/repository/system/role.go:141-146`
- `internal/data/repository/system/role.go:198-202`
- `internal/data/repository/system/role.go:284-288`
- `internal/data/repository/system/role.go:721-726`
- `internal/data/repository/system/role.go:907-912`

**修复方向**:
1. `LoadPolicy` 调用统一放在 biz 层的事务提交之后
2. 可以在 `CasbinRuleRepo` 的上层 usecase 中设置一个 `defer casbinEnforcer.LoadPolicy()`
3. 或者使用事件机制: 事务提交后发布事件 → 监听器触发 LoadPolicy

---

### FIX-013: 硬编码管理员角色 ID (roleId == 1)

**问题**: 管理员判断通过硬编码 `roleId == 1` 实现，散落在 repo 层。

**文件及行号**:
- `internal/data/repository/system/role.go:484-491` — `ListRolePermissionMenus()` 中 `if roleId == 1`
- `internal/data/repository/system/role.go:541-551` — `ListUserPermission()` 中 `if roleID == 1`

**修复方向**:
1. 管理员判断应在 biz 层，通过角色属性（如 `is_admin` 字段）而非硬编码 ID
2. 抽取为 `PermissionUsecase.IsAdmin(roleID)` 方法
3. Repo 层不应包含此类业务分支

---

## P1: Usecase 层是空壳透传

### FIX-014: SystemUserUsecase 大部分方法是纯透传

**问题**: Usecase 方法只是 `return uc.repo.Xxx()`，没有业务价值。说明业务逻辑实际在 repo 里。

**文件及行号**:
- `internal/biz/system/user.go:69-71` — `Update()` 纯透传
- `internal/biz/system/user.go:74-76` — `UpdateUserWithBindings()` 纯透传
- `internal/biz/system/user.go:78-80` — `CreateUser()` 纯透传
- `internal/biz/system/user.go:82-84` — `SelectUserProfile()` 纯透传
- `internal/biz/system/user.go:86-88` — `GetUserByName()` 纯透传
- `internal/biz/system/user.go:90-92` — `GetUserList()` 纯透传
- `internal/biz/system/user.go:94-96` — `GetUserByID()` 纯透传
- `internal/biz/system/user.go:98-100` — `GetUserByMobile()` 纯透传
- `internal/biz/system/user.go:102-104` — `GetUserRoles()` 纯透传

**修复方向**:
当 FIX-003 ~ FIX-005 的业务逻辑从 repo 上移后，这些 usecase 方法将自然充实。
- `CreateUser()` → 应包含用户名查重 + 调 repo.Create + 调 CasbinRuleRepo 分配角色
- `UpdateUserWithBindings()` → 应包含验证 + 事务编排
- 对于确实只需要查询的方法（GetUserByID 等），透传是合理的

---

### FIX-015: SystemRoleUsecase 全部方法是纯透传

**问题**: 与 FIX-014 相同，所有方法 1:1 委托给 repo。

**文件及行号**:
- `internal/biz/system/role.go:40-101` — 所有方法

**修复方向**: 同 FIX-014，等 repo 层业务逻辑上移后自然充实。

---

### FIX-016: SystemUserUsecase 包含未实现的 panic 方法

**问题**: 多个方法直接 `panic("unimplemented")`，是代码异味。

**文件及行号**:
- `internal/biz/system/user.go:44-46` — `FindByMobile()` panic
- `internal/biz/system/user.go:49-51` — `FindByName()` panic
- `internal/biz/system/user.go:54-56` — `ListAll()` panic
- `internal/biz/system/user.go:59-61` — `ListByHello()` panic
- `internal/biz/system/user.go:64-66` — `Save()` panic

**修复方向**:
1. 如果不需要这些方法，直接删除
2. 如果需要，实现具体逻辑
3. `ListByHello` 看起来是遗留代码，建议删除

---

## P1: Service 层问题

### FIX-017: UserService 依赖 RoleUsecase

**问题**: `UserService` 同时持有 `userUc` 和 `roleUc`。

**文件及行号**:
- `internal/service/system/user.go:22-25` — 结构体定义
- `internal/service/system/user.go:27-33` — 构造函数

**当前使用情况**: 经审查，`roleUc` 在 UserService 中实际**未被使用**（所有方法只调用了 `userUc`）。

**修复方向**:
1. 从 `UserService` 中移除 `roleUc` 字段
2. 修改构造函数签名，移除 `SystemRoleUsecase` 参数
3. 如果未来需要角色相关操作，应通过 `userUc` 间接完成（由 userUc 持有 roleRepo interface）

---

### FIX-018: AuthService 直接 import infrastructure 包 [P0]

**问题**: `AuthService` 直接依赖 `infrastructure.Database` 来获取 `kvstore.Store`。`auth.go:14` 的 `import "projecttemplate/internal/infrastructure"` 直接违反 spec 禁止的 `service -> infrastructure` 规则。

**文件及行号**:
- `internal/service/system/auth.go:14` — import `projecttemplate/internal/infrastructure`
- `internal/service/system/auth.go:30-37` — 构造函数接收 `*infrastructure.Database`
- L36 — `data.KV` 直接取 KV store

**修复方向**:
1. `AuthService` 应直接接收 `kvstore.Store` 而非 `*infrastructure.Database`
2. 修改构造函数: `NewAuthService(userUc, logger, config, kv kvstore.Store)`
3. FX 注入直接提供 `kvstore.Store`
4. 移除 `auth.go` 对 `infrastructure` 包的 import

---

## P1: UserRepo 的 interface 定义问题

### FIX-019: SystemUserRepo 接口包含不属于 User 域的方法

**问题**: `GetUserRoles` 返回 `[]*model.SysRole`，`ExistsRole` 验证角色存在性 — 这些属于角色域。

**文件及行号**:
- `internal/biz/system/user.go:17` — `GetUserRoles` 方法签名
- `internal/biz/system/user.go:20` — `ExistsRole` 方法签名

**修复方向**:
1. 从 `SystemUserRepo` 移除 `GetUserRoles` 和 `ExistsRole`
2. 角色查询通过 `SystemRoleRepo.ListUserRoles()` 完成
3. UserUsecase 需要查角色时，通过注入 RoleRepo interface 完成

---

### FIX-020: SystemUserRepo.GetUserProfile 返回聚合对象

**问题**: `GetUserProfile` 返回 `*do.SysUserProfile` 包含用户、角色、岗位、部门、菜单数据。这是跨多表的聚合查询，不应由单个 repo 完成。

**文件及行号**:
- `internal/biz/system/user.go:18` — interface 定义
- `internal/data/repository/system/user.go:50-171` — 170 行实现

**修复方向**:
1. 从 `SystemUserRepo` 中移除 `GetUserProfile`
2. Profile 聚合由 `SystemUserUsecase.GetUserProfile()` 编排多个 repo 完成
3. 参见 FIX-001 的草案

---

## P0: RoleRepo 接口过大 — 后续修复的前置条件

### FIX-021: SystemRoleRepo 接口有 18 个方法

**问题**: 接口过大，违反接口隔离原则。包含了角色 CRUD、用户角色关系、菜单权限管理、权限检查等多个职责。拆分此接口是 Batch 1 所有后续修复的前置条件。

**文件及行号**:
- `internal/biz/system/role.go:11-29` — 完整接口定义

**修复方向**: 拆分为多个聚焦的接口:

```
SystemRoleRepo (角色 CRUD)
  - Get, List, Create, Update, UpSert, Delete

CasbinRuleRepo (casbin_rule 表操作)
  - AssignUserRole, RevokeUserRole, ListUserRoleIDs
  - SetRoleMenus, ListRoleMenuIDs
  - ListUserPermissionMenuIDs

PermissionChecker (权限查询, 可选独立接口)
  - CheckPermission
  - ListRolePermissionMenus
  - ListUserPermissionMenus
```

---

## P2: 代码异味和改善项

### FIX-022: UserRepo.GetUserProfile 和 GetUserRoles 有大量重复代码

**问题**: 两个方法都从 casbin_rule 表查询用户角色，代码重复。

**文件及行号**:
- `internal/data/repository/system/user.go:58-80` (GetUserProfile 中)
- `internal/data/repository/system/user.go:194-226` (GetUserRoles)

**修复方向**: 当业务逻辑上移后自然消除。在此之前可先抽取私有方法 `getUserRoleIDs()`。

---

### FIX-023: UserRepo.ListUser 做了跨表部门树查询

**问题**: 用户列表查询中包含部门树递归查询逻辑。

**文件及行号**:
- `internal/data/repository/system/user.go:463-490` — 部门树查询

**修复方向**:
1. 部门树查询应由 `DeptRepo` 提供 `ListChildDeptIDs(parentID)` 方法
2. UserRepo 只接收部门 ID 列表作为过滤条件
3. 编排在 usecase 层: 先查部门树 → 再过滤用户

---

### FIX-024: MenuRepo.Create/Update 包含父子关系验证

**问题**: 菜单的同名检查、父菜单存在性验证、循环引用检查是业务规则。

**文件及行号**:
- `internal/data/repository/system/menu.go:72-96` — `Create()` 中的验证
- `internal/data/repository/system/menu.go:98-207` — `Update()` 中的验证

**修复方向**: 验证逻辑上移到 `SystemMenuUsecase`。MenuRepo 只做 INSERT/UPDATE。

---

### FIX-025: Service 层硬编码默认密码

**问题**: `generatePassword()` 返回硬编码密码 `"123456aA!"`。

**文件及行号**:
- `internal/service/system/user.go:333-335`

**修复方向**:
1. 使用随机密码生成或从配置读取
2. 或者要求创建用户时必须传入密码

---

### FIX-026: Service 层 verifySmsCode 是空函数

**文件及行号**:
- `internal/service/system/user.go:342-344`

**修复方向**: 实现实际的验证码校验逻辑，或标记为 TODO 并在未实现时返回 error。

---

### FIX-027: RoleService.GetUserPermission 查询了完整菜单但未使用

**问题**: 同时调用了 `ListUserPermission` 和 `ListUserPermissionMenus`，但 `menuObjs` 转换后的 `pbMenus` 未放入返回值。

**文件及行号**:
- `internal/service/system/role.go:252-255` — 查询了完整菜单对象
- `internal/service/system/role.go:258-263` — 转换了但未返回
- `internal/service/system/role.go:265-267` — 只返回了 menuIds

**修复方向**: 要么在返回值中包含 `pbMenus`，要么移除冗余查询。

---

### FIX-028: RoleRepo.UpdateRoleDataScope 事务管理不一致 [P1 - 数据完整性 Bug]

**问题**: 方法中开启了事务，但 delete 操作使用的是非事务的 `s.query` 对象，而 create 操作使用事务内的 `tx.Model()`。这意味着如果后续 create 失败并回滚，delete 已经生效且无法撤销，**导致数据丢失**。

**文件及行号**:
- `internal/data/repository/system/role.go:755` — 开始事务 `tx := s.data.DB...Begin()`
- `internal/data/repository/system/role.go:763-765` — 使用 `s.query.SysRoleDept`（非事务）删除 — **事务外执行！**
- `internal/data/repository/system/role.go:777` — 使用 `tx.Model()`（事务内）创建

**修复方向**: 确保事务内的所有操作使用同一个 `tx` 对象。将 L763-765 的 delete 改为通过 `tx` 执行。

---

## 补充发现的问题

### FIX-029: PermissionUsecase 全部方法是纯透传 [P1]

**问题**: 与 FIX-014/015 同类，`SystemPermissionUsecase` 的所有 6 个方法都是 1:1 委托给 `roleRepo`。

**文件及行号**:
- `internal/biz/system/permission.go:19-46` — 所有方法

**修复方向**: 当 Casbin 操作集中化后（FIX-011/012），PermissionUsecase 将成为权限逻辑的真正归属地，自然充实。

---

### FIX-030: MenuUsecase 权限方法是纯透传 [P1]

**问题**: `SystemMenuUsecase` 的 `CheckMenuPermission`、`ListUserPermissions`、`CheckMultiplePermissions` 都是 1:1 透传给 MenuRepo。而 FIX-002 已指出这些方法不应在 MenuRepo 里。

**文件及行号**:
- `internal/biz/system/menu.go:124-127` — `CheckMenuPermission()` 透传
- `internal/biz/system/menu.go:128-130` — `ListUserPermissions()` 透传
- `internal/biz/system/menu.go:132-137` — `CheckMultiplePermissions()` 透传

**修复方向**:
1. 从 `SystemMenuRepo` interface 中移除这 3 个权限方法
2. 从 `SystemMenuUsecase` 中移除对应的透传方法
3. 权限检查统一由 `PermissionUsecase` 提供

---

### FIX-031: RoleService.CheckAPIPermission 包含业务逻辑 [P1]

**问题**: Service 层中包含路径匹配循环（遍历菜单列表检查 path 是否匹配），这是业务规则，违反 spec 6.1 "不得包含业务逻辑"。

**文件及行号**:
- `internal/service/system/role.go:209-237` — `CheckAPIPermission()` 方法
- L223-232 — 遍历菜单列表进行路径匹配（业务逻辑）

**修复方向**:
1. 路径匹配逻辑移到 `PermissionUsecase.CheckAPIPermission()`
2. Service 层只做参数校验和调用 usecase

---

### FIX-032: RoleService 在 Service 层做数据库存在性校验 [P2]

**问题**: `UpdateRoleMenus` 和 `UpdateRoleDataScope` 先调用 `roleUc.Get()` 验证角色是否存在，这属于业务验证而非请求参数格式校验。

**文件及行号**:
- `internal/service/system/role.go:134-138` — `UpdateRoleMenus()` 中 `s.roleUc.Get()` 校验
- `internal/service/system/role.go:154-157` — `UpdateRoleDataScope()` 中 `s.roleUc.Get()` 校验

**修复方向**: 存在性校验移到对应的 usecase 方法内部。

---

### FIX-033: AuthService.Login 依赖将被移除的 GetUserRoles [P1]

**问题**: `AuthService.Login()` 调用 `s.userUc.GetUserRoles()`，而 FIX-019 计划从 `SystemUserRepo` 中移除 `GetUserRoles`。修复 FIX-019 时如果不同时处理此处，将导致编译失败。

**文件及行号**:
- `internal/service/system/auth.go:77` — 调用 `s.userUc.GetUserRoles(ctx, int64(loginUser.ID))`

**修复方向**:
1. 与 FIX-019 一起修复
2. Login 流程中的角色查询改为通过 `RoleUsecase` 或由 `AuthUsecase`（新建）编排
3. 或者 `UserUsecase` 注入 `CasbinRuleRepo` 后提供 `GetUserRoles` 能力

---

### FIX-034: UserRepo.ExistsRole 查询 sys_role 表 [P1]

**问题**: `systemUserRepo.ExistsRole()` 查询 `sys_role` 表验证角色是否存在，违反 "每个 repo 只操作自己的主表" 原则。

**文件及行号**:
- `internal/data/repository/system/user.go:173-180` — `ExistsRole()` 查询 `SysRole` 表

**修复方向**:
1. 从 `SystemUserRepo` interface 中移除 `ExistsRole` 方法（与 FIX-019 合并处理）
2. 角色存在性验证由 `SystemRoleRepo.Get()` 或 biz 层编排完成

---

### FIX-035: UserRepo.GetUserRoles 查询 casbin_rule 和 sys_role 表 [P1]

**问题**: `systemUserRepo.GetUserRoles()` 同时查询 `casbin_rule` 表和 `sys_role` 表，违反单表原则。

**文件及行号**:
- `internal/data/repository/system/user.go:194-226` — 查询 casbin_rule (L196-203) 后查询 sys_role (L223-225)

**修复方向**:
1. 从 `SystemUserRepo` interface 中移除 `GetUserRoles`（与 FIX-019 合并处理）
2. 用户角色查询由 `CasbinRuleRepo.ListUserRoleIDs()` + `RoleRepo.ListByIDs()` 组合完成
3. 由 biz 层编排

---

## 修复优先级排序建议

### 第一批：基础设施改造（全部 P0，后续所有修复的前置条件）

1. **FIX-021** — 拆分 RoleRepo 接口，创建 `CasbinRuleRepo` interface
2. **FIX-011/012** — Casbin Enforce + LoadPolicy 集中到 biz 层
3. **FIX-013** — 管理员硬编码 `roleId == 1` 改为可配置
4. **FIX-018** — AuthService 移除对 infrastructure 包的直接 import

### 第二批：上移业务逻辑（P0 核心修复）

5. **FIX-001/002** — 移除 UserRepo→RoleRepo、MenuRepo→RoleRepo 的依赖
6. **FIX-003/004/005** — UserRepo 业务逻辑上移到 UserUsecase
7. **FIX-006/007/008/009/010** — RoleRepo 业务逻辑上移到 RoleUsecase
8. **FIX-028** — 修复 UpdateRoleDataScope 事务 bug（数据完整性）
9. **FIX-034/035** — UserRepo 中跨表查询方法移除

### 第三批：充实 Usecase + 清理 Service（P1）

10. **FIX-014/015/029** — UserUsecase / RoleUsecase / PermissionUsecase 充实
11. **FIX-016** — 删除 panic 占位方法
12. **FIX-017** — UserService 移除未使用的 roleUc
13. **FIX-019/020** — SystemUserRepo interface 精简
14. **FIX-030** — MenuUsecase 权限方法迁移到 PermissionUsecase
15. **FIX-031** — RoleService.CheckAPIPermission 业务逻辑下移到 biz 层
16. **FIX-033** — AuthService.Login 角色查询路径更新（与 FIX-019 联动）

### 第四批：代码改善（P2）

17. **FIX-022 ~ FIX-027** — 重复代码、跨表查询、验证逻辑位置、硬编码密码等
18. **FIX-032** — Service 层存在性校验下移

---

## 新增文件/接口建议

在修复过程中，可能需要新增以下内容:

| 新增项 | 位置 | 说明 |
|--------|------|------|
| `CasbinRuleRepo` interface | `internal/biz/system/casbin.go` | casbin_rule 表操作接口 |
| `casbinRuleRepo` impl | `internal/data/repository/system/casbin.go` | 接口实现 |
| `UserPostRepo` interface | `internal/biz/system/user_post.go` | sys_user_post 关联表操作 |
| `TxManager` interface | `internal/biz/transaction.go` | 事务管理抽象 |
| `TxManager` impl | `internal/data/transaction.go` | GORM 事务实现 |
