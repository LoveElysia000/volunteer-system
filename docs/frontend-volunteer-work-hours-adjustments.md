# 志愿者工时流水接口拆分前端调整说明

## 背景

本次后端将志愿者工时流水查询从组织端通用接口中拆分出来，新增志愿者专用接口：

- 志愿者端：`POST /api/volunteers/work-hours`
- 组织端：`POST /api/work-hours/list`

调整目标是让志愿者端和组织端的接口职责、权限语义、响应结构彻底分离，避免前后端继续耦合同一套后台管理 DTO。

---

## 前端需要修改的内容

### 1. 志愿者端请求地址切换

志愿者端工时流水页面不要再调用：

```http
POST /api/work-hours/list
```

需要改为：

```http
POST /api/volunteers/work-hours
```

组织端页面维持原接口不变：

```http
POST /api/work-hours/list
```

---

### 2. 志愿者端请求参数调整

志愿者端新接口请求参数如下。

## 接口定义

### 1. 志愿者端查询接口

- 方法：`POST`
- 路径：`/api/volunteers/work-hours`
- 权限：志愿者登录
- 用途：查询当前登录志愿者自己的工时流水

### 2. 组织端查询接口

- 方法：`POST`
- 路径：`/api/work-hours/list`
- 权限：组织账号登录
- 用途：查询组织侧工时流水

说明：

- 志愿者端和组织端不要再共用同一个请求方法和同一个响应类型
- 志愿者端请只对接 `/api/volunteers/work-hours`

---

## 志愿者端请求参数

### 1. JSON 示例

```json
{
  "page": 1,
  "pageSize": 20,
  "activityId": 0,
  "operationTypes": [1, 3]
}
```

### 1.1 明确的合法写法

后端当前约定 `operationTypes` 必须是数字数组，前端请按下面几种形式传：

```json
{ "operationTypes": [1, 2, 3] }
```

表示筛选“发放 + 作废 + 重算”。

```json
{ "operationTypes": [1] }
```

表示只筛选“发放”。

```json
{ "operationTypes": [2] }
```

表示只筛选“作废”。

```json
{ "operationTypes": [3] }
```

表示只筛选“重算”。

```json
{}
```

或者完全不传 `operationTypes`，表示“全部”。

### 1.2 不建议或不支持的写法

下面这些写法不要使用：

```json
{ "operationType": 1 }
```

旧单值字段，后端不再按新查询契约使用。

```json
{ "operationTypes": 1 }
```

错误，`operationTypes` 必须是数组。

```json
{ "operationTypes": [0] }
```

错误，`0` 不是后端枚举值。

```json
{ "operationTypes": ["1", "2"] }
```

错误，元素应该是数字，不是字符串。

### 2. 字段表

| 字段名 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `page` | `number` | 否 | 页码，`<= 0` 时后端按 `1` 处理 |
| `pageSize` | `number` | 否 | 每页条数，`<= 0` 时后端按默认值处理，最大 `100` |
| `activityId` | `number` | 否 | 按活动 ID 过滤 |
| `operationTypes` | `number[]` | 否 | 工时流水类型列表过滤，数组元素只能是 `1 / 2 / 3` |

### 3. TypeScript 类型建议

```ts
export interface VolunteerWorkHoursRequest {
  page?: number;
  pageSize?: number;
  activityId?: number;
  operationTypes?: Array<1 | 2 | 3>;
}
```

说明：

- 志愿者端不再支持 `signupId` 查询条件
- 当前登录志愿者身份由后端自动识别，无需前端传 `volunteerId`
- 推荐前端统一传数组，不要再混用旧的单值筛选写法

---

## 志愿者端响应结构

### 1. JSON 示例

志愿者端新接口返回字段如下：

```json
{
  "total": 1,
  "list": [
    {
      "id": 1001,
      "activityId": 2001,
      "signupId": 3001,
      "operationType": 1,
      "hoursDelta": 3.5,
      "beforeTotalHours": 10,
      "afterTotalHours": 13.5,
      "reason": "",
      "createdAt": "2026-03-29 12:00:00"
    }
  ]
}
```

### 2. 顶层响应字段

