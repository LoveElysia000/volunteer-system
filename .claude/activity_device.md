没问题。采用**“大厅（发现）”与“我的（记录）”分离**的方案，能让后端逻辑最清晰，也能完美适配前端不同的页面结构。

这份方案将接口拆分为 **A类（公共大厅）** 和 **B类（个人中心）**，重点解决你担心的“状态混乱”问题。

---

# 志愿者平台 - 活动接口分层设计方案

## 核心设计理念

1. **大厅接口 (Public Feed)**：只关注“未来”和“可参与性”。过滤掉历史垃圾数据，侧重于让用户**发现和报名**。
2. **个人接口 (My Records)**：关注“所有历史”和“执行状态”。包含已取消、审核中、缺席、已完成等复杂状态，侧重于**回顾和管理**。

---

## 🛠️ A类接口：活动大厅 (Public Activity Hall)

**场景**：首页“即将开始的活动”列表，或专门的“找活动”页面。
**原则**：只展示**发布中**且**未结束**的活动。

### 1. 获取可报名活动列表

* **URL**: `GET /api/v1/activities/public`
* **用途**：展示新鲜的、即将开始的活动。
* **筛选参数**：
* `category_id` (可选): 筛选类型


* **后端逻辑**：
1. `SELECT * FROM activities WHERE status = 'PUBLISHED' AND end_time > NOW()`
2. `LEFT JOIN signups` 判断当前用户是否报过名。



**响应数据结构 (JSON)**：
*重点看 `user_status` 字段，前端仅根据此字段渲染按钮。*

```json
{
  "list": [
    {
      "id": 101,
      "title": "社区垃圾分类指导",
      "start_time": "2024-02-01 09:00",
      "location": "朝阳社区",
      "quota": { "current": 28, "max": 30 },
      
      // ✅ 核心逻辑：后端计算出的最终展示状态
      // 枚举值: 
      // "AVAILABLE" (绿色按钮-立即报名)
      // "FULL"      (灰色按钮-名额已满)
      // "SIGNED"    (灰色/彩色按钮-已报名)
      "display_status": "AVAILABLE" 
    },
    {
      "id": 102,
      "title": "敬老院关爱行动",
      "start_time": "2024-02-02 10:00",
      "quota": { "current": 10, "max": 10 },
      "display_status": "FULL" // 虽然还没开始，但满了
    }
  ]
}

```

---

## 🛠️ B类接口：我的活动 (My Activities)

**场景**：点击个人中心“查看记录”或底部的“我的”Tab。
**原则**：展示**所有**我和系统产生过交互的活动，包括过去的、取消的。

### 2. 获取我的活动列表

* **URL**: `GET /api/v1/user/activities`
* **用途**：查看我的报名历史、工时记录。
* **筛选参数 (`tab`)**：
* `upcoming`: 待参加 (对应“我的日程”)
* `history`: 已结束/已完成 (对应“历史贡献”)



**响应数据结构 (JSON)**：
*这里的状态比大厅更详细，关注“执行结果”。*

```json
{
  "list": [
    {
      "activity_id": 101,
      "title": "社区垃圾分类指导",
      "start_time": "2024-02-01 09:00",
      "your_signup_time": "2024-01-25 14:30",
      
      // ✅ 核心逻辑：这里展示的是【报名记录表】的状态
      // 枚举值: 
      // "WAIT_ATTEND" (待参加 - 还没到时间)
      // "COMPLETED"   (已完成 - 成功获得工时)
      // "ABSENT"      (缺席 - 报名了没去)
      // "CANCELLED"   (已取消 - 我自己取消的)
      "audit_status": "WAIT_ATTEND",
      
      // 只有在 history tab 下且状态为 COMPLETED 时才有值
      "earned_rewards": {
        "points": 0,
        "hours": 0.0
      }
    },
    {
      "activity_id": 99,
      "title": "上个月的植树节",
      "start_time": "2024-01-01 09:00",
      "audit_status": "COMPLETED", // 已完成
      "earned_rewards": {
        "points": 10,
        "hours": 3.0
      }
    }
  ]
}

```

---

## 🧠 后端状态映射表 (The Mapping Logic)

这是最关键的部分。为了不让前端晕头转向，后端需要按照下面的表格来赋值。

### 1. 大厅接口 (`display_status`) 的计算规则

*针对 `GET /activities/public*`

| 优先级 | 条件判断 (后端伪代码) | 返回给前端的状态值 | 前端按钮样式 |
| --- | --- | --- | --- |
| 1 (高) | 用户在 `signups` 表中有记录 且未取消 | **SIGNED** | 灰色文案“已报名” |
| 2 | `current_people >= max_people` | **FULL** | 禁用按钮“名额已满” |
| 3 (低) | 其他情况 | **AVAILABLE** | 绿色按钮“立即报名” |

### 2. 我的接口 (`audit_status`) 的计算规则

*针对 `GET /user/activities*`

| 数据库 `signups.status` | 数据库 `signups.check_in` | 活动是否结束 | 返回给前端的状态值 | UI 展示文案 |
| --- | --- | --- | --- | --- |
| CANCELLED (已取消) | - | - | **CANCELLED** | 已取消 |
| SUCCESS (正常) | 否 (未签到) | 是 (已过期) | **ABSENT** | 缺席/未签到 |
| SUCCESS (正常) | 是 (已签到) | 是 | **COMPLETED** | ✅ 已完成 |
| SUCCESS (正常) | 否 | 否 (未来) | **WAIT_ATTEND** | ⏳ 待参加 |

---

## 💡 为什么这么拆分更好？

1. **性能优化**：
* **大厅接口**可以做缓存（Redis），因为它对大部分人来说内容是一样的（只要去掉 `is_signed` 状态或者把 `is_signed` 做成单独的小接口）。
* **我的接口**不能缓存，但访问频率低。


2. **前端逻辑极简**：
* 前端做大厅时，不用写 `if (status == 'CANCELLED')`，因为后端压根就不把取消的活动发给大厅。
* 前端做“我的记录”时，不用管“名额满没满”，因为那已经是过去式了。


3. **扩展性**：
* 如果未来你要加一个“审核中”的状态（比如报名需要管理员批准），只需要修改 **B类接口** 的 `audit_status`，完全不影响大厅的展示。



### 建议的下一步

你可以把这个方案发给前端，告诉他：“首页用 `/public` 接口，渲染按钮只看 `display_status`；个人中心用 `/user/activities` 接口，渲染状态标签看 `audit_status`。” 这样沟通效率最高。