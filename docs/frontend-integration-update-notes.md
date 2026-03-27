# 前端联调改动说明

## 说明

这份文档只整理这次后端实际改动后，前端需要同步处理的接口联调点。

特别注意参数位置：

- `GET /api/notifications`
  - 参数放在 query 中
- `POST /api/audits/pending`
  - 参数放在 body 中
- `POST /api/activities`
  - 参数放在 body 中
- `POST /api/activities/my`
  - 参数放在 body 中

不要把 `POST` 接口里的筛选参数误传成 query。

---

## 1. 通知中心

### 接口

- `GET /api/notifications`

### 后端改动

新增 query 参数：

- `keyword?: string`

搜索范围：

- `title`
- `content`
- `bizType`
- `eventType`

### 前端需要修改

- 把通知页搜索框内容传给后端
- 不再只依赖当前页数据做本地搜索过滤

### 参数位置

以下字段放在 query 中：

- `page`
- `pageSize`
- `unreadOnly`
- `keyword`

### 示例

```text
GET /api/notifications?page=1&pageSize=20&unreadOnly=true&keyword=审核
```

---

## 2. 审核中心

### 接口

- `POST /api/audits/pending`

### 后端口径

继续复用原有 `status` 筛选，不新增新字段。

- `status?: number[]`

状态值：

- `1` 待审核
- `2` 审核通过
- `3` 审核拒绝

### 前端需要修改

如果页面筛选是：

- 全部
- 待审核
- 审核通过
- 审核拒绝

对应传值：

- 全部：`[1,2,3]`
- 待审核：`[1]`
- 审核通过：`[2]`
- 审核拒绝：`[3]`

### 参数位置

以下字段全部放在 body 中：

- `targetTypes`
- `status`
- `keyword`
- `page`
- `pageSize`
- `createdFrom`
- `createdTo`
- `slaHours`

### 示例

```json
{
  "targetTypes": [1, 2, 3, 4],
  "status": [1, 2, 3],
  "keyword": "张三",
  "page": 1,
  "pageSize": 20,
  "createdFrom": "2026-03-01 00:00:00",
  "createdTo": "2026-03-31 23:59:59",
  "slaHours": 24
}
```

### 不需要做的事情

- 不需要新增 `queueState`
- 不需要新增 `overdueOnly`

---

## 3. 活动列表

### 接口

- `POST /api/activities`

### 后端改动

`status` 从单值改成数组：

- 原来：`status?: number`
- 现在：`status?: number[]`

状态值：

- `1` 招募中
- `2` 已结束
- `3` 已取消

### 前端需要修改

- 把活动列表请求里的 `status` 改成数组类型
- “全部”状态按数组传值，不再传单个数字

### 参数位置

以下字段全部放在 body 中：

- `page`
- `pageSize`
- `status`
- `keyword`
- `startFrom`
- `startTo`
- `sortBy`
- `sortOrder`

### 传值约定

- 全部：`[1,2,3]`
- 招募中：`[1]`
- 已结束：`[2]`
- 已取消：`[3]`

### 示例

```json
{
  "page": 1,
  "pageSize": 20,
  "status": [1, 2, 3],
  "keyword": "环保",
  "startFrom": "2026-03-01 00:00:00",
  "startTo": "2026-03-31 23:59:59",
  "sortBy": "start_time",
  "sortOrder": "asc"
}
```

---

## 4. 我的活动

### 接口

- `POST /api/activities/my`

### 后端改动

新增独立接口：

- `/api/activities/my`

这是独立新接口，不再建议继续通过：

- `POST /api/activities`
- 再传 `registeredOnly: true`

来表示“我的活动”。

### 前端需要修改

- 新增对应的前端 API 方法，例如 `listMyActivities`
- “我的活动 / 我的报名”场景改走 `/api/activities/my`
- 不再依赖 `registeredOnly` 表达“我的活动”

### 参数位置

以下字段全部放在 body 中：

- `page`
- `pageSize`
- `status`
- `keyword`
- `startFrom`
- `startTo`
- `sortBy`
- `sortOrder`

### 请求示例

```json
{
  "page": 1,
  "pageSize": 20,
  "status": [1, 2, 3],
  "keyword": "环保",
  "startFrom": "2026-03-01 00:00:00",
  "startTo": "2026-03-31 23:59:59",
  "sortBy": "start_time",
  "sortOrder": "asc"
}
```

### 返回

继续复用原有活动列表返回结构：

- `ActivityListResponse`

---

## 5. 前端改动汇总

### 新增

- 新增 `/api/activities/my` 的前端请求方法
- 通知接口支持 `keyword`

### 修改

- 审核中心筛选按 `status[]` 传值
- 活动列表 `status` 改成数组
- 活动“全部”筛选传 `[1,2,3]`
- 我的活动改走 `/api/activities/my`
- 通知页搜索改为请求后端

### 删除或停止使用

- 我的活动场景下继续使用 `registeredOnly: true`
- 通知页主搜索依赖当前页数据做本地过滤