| 字段名 | 类型 | 说明 |
| --- | --- | --- |
| `total` | `number` | 总条数 |
| `list` | `VolunteerWorkHourItem[]` | 当前页数据 |

### 3. 列表项字段

| 字段名 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `number` | 工时流水 ID |
| `activityId` | `number` | 活动 ID |
| `signupId` | `number` | 报名 ID |
| `operationType` | `number` | 工时流水类型，单条记录上的类型值 |
| `hoursDelta` | `number` | 本次工时变动值，作废时可能为负数 |
| `beforeTotalHours` | `number` | 变更前总工时 |
| `afterTotalHours` | `number` | 变更后总工时 |
| `reason` | `string` | 原因说明，可能为空字符串 |
| `createdAt` | `string` | 创建时间，格式 `YYYY-MM-DD HH:mm:ss` |

### 4. TypeScript 类型建议

```ts
export interface VolunteerWorkHourItem {
  id: number;
  activityId: number;
  signupId: number;
  operationType: 1 | 2 | 3;
  hoursDelta: number;
  beforeTotalHours: number;
  afterTotalHours: number;
  reason: string;
  createdAt: string;
}

export interface VolunteerWorkHoursResponse {
  total: number;
  list: VolunteerWorkHourItem[];
}
```

---

### 2. 志愿者端需要移除的字段依赖

如果当前志愿者端页面、类型定义、表格列、状态映射复用了组织端 DTO，需要移除对以下字段的依赖：

- `volunteerId`
- `operatorId`
- `idempotencyKey`
- `beforeServiceCount`
- `afterServiceCount`
- `workHourVersion`
- `refLogId`

这些字段仍可能存在于组织端接口，但不会出现在志愿者端接口中。

---

## 枚举定义

### 后端真实枚举：`operationTypes[]`

| 值 | 含义 | 前端展示建议 |
| --- | --- | --- |
| `1` | 发放 | `工时发放` |
| `2` | 作废 | `工时作废` |
| `3` | 重发 / 重算 | `工时重算` |

### TypeScript 枚举建议

```ts
export type WorkHourOperationType = 1 | 2 | 3;

export const workHourOperationTypeTextMap: Record<WorkHourOperationType, string> = {
  1: '工时发放',
  2: '工时作废',
  3: '工时重算',
};
```

说明：

- 后端真实枚举值只有 `1 | 2 | 3`
- `operationTypes` 是数组字段
- 当不传 `operationTypes` 时，表示“不筛选”
- 前端联调时请直接按 `[1,2,3]`、`[1]`、`[2]`、`[3]` 这几种形式理解和传参

如果筛选组件需要“全部”，建议仅前端本地定义一个筛选态：

```ts
export type WorkHourOperationTypeFilter = 0 | 1 | 2 | 3;
```

其中：

- `0` 仅表示前端本地的“全部 / 不筛选”状态
- 发请求时：
  - `0` 不要传给后端
  - 单选时传 `operationTypes: [1]`
  - 多选时传 `operationTypes: [1, 3]`
  - 全部时直接不传 `operationTypes`

---

## 页面与代码层建议调整

### 1. 接口层拆分方法

如果前端当前只有一个通用“工时流水查询”方法，建议拆成两个方法：

```ts
getVolunteerWorkHours(params)
getOrganizationWorkHours(params)
```

不要继续让志愿者端和组织端共用同一个接口方法或同一个响应类型。

建议至少拆成下面这样：

```ts
export async function getVolunteerWorkHours(
  params: VolunteerWorkHoursRequest,
): Promise<VolunteerWorkHoursResponse> {
  return request.post('/api/volunteers/work-hours', params);
}

export async function getOrganizationWorkHours(params: OrganizationWorkHoursRequest) {
  return request.post('/api/work-hours/list', params);
}
```

---

### 2. 页面影响范围

建议重点排查以下位置：

- 志愿者端工时流水页面
- 志愿者端 API 请求封装
- 工时流水相关 TypeScript 类型定义
- 工时流水列表表格列定义
- `operationType` 枚举映射

---

### 3. 表格列建议

如果志愿者端已有工时流水表格，建议列定义优先按下面整理：

