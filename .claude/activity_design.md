# 志愿者活动模块设计方案

## 一、概述

基于现有项目架构（Go + Hertz + GORM + Protobuf），设计活动管理模块，实现活动发布、浏览、报名等核心功能。

---

## 二、数据库设计

### 2.1 活动主表 (`activities`)

```sql
CREATE TABLE `activities` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `org_id` BIGINT UNSIGNED NOT NULL DEFAULT '' COMMENT '发布组织ID (关联organizations.id)',

  -- 活动基本信息
  `title` VARCHAR(100) NOT NULL COMMENT '活动标题',
  `description` TEXT COMMENT '活动描述/副标题',
  `cover_url` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '活动封面图URL',

  -- 时间地点
  `start_time` DATETIME NOT NULL COMMENT '开始时间',
  `end_time` DATETIME NOT NULL COMMENT '结束时间',
  `location` VARCHAR(100) COMMENT '地点名称',
  `address` VARCHAR(255) COMMENT '详细地址',

  -- 招募信息
  `duration` DECIMAL(4, 1) NOT NULL DEFAULT '0.0' COMMENT '预估工时(小时)',
  `max_people` INT NOT NULL DEFAULT '0' COMMENT '最大招募人数 (0表示不限)',
  `current_people` INT NOT NULL DEFAULT '0' COMMENT '当前已报名人数(冗余字段)',

  -- 状态
  `status` TINYINT NOT NULL DEFAULT '1' COMMENT '状态: 1-报名中, 2-已结束, 3-已取消',

  -- 系统元数据
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL COMMENT '软删除时间',

  PRIMARY KEY (`id`),
  KEY `idx_org_id` (`org_id`) COMMENT '组织ID索引',
  KEY `idx_status` (`status`) COMMENT '状态索引',
  KEY `idx_start_time` (`start_time`) COMMENT '开始时间索引',
  KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='活动主表';
```

### 2.2 报名记录表 (`activity_signups`)

```sql
CREATE TABLE `activity_signups` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `activity_id` BIGINT UNSIGNED NOT NULL COMMENT '活动ID (关联activities.id)',
  `volunteer_id` BIGINT UNSIGNED NOT NULL COMMENT '志愿者ID (关联volunteers.id)',

  -- 报名信息
  `signup_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '报名时间',
  `status` TINYINT NOT NULL DEFAULT '1' COMMENT '状态: 1-已报名, 2-已取消',

  -- 系统元数据
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_act_vol` (`activity_id`, `volunteer_id`) COMMENT '防止重复报名',
  KEY `idx_activity_id` (`activity_id`) COMMENT '活动ID索引',
  KEY `idx_volunteer_id` (`volunteer_id`) COMMENT '志愿者ID索引',
  KEY `idx_status` (`status`) COMMENT '状态索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='活动报名记录表';
```

---

## 三、API 设计 (Protobuf)

### 3.1 文件位置
`internal/api/activity.proto`

### 3.2 Proto 定义

```proto
syntax = "proto3";

package activity;

import "google/api/annotations.proto";
import "google/api/client.proto";

option go_package = "volunteer-system/internal/api;api";

// 活动管理服务端接口
service ActivityService {
  option (google.api.default_host) = "0.0.0.0:8080";

  // 获取活动列表
  rpc ActivityList(ActivityListRequest) returns (ActivityListResponse) {
    option (google.api.http) = {
      get: "/api/v1/activities"
    };
  }

  // 活动报名
  rpc ActivitySignup(ActivitySignupRequest) returns (ActivitySignupResponse) {
    option (google.api.http) = {
      post: "/api/v1/activities/signup"
      body: "*"
    };
  }

  // 取消报名
  rpc ActivityCancel(ActivityCancelRequest) returns (ActivityCancelResponse) {
    option (google.api.http) = {
      post: "/api/v1/activities/cancel"
      body: "*"
    };
  }

  // 获取活动详情
  rpc ActivityDetail(ActivityDetailRequest) returns (ActivityDetailResponse) {
    option (google.api.http) = {
      get: "/api/v1/activities/:id"
    };
  }
}

// ========== 活动列表 ==========

message ActivityListRequest {
  // 页码 可选 @gotags: query:"page"
  int32 page = 1;
  // 页大小 可选 @gotags: query:"pageSize"
  int32 pageSize = 2;
  // 状态筛选 可选 @gotags: query:"status"
  int32 status = 3;
}

message ActivityListResponse {
  int32 total = 1;
  repeated ActivityItem list = 2;
}

message ActivityItem {
  int64 id = 1;
  // 活动标题
  string title = 2;
  // 活动描述
  string description = 3;
  // 封面图URL
  string coverUrl = 4;
  // 开始时间
  string startTime = 5;
  // 结束时间
  string endTime = 6;
  // 地点名称
  string location = 7;
  // 预估工时
  double duration = 8;
  // 最大招募人数
  int32 maxPeople = 9;
  // 当前已报名人数
  int32 currentPeople = 10;
  // 状态: 1-报名中, 2-已结束, 3-已取消
  int32 status = 11;
  // 是否已报名 (当前用户)
  bool isRegistered = 12;
  // 是否已满员
  bool isFull = 13;
}

// ========== 活动报名 ==========

message ActivitySignupRequest {
  // 活动ID 必填 @gotags: json:"activityId,required"
  int64 activityId = 1;
}

message ActivitySignupResponse {
  // 报名成功
  bool success = 1;
}

// ========== 取消报名 ==========

message ActivityCancelRequest {
  // 活动ID 必填 @gotags: json:"activityId,required"
  int64 activityId = 1;
}

message ActivityCancelResponse {
  // 取消成功
  bool success = 1;
}

// ========== 活动详情 ==========

message ActivityDetailRequest {
  // 活动ID 必填 @gotags: path:"id,required"
  int64 id = 1;
}

message ActivityDetailResponse {
  ActivityInfo activity = 1;
}

message ActivityInfo {
  int64 id = 1;
  // 组织ID
  int64 orgId = 2;
  // 组织名称
  string orgName = 3;
  // 活动标题
  string title = 4;
  // 活动描述
  string description = 5;
  // 封面图URL
  string coverUrl = 6;
  // 开始时间
  string startTime = 7;
  // 结束时间
  string endTime = 8;
  // 地点名称
  string location = 9;
  // 详细地址
  string address = 10;
  // 预估工时
  double duration = 11;
  // 最大招募人数
  int32 maxPeople = 12;
  // 当前已报名人数
  int32 currentPeople = 13;
  // 状态: 1-报名中, 2-已结束, 3-已取消
  int32 status = 14;
  // 是否已报名 (当前用户)
  bool isRegistered = 15;
  // 创建时间
  string createdAt = 16;
}
```

