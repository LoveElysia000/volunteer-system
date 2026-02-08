
---

# 志愿者信息变更审核 - 数据库设计方案

**版本**：V1.0
**设计目标**：实现对敏感信息修改的审批流，确保主档案数据的真实性，并保留完整的变更历史以便追溯。

## 1. 核心设计思路

### 1.1 为什么不能直接修改主表？

如果在审核通过前就直接修改 `volunteers` 表，会导致“非法数据”在前台展示。例如：用户乱填了一个名字，管理员还没看，页面上就已经显示这个乱填的名字了。

### 1.2 解决方案：读写分离策略

* **读数据（Read）**：前端展示永远读取 `volunteers` 主表（始终展示当前合法的旧数据）。
* **写数据（Write）**：用户修改申请写入 `volunteer_audits` 审核表（暂存新数据）。
* **同步（Sync）**：只有审核通过的瞬间，系统才将审核表中的数据覆盖到主表。

### 1.3 关键特性：双重快照 (Before & After)

为了辅助管理员决策和防止误操作，每一条审核记录必须同时保存：

* **变更前数据 (`before_json`)**：用于对比和回滚。
* **变更后数据 (`after_json`)**：用户的修改目标。

---

## 2. 数据库表结构 (SQL)

请执行以下 SQL 建立审核流水表。

```sql
CREATE TABLE `volunteer_audits` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `volunteer_id` BIGINT UNSIGNED NOT NULL COMMENT '申请人ID (关联 volunteers.id)',
  
  -- ==========================================
  -- 1. 核心数据区 (采用 JSON 存储以适应不同字段的修改)
  -- ==========================================
  `before_json` JSON DEFAULT NULL COMMENT '变更前的数据快照 (用于对比/回滚)',
  `after_json` JSON NOT NULL COMMENT '变更后的新数据 (拟修改内容)',
  
  -- 示例数据结构:
  -- before_json: {"real_name": "张三", "mobile": "13800000000"}
  -- after_json:  {"real_name": "张三丰", "mobile": "13911111111"}
  
  -- ==========================================
  -- 2. 审核流程状态
  -- ==========================================
  `status` TINYINT UNSIGNED DEFAULT '0' COMMENT '状态: 0-待审核, 1-已通过, 2-已驳回, 3-已撤销',
  `reject_reason` VARCHAR(255) DEFAULT NULL COMMENT '驳回原因 (仅在 status=2 时有值)',
  
  -- ==========================================
  -- 3. 审计追踪
  -- ==========================================
  `auditor_id` BIGINT UNSIGNED DEFAULT NULL COMMENT '操作人/管理员ID',
  `audit_time` TIMESTAMP NULL COMMENT '审核完成时间',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '申请提交时间',
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  -- ==========================================
  -- 4. 索引优化
  -- ==========================================
  KEY `idx_volunteer` (`volunteer_id`) COMMENT '查询某人的修改历史',
  KEY `idx_status` (`status`) COMMENT '后台查询待审核列表'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='志愿者信息变更审核流水表';

```

---

## 3. 业务流程与数据流转图解

### 阶段一：提交申请 (User Submit)

用户在前端修改个人信息（例如改名）。

1. **后端动作**：
* 查询 `volunteers` 表当前数据，存入 `before_json`。
* 获取前端提交的新数据，存入 `after_json`。
* **SQL 操作**：


```sql
INSERT INTO volunteer_audits (volunteer_id, before_json, after_json, status)
VALUES (55, '{"real_name": "张三"}', '{"real_name": "张三丰"}', 0);

```


2. **结果**：主表数据未变，审核表增加一条 `status=0` 的记录。

### 阶段二：管理员审核 (Admin Review)

管理员在后台看到一条申请。

* **界面展示**：
* 原值：张三
* 新值：**张三丰**
* *管理员心理活动*：“哦，这是合理的改名，不是乱填。” -> **点击【通过】**



### 阶段三：数据生效 (Apply Changes)

这是最关键的一步，必须使用**数据库事务 (`Transaction`)** 保证原子性。

* **后端逻辑**：
1. 解析 `after_json` 字段。
2. 更新 `volunteers` 主表。
3. 更新 `volunteer_audits` 状态为“已通过”。


* **SQL 操作 (事务中执行)**：
```sql
-- 1. 更新主表
UPDATE volunteers SET real_name = '张三丰' WHERE id = 55;

-- 2. 完结审核单
UPDATE volunteer_audits 
SET status = 1, auditor_id = 999, audit_time = NOW() 
WHERE id = 2024;

```



---

## 4. 方案优势总结

| 维度 | 方案优势 |
| --- | --- |
| **灵活性** | **JSON字段**意味着如果你明天想审核“性别”或“民族”，不需要修改数据库表结构，代码改一下即可。 |
| **安全性** | **Before/After快照**让管理员能清晰对比变化，防止恶意篡改；同时也提供了数据回滚的能力。 |
| **业务连续性** | 在审核期间，志愿者依然可以使用旧身份正常报名活动，不会因为“审核中”而被系统锁定。 |
| **审计合规** | 每一笔修改都有迹可循（谁申请的、改了什么、谁批的、什么时候批的），符合政务/公益平台的合规要求。 |

## 5. 开发建议

1. **锁定机制**：建议限制用户在“待审核”状态下不能再次提交新的申请。即：`SELECT count(1) FROM volunteer_audits WHERE volunteer_id = ? AND status = 0`，如果有值，提示用户“您有正在审核中的申请，请耐心等待”。
2. **自动驳回**：如果用户改的新值和旧值一模一样（before == after），后端可以直接拦截，或者自动审核通过（视业务而定）。