| 列名 | 对应字段 | 备注 |
| --- | --- | --- |
| 时间 | `createdAt` | 直接展示 |
| 类型 | `operationType` | 通过枚举映射显示文案 |
| 工时变动 | `hoursDelta` | 建议按正负值着色 |
| 变更前总工时 | `beforeTotalHours` | 数字展示 |
| 变更后总工时 | `afterTotalHours` | 数字展示 |
| 原因 | `reason` | 空字符串时可显示 `-` |

如果页面需要跳转活动详情，可以使用：

- `activityId`
- `signupId`

---

### 4. 展示层适配建议

由于志愿者端接口已经裁剪掉后台审计字段，页面展示建议聚焦用户关心的信息：

- 工时变动时间
- 工时增量
- 变动前总工时
- 变动后总工时
- 操作类型
- 原因

如果页面还需要展示活动标题、活动时间等信息，当前后端接口尚未返回，需要前端自行补充关联数据，或后续再扩展后端字段。

---

## 前端展示文案建议

### 1. 筛选区文案

由于筛选字段已经从单值思路切成数组，前端文案建议明确体现“可多选”，避免用户误解。

推荐写法：

- 筛选标题：`工时类型`
- 筛选占位文案：`请选择工时类型（可多选）`
- 全部选项文案：`全部类型`

如果页面上有辅助说明，可以补一句：

- `支持多选，不选默认展示全部`

不建议继续使用这些容易引起歧义的写法：

- `请选择一种工时类型`
- `工时类型（单选）`
- `仅可选择一项`

---

### 2. 类型展示文案

列表里单条记录的 `operationType` 展示文案建议保持如下映射：

| 值 | 展示文案 |
| --- | --- |
| `1` | `工时发放` |
| `2` | `工时作废` |
| `3` | `工时重算` |

如果页面想更短，也可以使用：

| 值 | 简短文案 |
| --- | --- |
| `1` | `发放` |
| `2` | `作废` |
| `3` | `重算` |

建议在同一个页面内统一使用一套，不要列表写“发放”，筛选又写“工时发放”。

---

### 3. 空态文案建议

如果未传 `operationTypes`，默认表示“全部”，空态文案建议不要误导成筛选异常。

推荐写法：

- 默认空态：`暂无工时流水`
- 筛选后空态：`当前筛选条件下暂无工时流水`

如果有重置筛选按钮，可以配：

- 按钮文案：`查看全部`

---

### 4. 多选交互文案建议

如果前端使用多选下拉、复选组或标签筛选，建议统一交互表达：

- 选择全部时：直接清空 `operationTypes`
- 页面文案显示：`全部类型`
- 选择一项时：显示对应类型文案
- 选择多项时：显示 `已选 2 项` / `已选 3 项`

这样比在请求里传 `[1,2,3]` 更符合用户理解，也更容易和“全部”语义区分。

---

## 联调注意事项

### 1. 志愿者端

- 必须使用志愿者账号登录
- 调用 `/api/volunteers/work-hours`
- 不要传组织端特有筛选参数
- 不要依赖组织端返回的审计字段

### 2. 组织端

- 继续使用 `/api/work-hours/list`
- 保持现有筛选逻辑不变

### 3. 错误预期

如果志愿者账号误调组织端接口 `/api/work-hours/list`，后端会直接拒绝。

---

## 建议联调清单

- 志愿者端工时流水页面可以正常拉取列表
- 志愿者端分页正常
- 志愿者端 `operationTypes` 筛选正常
- 志愿者端按 `activityId` 筛选正常
- 志愿者端页面不再依赖后台审计字段
- 组织端工时流水页面行为保持不变

---

## 总结

本次前端改动核心只有一句话：

**志愿者端工时流水改调 `/api/volunteers/work-hours`，并使用志愿者专用响应结构；组织端继续使用 `/api/work-hours/list`。**

如果前端要最低成本落地，可以按下面顺序执行：

1. 新增 `VolunteerWorkHoursRequest / VolunteerWorkHourItem / VolunteerWorkHoursResponse` 类型
2. 新增 `getVolunteerWorkHours` 请求方法
3. 志愿者端页面切换到新接口
4. 删除志愿者端对组织端审计字段的依赖
5. 联调分页、筛选、空数据和错误场景