---

## 四、代码结构

```
volunteer-system/
├── internal/
│   ├── api/
│   │   └── activity.proto          # 新增：活动API定义
│   ├── dao/                        # GORM Gen 生成
│   │   ├── activities.gen.go       # 新增：活动DAO
│   │   └── activity_signups.gen.go # 新增：报名DAO
│   ├── model/                      # GORM Gen 生成
│   │   ├── activities.gen.go       # 新增：活动Model
│   │   └── activity_signups.gen.go # 新增：报名Model
│   ├── handler/
│   │   └── activity.go             # 新增：活动Handler
│   ├── repository/
│   │   └── activity.go             # 新增：活动Repository
│   ├── router/
│   │   └── activity.go             # 新增：活动Router
│   └── service/
│       └── activity.go             # 新增：活动Service
├── sql/
│   └── ddl/
│       └── activity.sql            # 新增：活动表DDL
└── gen.yaml                        # 修改：添加活动表
```

---

## 五、实现步骤

### 步骤 1: 创建数据库表
执行 `sql/ddl/activity.sql` 创建 `activities` 和 `activity_signups` 表。

### 步骤 2: 更新 GORM Gen 配置
修改 `gen.yaml`，添加活动相关表：
```yaml
tables:
  - "sys_accounts"
  - "volunteers"
  - "organizations"
  - "activities"        # 新增
  - "activity_signups"  # 新增
```

### 步骤 3: 生成 Model 和 DAO
运行 GORM Gen 生成代码：
```bash
go run cmd/gen/main.go
```

### 步骤 4: 创建 Proto 文件
创建 `internal/api/activity.proto`，定义 API 接口。

### 步骤 5: 生成 Proto 代码
运行 protoc 生成 `.pb.go` 文件：
```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       internal/api/activity.proto
```

### 步骤 6: 实现 Repository 层
创建 `internal/repository/activity.go`，实现数据访问方法：
- `FindActivitiesByStatus(status int32, page, pageSize int)`
- `FindActivityByID(id int64)`
- `CreateActivity(activity *model.Activity)`
- `FindSignup(activityID, volunteerID int64)`
- `CreateSignup(signup *model.ActivitySignup)`
- `UpdateSignupStatus(signup *model.ActivitySignup)`
- `IncrementActivityPeople(activityID int64)`

### 步骤 7: 实现 Service 层
创建 `internal/service/activity.go`，实现业务逻辑：
- `ActivityList()` - 查询活动列表，包含当前用户报名状态
- `ActivitySignup()` - 报名校验（名额、重复）+ 事务处理
- `ActivityCancel()` - 取消报名 + 更新人数
- `ActivityDetail()` - 查询活动详情

### 步骤 8: 实现 Handler 层
创建 `internal/handler/activity.go`，实现 HTTP 处理器：
- `ActivityList()`
- `ActivitySignup()`
- `ActivityCancel()`
- `ActivityDetail()`

