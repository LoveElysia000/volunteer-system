# 签到码/签退码改造实施方案（简化版）

## 1. 结论

本期采用简化实现：
- 签到码、签退码直接明文存储在数据库；
- 暂不做哈希计算与哈希校验；
- 可先预留哈希字段，但本期不使用。

需要落库，不建议只在内存里校验。原因：
- 服务重启后内存数据会丢失，签到码/签退码会失效；
- 多实例部署时，各实例内存不一致，校验会随机失败；
- 需要保留“当前有效码”与过期时间，便于排错。

## 2. 目标与范围

目标：
- 组织端可为活动生成签到码、签退码；
- 志愿者端签到/签退时提交对应码并通过校验；
- 与现有签到签退、工时发放逻辑兼容。

不在本期范围（后续迭代）：
- 哈希存储与常量时间比较；
- 动态二维码（短时轮换码）；
- 地理围栏、蓝牙/NFC 到场校验。

## 3. 数据库设计（本期）

在 `activities` 表增加以下字段（只保存“当前有效码”）：

本期实际使用：
- `check_in_code` `VARCHAR(32) NOT NULL DEFAULT ''`：签到码（明文）
- `check_out_code` `VARCHAR(32) NOT NULL DEFAULT ''`：签退码（明文）
- `check_in_code_expire_at` `DATETIME NULL`：签到码过期时间
- `check_out_code_expire_at` `DATETIME NULL`：签退码过期时间
- `attendance_code_version` `BIGINT NOT NULL DEFAULT 0`：码版本号（每次重置 +1）
- `attendance_code_updated_at` `DATETIME NULL`：最后更新时间

预留但本期不使用：
- `check_in_code_hash` `VARCHAR(128) NOT NULL DEFAULT ''`：签到码哈希预留字段
- `check_out_code_hash` `VARCHAR(128) NOT NULL DEFAULT ''`：签退码哈希预留字段

建议迁移文件：
- `sql/ddl/ddl_v1.1.9.sql`

## 4. 接口改造

### 4.1 组织端：生成签到码/签退码

新增接口：
- `POST /api/activities/{id}/attendance-codes/generate`

请求参数建议：
- `id`：活动ID（path）
- `regenerateCheckIn`：是否重置签到码
- `regenerateCheckOut`：是否重置签退码
- `checkInValidMinutes`：签到码有效时长（分钟，可选，`<=0` 表示不过期）
- `checkOutValidMinutes`：签退码有效时长（分钟，可选，`<=0` 表示不过期）

响应参数建议：
- `checkInCode`：仅当 `regenerateCheckIn=true` 时返回本次明文
- `checkOutCode`：仅当 `regenerateCheckOut=true` 时返回本次明文
- `attendanceCodeVersion`
- `attendanceCodeUpdatedAt`
- `checkInExpireAt`：签到码过期时间（为空表示不过期）
- `checkOutExpireAt`：签退码过期时间（为空表示不过期）

权限：
- 仅活动所属组织账号可调用。

### 4.2 志愿者端：签到/签退请求体增加校验码

改造 proto：
- `ActivityCheckInRequest` 增加 `checkInCode`
- `ActivityCheckOutRequest` 增加 `checkOutCode`

接口路径保持不变：
- `POST /api/activities/checkin`
- `POST /api/activities/checkout`

## 5. 服务层流程改造

### 5.1 生成码流程（组织端）

1. 校验活动归属（复用 `ensureActivityOperableByCurrentOrg`）。
2. 生成随机码（6 位，数字+大写字母混合）。
3. 直接写入 `check_in_code` / `check_out_code`（明文）。
4. 根据 `checkInValidMinutes` / `checkOutValidMinutes` 计算并更新过期时间，同时更新版本号、更新时间。
5. 明文码只在响应返回一次，不落日志。

### 5.2 签到流程（志愿者端）

在 `ActivityCheckIn` 增加以下校验：
1. 请求体必须有 `checkInCode`。
2. 活动存在且未取消。
3. 活动已配置签到码且未过期。
4. `inputCode` 与 `check_in_code` 做字符串比较（建议先 `TrimSpace`）。
5. 校验通过后执行原签到逻辑（状态更新不变）。

### 5.3 签退流程（志愿者端）

在 `ActivityCheckOut` 增加以下校验：
1. 请求体必须有 `checkOutCode`。
2. 活动存在且未取消。
3. 活动已配置签退码且未过期。
4. `inputCode` 与 `check_out_code` 做字符串比较（建议先 `TrimSpace`）。
5. 校验通过后执行原签退+工时结算逻辑（复用现有事务）。

## 6. 兼容与灰度策略

配置项：
- `attendance.code_required`（默认 `false`）

约定：
- `false`：兼容模式，不强制码校验；
- `true`：强制校验签到码/签退码。

灰度步骤：
1. 先发布“后端支持生成和校验，但不强制”的版本；
2. 组织端先开始使用签到码/签退码；
3. 观察后切换 `attendance.code_required=true`。

## 7. 安全与风控建议（明文方案）

- 明文方案仅作为本期快速落地方案；
- 接口错误提示统一为“签到码错误或已过期”，避免泄露细节；
- 对签到/签退增加频控（按 `volunteer_id + activity_id`，如 5 次/分钟）；
- 组织端重置码后立即生效，旧码失效（数据库仅保留当前有效码，并同步更新版本号）；
- 日志脱敏：请求体里的 code 不打印；
- 严格控制数据库读权限（仅服务账号可读这些字段）。

## 8. 代码落点清单（基于当前项目）

- `sql/ddl/ddl_v1.1.9.sql`：新增明文字段 + 过期字段 + 版本字段（可选预留 hash 字段）
- `internal/api/activities.proto`：新增生成码接口；签到签退请求增加 code 字段
- `internal/handler/activities.go`：新增生成码 handler
- `internal/router/activities.go`：新增生成码路由
- `internal/service/activities.go`：
  - 新增 `GenerateAttendanceCodes`
  - 改造 `ActivityCheckIn`/`ActivityCheckOut` 做明文校验
- `internal/model/activities.gen.go`、`internal/dao/activities.gen.go`：迁移后重新生成
- `internal/repository/activities.go`：补充更新码字段方法（可选）

## 9. 测试清单

单元测试：
- 生成码成功（仅签到、仅签退、同时生成）；
- 生码时长参数校验（负数拒绝；未重置对应码时不可单独传有效时长）；
- 非所属组织生成码被拒绝；
- 签到码错误/过期校验失败；
- 签退码错误/过期校验失败；
- 有效时长 `<=0` 时返回空过期时间（不过期）；
- 已签到/已签退幂等场景与码校验共存。

集成测试：
- 活动完整链路：报名成功 -> 组织生成码 -> 志愿者签到 -> 志愿者签退 -> 工时发放正确；
- 码重置后旧码失效、新码生效。

## 10. 开发顺序建议

1. DDL + model/dao 生成；
2. proto + handler/router；
3. service 明文校验逻辑；
4. 测试与联调；
5. 联调稳定后再开启强制校验开关。

## 11. 下期升级（迁移到哈希）

下期可基于预留字段平滑升级：
1. 生成码时同时写入明文与哈希；
2. 校验先读哈希，兼容旧明文；
3. 完成数据迁移后下线明文字段或停止读取明文。
