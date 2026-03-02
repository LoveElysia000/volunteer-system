
---

# 志愿者活动核心业务数据库设计文档 (精简版)

**版本**：V1.0
**核心模块**：活动管理、报名与审核
**适用场景**：组织发布任务 -> 志愿者报名 -> 组织审核通过/拒绝

## 1. 业务流程与数据流转

1. **发布**：组织管理者向 `activities` 表写入数据。
2. **报名**：志愿者向 `activity_signups` 表写入一条状态为“待审核”的记录。
3. **审核**：组织管理者修改 `activity_signups` 表中对应记录的状态（改为“通过”或“拒绝”）。

---

## 2. 数据库表结构设计 (MySQL)

### 2.1 活动主表 (`activities`)

**作用**：存储活动的具体内容、时间地点以及招募规则。

```sql
CREATE TABLE `activities` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `org_id` BIGINT UNSIGNED NOT NULL COMMENT '发布组织的ID (关联 organizations.id)',
  
  -- 核心展示信息
  `title` VARCHAR(100) NOT NULL COMMENT '活动标题',
  `content` TEXT COMMENT '活动详情描述',
  `location` VARCHAR(255) NOT NULL COMMENT '活动地点',
  
  -- 时间控制
  `start_time` DATETIME NOT NULL COMMENT '活动开始时间',
  `end_time` DATETIME NOT NULL COMMENT '活动结束时间',
  `recruit_end_time` DATETIME NOT NULL COMMENT '报名截止时间',
  
  -- 招募名额控制
  `recruit_count` INT UNSIGNED NOT NULL COMMENT '计划招募总人数',
  `signed_count` INT UNSIGNED DEFAULT '0' COMMENT '当前已通过审核的人数 (用于判断是否满员)',
  
  -- 活动状态
  `status` TINYINT UNSIGNED DEFAULT '1' COMMENT '状态: 0-草稿, 1-招募中, 2-进行中, 3-已结束, 4-已取消',
  
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  -- 索引优化
  KEY `idx_org_id` (`org_id`) COMMENT '查询某组织发布的所有活动',
  KEY `idx_status` (`status`) COMMENT '查询正在招募的活动'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='活动主表';

```

### 2.2 报名与审核表 (`activity_signups`)

**作用**：这是连接志愿者和活动的桥梁，同时承担了“审核记录表”的功能。它记录了谁报了名，以及组织批没批准。

```sql
CREATE TABLE `activity_signups` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `activity_id` BIGINT UNSIGNED NOT NULL COMMENT '关联活动ID',
  `volunteer_id` BIGINT UNSIGNED NOT NULL COMMENT '关联志愿者ID (volunteers.id)',
  
  -- 审核与流转状态 (核心字段)
  `status` TINYINT UNSIGNED DEFAULT '10' COMMENT '状态码: 10-待审核, 20-审核通过/待参加, 90-审核拒绝, 91-取消报名',
  
  -- 审核详情记录
  `audit_time` TIMESTAMP NULL COMMENT '审核操作时间',
  `audit_reason` VARCHAR(255) DEFAULT NULL COMMENT '审核意见 (如拒绝原因：人数已满/技能不符)',
  
  -- 记录时间
  `apply_time` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '志愿者发起报名的时间',

  -- 唯一性约束：防止同一个人重复报名同一个活动
  UNIQUE KEY `uk_act_vol` (`activity_id`, `volunteer_id`),
  -- 索引：方便组织方快速拉取“待审核”名单
  KEY `idx_act_status` (`activity_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='活动报名与审核表';

```

---

## 3. 核心状态字典说明

为了开发方便，请严格按照以下状态码编写业务逻辑：

### 活动状态 (`activities.status`)

| 状态码 | 含义 | 说明 |
| --- | --- | --- |
| **0** | 草稿 | 组织正在编辑，前端不可见 |
| **1** | **招募中** | 志愿者可以点击“报名”按钮 |
| **2** | 进行中 | 活动时间已到，停止报名 |
| **3** | 已结束 | 活动彻底完成 |

### 报名审核状态 (`activity_signups.status`)

| 状态码 | 含义 | 触发动作 |
| --- | --- | --- |
| **10** | **待审核** | 志愿者点击报名后，默认初始状态 |
| **20** | **审核通过** | 组织在后台点击“同意”，同时活动表 `signed_count` +1 |
| **90** | **审核拒绝** | 组织在后台点击“拒绝”，可填写 `audit_reason` |
| **91** | 已取消 | 志愿者主动撤回报名 |

---

## 4. 关键操作 SQL 示例

### 场景 A：志愿者发起报名

*检查活动是否在招募中，且名额未满，然后写入记录。*

```sql
INSERT INTO activity_signups (activity_id, volunteer_id, status)
VALUES (1001, 55, 10); -- 10 代表待审核

```

### 场景 B：组织管理者查看“待审核”名单

*查询该活动下所有状态为 10 的志愿者信息。*

```sql
SELECT s.id AS signup_id, v.real_name, v.mobile, s.apply_time
FROM activity_signups s
JOIN volunteers v ON s.volunteer_id = v.id
WHERE s.activity_id = 1001 AND s.status = 10;

```

### 场景 C：组织管理者“通过”审核

*更新状态，并记录时间。*

```sql
UPDATE activity_signups 
SET status = 20, audit_time = NOW() 
WHERE id = [报名记录ID];

-- 同时务必更新活动表的已报名人数
UPDATE activities 
SET signed_count = signed_count + 1 
WHERE id = 1001;

```

### 场景 D：组织管理者“拒绝”审核

```sql
UPDATE activity_signups 
SET status = 90, audit_time = NOW(), audit_reason = '抱歉，本次活动人数已满'
WHERE id = [报名记录ID];

```