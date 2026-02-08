这是一个按照**开发难度（由低到高）**排序的后端需求文档。

该文档基于现有的**志愿者表（Volunteers/Users）**基础，分阶段构建整个志愿者服务端（仪表盘 + 活动报名系统）。

---

# 志愿者服务平台 - 后端开发需求文档 (v1.0)

## 📌 项目背景

本项目旨在为志愿者提供一个个人中心仪表盘，展示个人贡献数据（积分、时长），并提供活动浏览与报名功能。
**前置条件**：数据库中已存在基础志愿者表 (`volunteers`)。

---

## 阶段一：基础架构搭建 (MVP)

**难度：⭐⭐**
**目标**：跑通最核心流程（看活动 -> 报名 -> 数据变动）。不涉及复杂算法和并发处理。

### 1. 数据库变更 (Database)

由于已有 `volunteers` 表，我们需要补充活动和报名关联表。

**A. 扩展志愿者表 (`volunteers`)**
*确认表中包含以下统计字段，若无需新增（暂时作为缓存字段直接读写）：*

```sql
ALTER TABLE volunteers ADD COLUMN total_points INT DEFAULT 0 COMMENT '总积分';
ALTER TABLE volunteers ADD COLUMN total_hours DECIMAL(10, 1) DEFAULT 0.0 COMMENT '总服务时长';
ALTER TABLE volunteers ADD COLUMN level_id INT DEFAULT 1 COMMENT '当前等级';

```

**B. 新增：活动主表 (`activities`)**

```sql
CREATE TABLE activities (
  id INT PRIMARY KEY AUTO_INCREMENT,
  title VARCHAR(100) NOT NULL COMMENT '活动标题',
  description TEXT COMMENT '活动描述/副标题',
  start_time DATETIME NOT NULL COMMENT '开始时间',
  end_time DATETIME NOT NULL COMMENT '结束时间',
  location VARCHAR(100) COMMENT '地点名称',
  duration DECIMAL(4, 1) DEFAULT 0.0 COMMENT '预估工时(用于展示)',
  max_people INT DEFAULT 0 COMMENT '最大招募人数',
  current_people INT DEFAULT 0 COMMENT '当前已报名人数(冗余字段，方便查询)',
  status TINYINT DEFAULT 1 COMMENT '1:报名中 2:已结束'
);

```

**C. 新增：报名记录表 (`activity_signups`)**

```sql
CREATE TABLE activity_signups (
  id INT PRIMARY KEY AUTO_INCREMENT,
  activity_id INT NOT NULL,
  volunteer_id INT NOT NULL,
  signup_time DATETIME DEFAULT CURRENT_TIMESTAMP,
  status TINYINT DEFAULT 1 COMMENT '1:已报名 2:已取消',
  UNIQUE KEY `uk_act_vol` (`activity_id`, `volunteer_id`) -- 防止重复报名
);

```

### 2. 接口需求 (API - Level 1)

#### 接口 1.1：获取个人概览数据

* **方法**：`GET /api/v1/home/summary`
* **逻辑**：直接查询 `volunteers` 表的 `total_points`, `total_hours` 字段返回。暂不计算“本月新增”。
* **返回示例**：
```json
{
  "nickname": "志愿者123",
  "level": 1,
  "stats": { "points": 100, "hours": 5.5, "activity_count": 3 }
}

```



#### 接口 1.2：获取活动列表（基础版）

* **方法**：`GET /api/v1/activities`
* **逻辑**：查询 `activities` 表中 `status=1` 的数据。暂时不判断当前用户是否已报名，所有按钮默认返回“可报名”。

#### 接口 1.3：活动报名

* **方法**：`POST /api/v1/activities/signup`
* **参数**：`{ activity_id: 101 }`
* **逻辑**：
1. 向 `activity_signups` 插入一条记录。
2. 更新 `activities` 表：`current_people = current_people + 1`。
3. (可选) 更新 `volunteers` 表的 `activity_count + 1`。



---

## 阶段二：状态联动与业务逻辑完善

**难度：⭐⭐⭐**
**目标**：解决“我是否已报名”的显示问题，并保证数据准确性。

### 1. 核心逻辑升级

**A. 动态判断用户报名状态 (重点)**
在获取活动列表时，不能只查活动表，必须结合 `signups` 表判断当前用户状态。

