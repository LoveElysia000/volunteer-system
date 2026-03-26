# 前后端联调变更说明

本文档整理近期活动与组织接口的联调变更，供前端调整页面调用逻辑时参考。

## 变更概览

本次主要涉及两部分：

1. 活动列表接口统一
2. 志愿者端组织浏览接口新增

## 活动接口变更

### 旧接口

```http
POST /api/activities
POST /api/activities/my
```

### 新接口

```http
POST /api/activities
```

说明：

- `/api/activities/my` 已删除
- “活动大厅”和“我的活动”统一走 `/api/activities`

### `POST /api/activities` 请求参数

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `page` | `number` | 否 | 页码，默认 `1` |
| `pageSize` | `number` | 否 | 每页数量，默认 `50` |
| `status` | `number` | 否 | 活动状态：`1-报名中`、`2-已结束`、`3-已取消` |
| `keyword` | `string` | 否 | 匹配标题/描述/地点 |
| `startFrom` | `string` | 否 | 开始时间下限，格式：`yyyy-MM-dd HH:mm:ss` |
| `startTo` | `string` | 否 | 开始时间上限，格式：`yyyy-MM-dd HH:mm:ss` |
| `sortBy` | `string` | 否 | 预留字段，当前后端未使用 |
| `sortOrder` | `string` | 否 | 预留字段，当前后端未使用 |
| `registeredOnly` | `boolean` | 否 | 是否只看“我已报名”的活动 |

### `registeredOnly` 语义

当：

```json
{
  "registeredOnly": true
}
```

表示只返回当前志愿者：

- 待审核报名的活动
- 已报名成功的活动

不包含：

- 已驳回的活动
- 已取消报名的活动

### 请求示例

#### 全部活动

```json
{
  "page": 1,
  "pageSize": 10
}
```

#### 我的活动

```json
{
  "page": 1,
  "pageSize": 10,
  "registeredOnly": true
}
```

#### 我的活动 + 关键字筛选

```json
{
  "page": 1,
  "pageSize": 10,
  "registeredOnly": true,
  "keyword": "巡河"
}
```

说明：

- 后端会对 `registeredOnly`、`keyword`、`status`、`startFrom`、`startTo` 按交集处理
- 不会把未报名活动误查出来

### 返回结构

```json
{
  "total": 2,
  "list": [
    {
      "id": 101,
      "title": "河道清理行动",
      "description": "周末环保志愿活动",
      "coverUrl": "",
      "startTime": "2026-03-30 09:00:00",
      "endTime": "2026-03-30 12:00:00",
      "location": "滨河公园",
      "duration": 3,
      "maxPeople": 30,
      "currentPeople": 18,
      "status": 1,
      "isRegistered": true,
      "isFull": false
    }
  ]
}
```

### 前端改造建议

#### 活动大厅页

调用：

```http
POST /api/activities
```

请求：

```json
{
  "page": 1,
  "pageSize": 10
}
```

#### 我的活动页

原来如果调用的是：

```http
POST /api/activities/my
```

现在改成：

```http
POST /api/activities
```

请求改为：

```json
{
  "page": 1,
  "pageSize": 10,
  "registeredOnly": true
}
```

## 组织接口变更

### 新增接口

```http
POST /api/organizations/public-list
```

用途：

- 志愿者浏览可加入组织
- 不走管理端组织权限范围过滤
- 默认只返回正常状态组织

### 请求参数

请求体复用原来的 `OrganizationListRequest`：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `keyword` | `string` | 否 | 组织名称关键字搜索 |
| `status` | `number[]` | 否 | 前端可不传，后端固定只查正常组织 |
| `organizationType` | `string` | 否 | 预留字段，目前未实际使用 |
| `region` | `string` | 否 | 预留字段，目前未实际使用 |
| `page` | `number` | 否 | 页码 |
| `pageSize` | `number` | 否 | 每页数量 |

### 请求示例

```json
{
  "page": 1,
  "pageSize": 10,
  "keyword": "环保"
}
```

### 返回结构

```json
{
  "total": 1,
  "list": [
    {
      "id": 12,
      "name": "绿色家园志愿协会",
      "organizationCode": "ORG-0012",
      "contactPerson": "张三",
      "contactPhone": "",
      "email": "",
      "address": "杭州市西湖区",
      "status": 1,
      "organizationType": "",
      "region": "",
      "createdAt": "2026-03-20 10:00:00"
    }
  ]
}
```

注意：

- `contactPhone` 在公开组织列表中不返回真实值
- 前端不要依赖 `contactPhone` 展示公开联系电话

### 前端改造建议

#### 志愿者组织发现页

改用：

```http
POST /api/organizations/public-list
```

#### 管理端组织列表页

继续使用：

```http
POST /api/organizations/list
```

不要混用。

## 接口对照表

### 活动相关

| 页面/功能 | 旧接口 | 新接口 | 说明 |
|---|---|---|---|
| 活动大厅 | `POST /api/activities` | `POST /api/activities` | 不变 |
| 我的活动 | `POST /api/activities/my` | `POST /api/activities` | 传 `registeredOnly=true` |
| 活动详情 | `GET /api/activities/:id` | `GET /api/activities/:id` | 不变 |
| 活动报名 | `POST /api/activities/signup` | `POST /api/activities/signup` | 不变 |
| 取消报名 | `POST /api/activities/cancel` | `POST /api/activities/cancel` | 不变 |

### 组织相关

| 页面/功能 | 旧接口 | 新接口 | 说明 |
|---|---|---|---|
| 志愿者浏览组织 | 无 / 错误使用管理端接口 | `POST /api/organizations/public-list` | 新增 |
| 管理端组织列表 | `POST /api/organizations/list` | `POST /api/organizations/list` | 不变 |
| 志愿者已加入组织 | `GET /api/volunteers/{volunteerId}/organizations` | `GET /api/volunteers/{volunteerId}/organizations` | 不变 |

## 前端必须修改项

1. 删除对 `POST /api/activities/my` 的调用
2. “我的活动”页面改为调用 `POST /api/activities` 并传 `registeredOnly:true`
3. 志愿者组织发现页改为调用 `POST /api/organizations/public-list`
4. 公开组织列表页不要展示 `contactPhone`
