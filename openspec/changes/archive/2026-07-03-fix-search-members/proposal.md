## Why

group-svc 的 `SearchMembers` 接口目前为 stub 实现，keyword 参数被完全忽略，实际返回的是全部成员列表，导致前端搜索群成员功能不可用。

## What Changes

- 修复 `MySQLRepo.SearchMembers` — 实现基于 keyword 的成员过滤查询
- `GroupMember` 模型仅有 `user_id` 可作为搜索字段，keyword 为数字时按 `user_id` 精确匹配，为空时返回全部成员（退化为 `ListMembers`）

## Capabilities

### New Capabilities
- 无

### Modified Capabilities
- 无（当前无 group-svc 的 openspec spec）

## Impact

- `services/group-svc/internal/repo/repo.go` — 修改 `SearchMembers` 方法
- 无 API 签名变更，无新增依赖