* **修改接口 1.2 (`GET /api/v1/activities`) 的逻辑**：
* **输入**：获取 Header 中的 `current_user_id`。
* **查询**：查询活动列表的同时，左连接 (Left Join) 或子查询 `activity_signups` 表。
* **输出**：为每个活动对象增加字段：
* `is_registered`: `true` (已报名) / `false` (未报名)
* `is_full`: `current_people >= max_people`


* **前端表现**：
* `is_registered=true` -> 按钮显示灰色“已报名”。
* `is_registered=false` & `is_full=false` -> 按钮显示绿色“立即报名”。





**B. 完善报名校验**

* **修改接口 1.3 的逻辑**：
* 先查活动：`if (current_people >= max_people)` -> 返回错误“名额已满”。
* 再查重复：`if (exists in signups)` -> 返回错误“请勿重复报名”。

```
用户状态,活动状态,按钮文案,按钮样式,点击行为
未报名,名额未满,立即报名,绿色/实心,弹窗确认报名
已报名,(任意),查看详情,墨绿色/实心,跳转详情页 (含取消按钮)
未报名,名额已满,名额已满,灰色/禁用,不可点击
未报名,时间冲突,时间冲突,灰色/禁用,提示与已有活动时间重叠 (可选的高级需求)
```

---

## 阶段三：统计分析与等级算法 (进阶)

**难度：⭐⭐⭐⭐**
**目标**：实现页面上的“本月新增+25”趋势数据，以及等级进度条计算。

### 1. 数据库变更

**新增：流水记录表 (`records`)**
*原本只存总分，现在需要存每一笔变动，才能算出“本月新增”。*

```sql
CREATE TABLE records (
  id INT PRIMARY KEY AUTO_INCREMENT,
  volunteer_id INT NOT NULL,
  type ENUM('POINT', 'HOUR') NOT NULL COMMENT '类型:积分或工时',
  amount DECIMAL(10,1) NOT NULL COMMENT '变动值,如 +3.0',
  create_time DATETIME DEFAULT CURRENT_TIMESTAMP
);

```

**新增：等级配置表 (`level_rules`)**

```sql
CREATE TABLE level_rules (
  level INT PRIMARY KEY,
  name VARCHAR(50), -- 如 '一星志愿者'
  threshold_hours INT -- 如 50 (升级所需小时数)
);

```

### 2. 接口逻辑升级

#### 升级接口 1.1：仪表盘复杂统计

* **计算“本月新增”**：
* SQL: `SELECT SUM(amount) FROM records WHERE volunteer_id = ? AND create_time >= '本月1号0点'`。
* 将结果作为 `monthly_growth` 字段返回。


* **计算“等级进度”**：
* 获取当前用户的 `total_hours`。
* 查询 `level_rules` 找到下一级所需的 `threshold`。
* 计算差值：`need_hours = threshold - total_hours`。
* 返回给前端提示文案：“还需 10 小时升级”。



---

## 阶段四：高并发保障与自动化 (高级)

**难度：⭐⭐⭐⭐⭐**
**目标**：防止活动“超卖”，处理过期活动。

### 1. 报名并发锁 (Concurrency Control)

* **场景**：活动剩 1 个名额，3 个人同时点报名。
* **解决方案**：使用数据库行锁或乐观锁更新名额。
```sql
-- 利用 update 的原子性
UPDATE activities 
SET current_people = current_people + 1 
WHERE id = {activity_id} 
  AND current_people < max_people; 
-- 如果 affected_rows == 0，说明手慢了，没抢到，返回“名额已满”。

```



### 2. 定时任务 (Cron Jobs)

* **任务**：自动归档过期活动。
* **逻辑**：每小时运行一次，将 `activities` 表中 `start_time < NOW()` 且 `status = 1` 的活动状态改为 `2 (已结束)`，或者前端直接根据时间过滤。

---

## ✅ 总结：开发路线图

1. **第一周**：建好 3 张表，写出不做校验的“列表”和“报名”接口。前端先能把假数据换成真数据。（阶段一）
2. **第二周**：加上 `is_registered` 字段，前端实现按钮变灰/变绿的逻辑；加上名额已满的拦截。（阶段二）
3. **第三周**：建立流水表，写 SQL 统计“本月新增”，实现等级计算。（阶段三）
4. **上线前**：压力测试报名接口，确保不会超卖。（阶段四）