### 步骤 9: 实现 Router 层
创建 `internal/router/activity.go`，注册路由：
```go
func RegisterActivityRouter(r *route.RouterGroup) {
    r.GET("/v1/activities", handler.ActivityList)
    r.POST("/v1/activities/signup", handler.ActivitySignup)
    r.POST("/v1/activities/cancel", handler.ActivityCancel)
    r.GET("/v1/activities/:id", handler.ActivityDetail)
}
```

### 步骤 10: 注册路由
修改 `internal/router/router.go`，添加活动路由注册：
```go
authApi := api.Group("", middleware.Auth())
RegisterVolunteerRouter(authApi)
RegisterOrganizationRouter(authApi)
RegisterActivityRouter(authApi)  // 新增
```

---

## 六、关键业务逻辑

### 6.1 活动列表 - 动态判断用户报名状态

```go
// 查询活动列表时，需要判断当前用户是否已报名
func (s *ActivityService) ActivityList(req *api.ActivityListRequest) (*api.ActivityListResponse, error) {
    // 1. 获取当前用户ID (从JWT中获取)
    userID := middleware.GetUserIDInt(s.c)

    // 2. 查询活动列表
    activities, err := s.repo.FindActivitiesByStatus(req.Status, req.Page, req.PageSize)

    // 3. 查询当前用户的报名记录
    signupMap := s.repo.FindUserSignupMap(userID, activityIDs)

    // 4. 组装返回数据，设置 isRegistered 和 isFull
    for _, act := range activities {
        item := &api.ActivityItem{
            Id:           act.ID,
            Title:        act.Title,
            // ... 其他字段
            IsRegistered: signupMap[act.ID] != nil,
            IsFull:       act.CurrentPeople >= act.MaxPeople,
        }
        resp.List = append(resp.List, item)
    }
}
```

### 6.2 活动报名 - 事务处理

```go
func (s *ActivityService) ActivitySignup(req *api.ActivitySignupRequest) (*api.ActivitySignupResponse, error) {
    userID := middleware.GetUserIDInt(s.c)

    // 1. 查询活动信息
    activity, err := s.repo.FindActivityByID(req.ActivityId)
    if err != nil {
        return nil, errors.New("活动不存在")
    }

    // 2. 校验活动状态
    if activity.Status != 1 {
        return nil, errors.New("活动已结束或已取消")
    }

    // 3. 校验名额
    if activity.MaxPeople > 0 && activity.CurrentPeople >= activity.MaxPeople {
        return nil, errors.New("名额已满")
    }

    // 4. 校验是否重复报名
    existing, _ := s.repo.FindSignup(req.ActivityId, userID)
    if existing != nil {
        return nil, errors.New("请勿重复报名")
    }

    // 5. 事务处理
    err = s.repo.Transaction(func(tx *gorm.DB) error {
        // 插入报名记录
        signup := &model.ActivitySignup{
            ActivityID:  req.ActivityId,
            VolunteerID: userID,
            Status:      1,
        }
        if err := tx.Create(signup).Error; err != nil {
            return err
        }

        // 更新活动人数 (使用原子操作防止并发问题)
        result := tx.Model(&model.Activity{}).
            Where("id = ? AND current_people < max_people", req.ActivityId).
            Update("current_people", gorm.Expr("current_people + 1"))
        if result.RowsAffected == 0 {
            return errors.New("名额已满")
        }

        return nil
    })

    return &api.ActivitySignupResponse{Success: true}, nil
}
```

---

## 七、错误码定义

在 `internal/response/errors.go` 中添加活动相关错误码：

```go
const (
    // ... 现有错误码

    // 活动相关错误码 (2000-2099)
    ErrActivityNotFound     = 2001 // 活动不存在
    ErrActivityEnded        = 2002 // 活动已结束
    ErrActivityFull         = 2003 // 名额已满
    ErrDuplicateSignup      = 2004 // 重复报名
    ErrSignupNotFound       = 2005 // 报名记录不存在
    ErrCancelNotAllowed     = 2006 // 不允许取消报名
)
```

---

## 八、测试验证

### 8.1 API 测试

```bash
# 1. 获取活动列表
GET /api/v1/activities?page=1&pageSize=10&status=1

# 2. 活动报名
POST /api/v1/activities/signup
Authorization: Bearer <token>
{
  "activityId": 1
}

# 3. 取消报名
POST /api/v1/activities/cancel
Authorization: Bearer <token>
{
  "activityId": 1
}

# 4. 获取活动详情
GET /api/v1/activities/1
Authorization: Bearer <token>
```

### 8.2 并发测试

使用 Apache Bench 或 JMeter 测试报名接口的并发安全性，确保不会超卖。

---

## 九、后续扩展

### 阶段二：状态联动与业务逻辑完善
- 添加时间冲突检测
- 完善报名校验逻辑

### 阶段三：统计分析
- 新增 `records` 流水记录表
- 新增 `level_rules` 等级配置表
- 实现本月新增统计
- 实现等级进度计算

### 阶段四：高并发保障
- 使用 Redis 缓存活动列表
- 实现定时任务自动归档过期活动