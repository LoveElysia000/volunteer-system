# 签到码/签退码改造实施方案

## 1. 结论

需要落库，不建议只在内存里校验。

原因：
- 服务重启后内存数据会丢失，签到码/签退码会失效。
- 多实例部署时，各实例内存不一致，校验会随机失败。
- 需要保留“当前有效码”与过期时间，便于审计和排错。

另外，建议不要直接存明文签到码/签退码，推荐存 `hash`。

## 2. 目标与范围

目标：
- 组织端可为活动生成签到码、签退码。
- 志愿者端签到/签退时必须提交对应码并通过校验。
- 与现有签到签退、工时发放逻辑兼容。

不在本期范围（可后续迭代）：
- 动态二维码（短时轮换码）
- 地理围栏、蓝牙/NFC 到场校验

## 3. 数据库设计（推荐）

在 `activities` 表增加以下字段（只保存“当前有效码”）：

- `check_in_code_hash` `VARCHAR(128) NOT NULL DEFAULT ''`：签到码哈希
- `check_out_code_hash` `VARCHAR(128) NOT NULL DEFAULT ''`：签退码哈希
- `check_in_code_expire_at` `DATETIME NULL`：签到码过期时间
- `check_out_code_expire_at` `DATETIME NULL`：签退码过期时间
- `attendance_code_version` `BIGINT NOT NULL DEFAULT 0`：码版本号（每次重置 +1）
- `attendance_code_updated_at` `DATETIME NULL`：最后更新时间

建议新增迁移文件：
- `sql/ddl/ddl_v1.1.9.sql`

说明：
- 你提到“加两个字段”在功能上可行，但生产上建议至少带过期时间和版本号，避免后续重构。
- 若未来要保留历史码，可再新增 `activity_attendance_codes` 历史表，本期先不做。

## 4. 接口改造

### 4.1 组织端：生成签到码/签退码

新增接口（放在 `ActivityService` 更顺）：
- `POST /api/activities/{id}/attendance-codes/generate`

请求参数建议：
- `id`：活动ID（path）
- `regenerateCheckIn`：是否重置签到码
- `regenerateCheckOut`：是否重置签退码
- `checkInExpireAt`：签到码过期时间（可选）
- `checkOutExpireAt`：签退码过期时间（可选）

响应参数建议：
- `checkInCode`：仅本次返回明文
- `checkOutCode`：仅本次返回明文
- `attendanceCodeVersion`

权限：
- 仅活动所属组织账号可调用。

### 4.2 志愿者端：签到/签退请求体增加校验码

改造现有 proto：
- `ActivityCheckInRequest` 增加 `checkInCode`
- `ActivityCheckOutRequest` 增加 `checkOutCode`

接口路径保持不变：
- `POST /api/activities/checkin`
- `POST /api/activities/checkout`

## 5. 服务层流程改造

## 5.1 生成码流程（组织端）

1. 校验活动归属（复用 `ensureActivityOperableByCurrentOrg`）。
2. 生成随机码（建议 6~8 位，数字或数字+大写字母）。
3. 使用服务端密钥做哈希（如 HMAC-SHA256）后写入 DB。
4. 更新过期时间、版本号、更新时间。
5. 明文码只在响应返回一次，不落日志。

## 5.2 签到流程（志愿者端）

在 `ActivityCheckIn` 增加以下校验：
1. 请求体必须有 `checkInCode`。
2. 活动存在且未取消。
3. 活动已配置签到码且未过期。
4. `hash(inputCode)` 与 `check_in_code_hash` 比较（常量时间比较）。
5. 校验通过后，执行原签到逻辑（状态更新不变）。

## 5.3 签退流程（志愿者端）

在 `ActivityCheckOut` 增加以下校验：
1. 请求体必须有 `checkOutCode`。
2. 活动存在且未取消。
3. 活动已配置签退码且未过期。
4. `hash(inputCode)` 与 `check_out_code_hash` 比较。
5. 校验通过后，执行原签退+工时结算逻辑（现有事务逻辑可复用）。

## 6. 兼容与灰度策略

为避免一次性切换导致线上失败，建议加开关：
- 配置项：`attendance.code_required`（默认 `false`）

灰度步骤：
1. 先发布“后端支持码校验但不强制”版本。
2. 组织端先开始生成并运营使用签到码/签退码。
3. 观察一段时间后，将 `attendance.code_required=true`。

## 7. 安全与风控建议

- 不存明文码，只存哈希。
- 接口错误提示统一为“签到码错误或已过期”，避免泄露细节。
- 对签到/签退增加频控（按 `volunteer_id + activity_id`，如 5 次/分钟）。
- 组织端重置码后立即生效，旧码失效（依赖版本号）。
- 日志脱敏：请求体里的 code 不打印。

## 8. 代码落点清单（基于当前项目）

- `sql/ddl/ddl_v1.1.9.sql`：新增字段迁移
- `internal/api/activities.proto`：新增生成码接口；签到签退请求增加 code 字段
- `internal/handler/activities.go`：新增生成码 handler
- `internal/router/activities.go`：新增生成码路由
- `internal/service/activities.go`：
  - 新增 `GenerateAttendanceCodes`
  - 改造 `ActivityCheckIn`/`ActivityCheckOut` 校验
- `internal/model/activities.gen.go`、`internal/dao/activities.gen.go`：迁移后重新生成
- `internal/repository/activities.go`：补充更新码字段方法（可选，或复用通用更新）

## 9. 测试清单

单元测试：
- 生成码成功（仅签到、仅签退、同时生成）
- 非所属组织生成码被拒绝
- 签到码错误/过期校验失败
- 签退码错误/过期校验失败
- 已签到/已签退幂等场景与码校验共存

集成测试：
- 活动完整链路：报名成功 -> 组织生成码 -> 志愿者签到 -> 志愿者签退 -> 工时发放正确
- 码重置后旧码失效、新码生效

## 10. 开发顺序建议

1. 先做 DDL + model/dao 生成。
2. 再改 proto + handler/router。
3. 最后改 service 逻辑和测试。
4. 联调通过后再开启强制校验开关。

