# 志愿者信息变更审核设计（当前实现版）

更新时间：2026-02-26  
适用范围：`/Users/Ein/project2/volunteer-system`

## 1. 实现状态

状态说明：`✅ 已实现` `🟡 部分实现` `❌ 未实现`

| 模块 | 状态 | 说明 |
| --- | --- | --- |
| 复用 `audit_records` 承载志愿者资料变更审核 | ✅ | 使用 `target_type=volunteer` + `operation_type=update` |
| 志愿者资料变更提审接口 | ✅ | `POST /api/volunteers/profile-change/submit` |
| 志愿者实名认证提审接口 | ✅ | `POST /api/volunteers/real-name/submit` |
| 审核通过按场景回写主表 | ✅ | `profile_update` / `real_name_verify` 分开处理 |
| 审核驳回副作用处理 | ✅ | 实名认证驳回会回写 `volunteers.audit_status=rejected` |
| `volunteer_audits` 专表方案 | ❌ | 当前不采用，避免重复模型和迁移成本 |
| 多目标统一待审列表（管理员） | 🟡 | 现有主要是成员待审列表，志愿者变更列表待补 |

## 2. 核心方案

### 2.1 为什么不新增 `volunteer_audits`

当前系统已有通用审核主线（`audit_records` + 审批接口），继续复用可以：

1. 降低迁移和维护成本（不新增表、不新增一套状态流转）。
2. 统一审核落库、审批、审计逻辑。
3. 通过快照 `scene` 字段区分同一 `target_type` 下的不同业务语义。

### 2.2 审核快照格式

`old_content` / `new_content` 统一存 JSON 字符串，更新类审核使用以下封装：

```json
{
  "scene": "volunteer_profile_update",
  "data": {
    "real_name": "张三"
  }
}
```

或：

```json
{
  "scene": "volunteer_real_name_verify",
  "data": {
    "real_name": "张三",
    "id_card": "110101199001011234"
  }
}
```

## 3. 场景定义

在 `internal/model/consts.go`：

1. `AuditSceneVolunteerProfileUpdate = "volunteer_profile_update"`
2. `AuditSceneVolunteerRealNameVerify = "volunteer_real_name_verify"`

## 4. 业务流程

### 4.1 资料变更提审

1. 志愿者调用 `POST /api/volunteers/profile-change/submit`。
2. 服务端读取当前 `volunteers` 主档，构建 old/new 快照（仅包含变更字段）。
3. 创建 `audit_records` 待审核记录（`operation_type=update`，`scene=volunteer_profile_update`）。
4. 审核通过后，根据 `new_content.data` 回写主表字段。

### 4.2 实名认证提审

1. 志愿者调用 `POST /api/volunteers/real-name/submit`。
2. 服务端校验 `realName/idCard` 并构建 old/new 快照。
3. 事务内写入审核记录，并将 `volunteers.audit_status` 置为 `pending`。
4. 审核通过：回写 `real_name/id_card`，并置 `audit_status=approved`。
5. 审核驳回：置 `audit_status=rejected`。

### 4.3 审批执行分发

`internal/service/audit.go` 中 `applyVolunteerAuditApproval` 使用 `switch`：

1. `scene=volunteer_profile_update`：解析资料变更 payload 回写。
2. `scene=volunteer_real_name_verify`：解析实名认证 payload 回写。
3. 非 update 历史记录：保留兼容逻辑，仅更新 `audit_status`。

## 5. 快照与序列化规范

1. 所有审核快照最终都序列化为 JSON 字符串存入 `audit_records`。
2. 创建/删除类审核优先使用 model 结构快照。
3. 更新类审核使用 patch/payload（按需字段）快照。
4. 统一由 `internal/service/audit_snapshot*.go` 负责构建与解析，减少重复逻辑。

## 6. 关键代码落点

1. `/Users/Ein/project2/volunteer-system/internal/service/audit_snapshot.go`
2. `/Users/Ein/project2/volunteer-system/internal/service/audit_snapshot_volunteer_update.go`
3. `/Users/Ein/project2/volunteer-system/internal/service/volunteer.go`
4. `/Users/Ein/project2/volunteer-system/internal/service/audit.go`
5. `/Users/Ein/project2/volunteer-system/internal/api/volunteer.proto`

## 7. 后续建议

1. 补管理员端“志愿者资料/实名”待审核列表接口，避免只能按审核 ID 操作。
2. 在管理端列表中直接展示 `old/new` 对比摘要（减少审核成本）。
3. 补充该链路集成测试：提审、审批通过、审批驳回、重复提审拦截。
