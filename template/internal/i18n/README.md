# Backend i18n 使用说明

## 概述

后端 i18n 系统基于 `go-i18n/v2` 实现，支持根据客户端的 `Accept-Language` 请求头自动翻译错误消息。

## 文件结构

```
internal/i18n/
├── translator.go         # 核心翻译逻辑
├── locales/
│   ├── zh-CN.json       # 中文翻译
│   └── en-US.json       # 英文翻译
└── README.md            # 本文档
```

## 使用方法

### 1. 在 Service 层抛出可翻译错误

```go
import "projecttemplate/internal/common/errors"

func (s *Service) AuthorizeGame(ctx context.Context, gameID, shopID int64) error {
    // 检查游戏授权...
    if !authorized {
        // 方式1: 使用预定义的可翻译错误构造函数
        return errors.NewGameNotAuthorizedTranslatable(gameID, shopID, gameName, shopName)

        // 方式2: 手动创建可翻译错误
        return errors.NewTranslatableError(
            "BIZ_GAME_NOT_AUTHORIZED",  // 翻译键
            403,                        // HTTP 状态码
            map[string]interface{}{     // 模板数据
                "GameName": gameName,
                "ShopName": shopName,
            },
        )
    }
    return nil
}
```

### 2. 错误响应格式

后端会根据 `Accept-Language` 自动翻译并返回：

```json
{
  "code": 403,
  "reason": "BIZ_GAME_NOT_AUTHORIZED",
  "msg": "游戏「Beat Saber」未授权，请先为门店「XX店」授权此游戏",
  "data": null,
  "timestamp": "2026-02-25T10:30:00+08:00",
  "request_id": "abc123"
}
```

### 3. 前端使用

前端直接显示 `msg` 字段，不需要维护翻译表：

```typescript
// 处理错误响应
function handleError(response: ErrorResponse) {
    // 直接显示后端返回的翻译后消息
    showToast(response.msg);

    // 如需根据错误类型做特殊处理，使用 reason 字段
    if (response.reason === "BIZ_GAME_NOT_AUTHORIZED") {
        // 跳转到授权页面
        navigateToAuthorizePage();
    }
}
```

### 4. 添加新的翻译键

1. 在 `locales/zh-CN.json` 中添加中文翻译：

```json
{
  "NEW_ERROR_KEY": {
    "other": "错误信息包含 {{.FieldName}}"
  }
}
```

2. 在 `locales/en-US.json` 中添加英文翻译：

```json
{
  "NEW_ERROR_KEY": {
    "other": "Error message with {{.FieldName}}"
  }
}
```

3. 创建对应的错误构造函数（可选）

## 已定义的翻译键

### 业务错误 (BIZ_*)

- `BIZ_GAME_NOT_AUTHORIZED` - 游戏未授权
- `BIZ_INSUFFICIENT_BALANCE` - 余额不足
- `BIZ_INSUFFICIENT_REMAINING` - 剩余次数不足
- `BIZ_RECHARGE_EXPIRED` - 充值已过期
- `BIZ_DEVICE_OFFLINE` - 设备离线
- `BIZ_PERMISSION_DENIED` - 权限不足
- `BIZ_INVALID_CREDENTIALS` - 登录凭据无效
- `BIZ_USER_DISABLED` - 用户已禁用

### 资源错误 (RESOURCE_*)

- `RESOURCE_NOT_FOUND` - 资源不存在
- `RESOURCE_USER_NOT_FOUND` - 用户不存在
- `RESOURCE_SHOP_NOT_FOUND` - 门店不存在
- `RESOURCE_DEVICE_NOT_FOUND` - 设备不存在
- `RESOURCE_GAME_NOT_FOUND` - 游戏不存在
- ...

### 重复错误 (DUPLICATE_*)

- `DUPLICATE_USERNAME` - 用户名已存在
- `DUPLICATE_SHOP_NAME` - 门店名称已存在
- `DUPLICATE_DEVICE_CODE` - 设备编码已存在
- ...

### 验证错误 (VALIDATION_*)

- `VALIDATION_ERROR` - 验证错误
- `VALIDATION_REQUIRED_FIELD` - 必填字段缺失
- ...

### 系统错误 (SYSTEM_*)

- `SYSTEM_ERROR` - 系统错误
- `SYSTEM_DATABASE_ERROR` - 数据库错误
- ...

## 语言匹配规则

中间件根据 `Accept-Language` 头解析语言：

| Accept-Language | 匹配语言 |
|----------------|---------|
| zh-CN, zh, zh-Hans | 简体中文 (zh-CN) |
| en-US, en, en-GB | 英文 (en-US) |
| 未匹配或空 | 默认 (zh-CN) |

支持 quality 值，如：`en-US,en;q=0.9,zh-CN;q=0.8`
