# 环保志愿者服务平台设计文档

## 1. 项目概述

### 1.1 项目背景和目标
构建一个连接环保志愿者与环保活动的平台，促进环保志愿服务的社会化、规范化管理，提升环保活动的参与度和影响力。

### 1.2 技术架构概览
构建采用前后端分离架构，前端使用Vue.js + Vite + TypeScript技术栈，后端采用Go + Hertz框架，数据库使用MySQL，具备完整的API接口和数据管理系统。

### 1.3 核心功能特色
- 志愿者注册和活动管理
- 积分激励系统
- 组织管理体系
- 数据导入导出功能
- 地理位置服务集成

## 2. 技术架构设计

### 2.1 整体架构图
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   前端层         │    │   后端API层      │    │   数据层         │
│                 │    │                 │    │                 │
│ Vue.js 3.0      │    │ Go + Hertz      │    │ MySQL数据库      │
│ Element Plus    │◄───│ Protobuf API    │◄───│ Redis缓存        │
│ Vite构建        │    │ GORM ORM        │    │ 文件存储         │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         │                        │                        │
         ▼                        ▼                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   用户浏览器     │    │   负载均衡       │    │   监控系统       │
│ Chrome/Safari   │    │ 反向代理         │    │ 日志收集         │
│ Firefox/移动端  │    │ 安全防护         │    │ 性能监控         │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 2.2 技术栈选择

#### 后端技术栈
- **编程语言**：Go 1.21+
- **Web框架**：Hertz (高性能HTTP框架)
- **ORM框架**：Gorm (数据库操作)
- **数据库**：MySQL 8.0+
- **缓存**：Redis (可选，用于会话缓存)
- **数据传输**：Protobuf (高效序列化)

#### 前端技术栈
- **框架**：Vue.js 3.0 + TypeScript
- **样式框架**：Tailwind CSS (基于Tailwind构建自定义组件)
- **构建工具**：Vite
- **状态管理**：Pinia
- **路由**：Vue Router
- **HTTP客户端**：Axios
- **图标库**：Lucide Vue (轻量级图标组件)
- **日期处理**：date-fns (日期格式化工具)
- **表单验证**：VeeValidate (表单验证库)
- **工具函数**：Lodash (实用工具函数库)
- **动画库**：VueUse (Vue组合式API工具集)

### 1.3 系统组件设计

#### 认证授权组件
```go
type AuthComponent struct {
    JWTService      // JWT令牌管理
    PermissionService // 权限验证
    SessionManager  // 会话管理
}
```

#### 核心业务组件
- 用户管理组件
- 活动管理组件
- 积分管理组件
- 组织管理组件
- 证书生成组件
- 数据导入导出组件

### 1.4 项目目录结构

```
volunteer-system/
├── backend/                   # 后端服务
│   ├── cmd/                   # 应用入口
│   │   └── server/            # 主服务入口
│   ├── internal/              # 内部包（不对外暴露）
│   │   ├── api/               # 前后端数据传输结构定义层
│   │   │   ├── user.proto     # 用户相关数据结构
│   │   │   ├── activity.proto # 活动相关数据结构
│   │   │   ├── points.proto   # 积分相关数据结构
│   │   │   ├── common.proto   # 通用数据结构
│   │   │   ├── user.pb.go     # protobuf生成的用户相关代码
│   │   │   ├── activity.pb.go # protobuf生成的活动相关代码
│   │   │   ├── points.pb.go   # protobuf生成的积分相关代码
│   │   │   └── common.pb.go   # protobuf生成的通用代码
│   │   ├── router/            # 路由目录
│   │   │   ├── user.go        # 用户路由
│   │   │   ├── activity.go    # 活动路由
│   │   │   └── auth.go        # 认证路由
│   │   │   └── router.go      # 注册总路由
│   │   ├── handler/           # 处理函数目录
│   │   │   ├── user_handler.go    # 用户处理函数
│   │   │   ├── activity_handler.go # 活动处理函数
│   │   │   └── auth_handler.go    # 认证处理函数
│   │   ├── service/           # 服务层
│   │   │   ├── user_service.go    # 用户服务
│   │   │   ├── activity_service.go # 活动服务
│   │   │   └── points_service.go  # 积分服务
│   │   │   └── service.go        # 通用服务
│   │   ├── repository/        # 数据操作层
│   │   │   ├── user_repo.go       # 用户数据操作
│   │   │   ├── activity_repo.go   # 活动数据操作
│   │   │   └── points_repo.go     # 积分数据操作
│   │   │   └── repository.go        # 通用数据访问层
│   │   ├── response/          # 响应层
│   │   │   ├── response.go        # 通用响应结构
│   │   ├── config/            # 配置管理
│   │   ├── middleware/        # 中间件
│   │   ├── model/             # 数据模型层
│   │   │   ├── user.go        # 用户模型
│   │   │   ├── activity.go    # 活动模型
│   │   │   ├── points.go      # 积分模型
│   │   │   ├── organization.go # 组织模型
│   │   │   └── base.go        # 基础模型
│   ├── pkg/                   # 可复用的公共包
│   │   ├── auth/              # 认证授权
│   │   ├── database/          # 数据库连接
│   │   ├── logger/            # 日志处理
│   │   ├── util/              # 工具函数
│   │   │   ├── time.go        # 时间处理工具
│   │   │   ├── crypto.go      # 加密工具
│   │   │   ├── string.go      # 字符串工具
│   │   │   └── file.go        # 文件处理工具
│   │   └── validator/         # 参数验证
│   ├── script/                # 脚本文件
│   │   ├── migration/         # 数据库迁移
│   │   └── deploy/            # 部署脚本
│   ├── test/                  # 测试文件
│   ├── go.mod                 # Go模块定义
│   └── go.sum                 # 依赖校验
├── frontend/                  # 前端应用
│   ├── public/                # 静态资源
│   ├── src/                   # 源代码
│   │   ├── api/               # API接口
│   │   ├── assets/            # 静态资源
│   │   ├── components/        # 通用组件
│   │   ├── pages/             # 页面组件
│   │   ├── router/            # 路由配置
│   │   ├── store/             # 状态管理
│   │   ├── utils/             # 工具函数
│   │   └── views/             # 视图组件
│   ├── package.json           # 依赖管理
│   ├── vite.config.js         # Vite配置
│   └── tailwind.config.js     # Tailwind配置
├── docs/                      # 文档
│   ├── api/                   # API文档
│   ├── database/              # 数据库文档
│   └── deployment/            # 部署文档
├── docker/                    # Docker配置
│   ├── backend/               # 后端Dockerfile
│   ├── frontend/              # 前端Dockerfile
│   └── docker-compose.yml     # 容器编排
├── config/                    # 配置文件
│   ├── dev/                   # 开发环境配置
│   ├── test/                  # 测试环境配置
│   └── prod/                  # 生产环境配置
├── script/                    # 项目脚本
│   ├── build.sh               # 构建脚本
│   ├── deploy.sh              # 部署脚本
│   └── test.sh                # 测试脚本
├── README.md                  # 项目说明
├── Makefile                   # 命令自动化集成
└── .gitignore                 # Git忽略文件
```

#### 后端目录说明
- **cmd/server**: 应用启动入口，包含main函数
- **internal**: 内部私有包，不对外暴露接口
  - **api**: 前后端数据传输结构定义层，使用protobuf定义数据结构
  - **router**: 路由定义，处理HTTP请求路由
  - **handler**: 处理函数，接收请求并调用服务层
  - **service**: 业务逻辑层，处理核心业务逻辑
  - **repository**: 数据操作层，与数据库交互
  - **response**: 响应层，统一响应格式
  - **model**: 数据模型层，定义数据库表结构对应的Go结构体
- **pkg**: 可复用的公共组件包
  - **auth**: 认证授权相关工具
  - **database**: 数据库连接和操作
  - **logger**: 日志处理
  - **util**: 通用工具函数（时间、加密、字符串、文件处理等）
  - **validator**: 参数验证

#### 前端目录说明
- **src/api**: API接口封装和请求处理，包含protobuf消息的序列化/反序列化
- **src/components**: 可复用的UI组件
- **src/pages**: 页面级组件
- **src/store**: Vuex/Pinia状态管理

#### Protobuf数据传输流程
```
前端请求 → protobuf序列化 → HTTP请求 → 后端接收 → protobuf反序列化 → handler处理
后端响应 → protobuf序列化 → HTTP响应 → 前端接收 → protobuf反序列化 → 页面渲染
```

#### Makefile常用命令
```makefile
# 开发相关
make dev           # 启动开发环境
make build         # 构建项目
make test          # 运行测试
make clean         # 清理构建文件

# 数据库相关
make model         # 使用gentools生成数据库模型文件

# Protobuf相关
make api-single    # 生成单个proto文件

# 部署相关
make docker-build  # 构建Docker镜像
make docker-run    # 运行Docker容器
make deploy        # 部署到生产环境
```

#### 代码生成流程
```
# 生成数据库模型
make model → 使用gentools根据数据库表结构生成Go模型文件

# 生成protobuf定义
make proto-gen → 根据业务需求创建proto文件模板

# 生成protobuf代码
make proto → 使用protoc编译proto文件生成Go代码

# 一键生成所有
make gen → 执行model + proto-gen + proto
```

## 2. 数据库设计

### 2.1 实体关系图(ERD)
```sql
-- 核心实体关系
用户 (User) ──┬── 志愿服务记录 (VolunteerRecord)
             ├── 活动报名 (ActivityRegistration)
             └── 积分记录 (PointsRecord)

活动 (Activity) ──┬── 活动报名 (ActivityRegistration)
                 ├── 活动成果 (ActivityResult)
                 └── 组织 (Organization)

组织 (Organization) ──┬── 用户 (组织成员)
                     └── 活动 (组织活动)
```

### 2.2 核心数据表结构

#### 用户表 (users)
```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    phone VARCHAR(20),
    real_name VARCHAR(50) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('volunteer', 'organization', 'admin') NOT NULL,
    avatar_url VARCHAR(255),
    skills JSON, -- 技能标签
    total_hours INT DEFAULT 0, -- 总服务时长
    points INT DEFAULT 0, -- 积分
    level INT DEFAULT 1, -- 等级
    status ENUM('active', 'inactive', 'pending') DEFAULT 'active',
    organization_id BIGINT, -- 所属组织
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (organization_id) REFERENCES organizations(id)
);
```

#### 活动表 (activities)
```sql
CREATE TABLE activities (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    type ENUM('cleaning', 'planting', 'education', 'other') NOT NULL,
    location VARCHAR(255) NOT NULL,
    latitude DECIMAL(10, 8), -- 纬度
    longitude DECIMAL(11, 8), -- 经度
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    max_participants INT,
    current_participants INT DEFAULT 0,
    status ENUM('draft', 'published', 'ongoing', 'completed', 'cancelled') DEFAULT 'draft',
    organizer_id BIGINT NOT NULL, -- 组织者
    contact_info JSON, -- 联系信息
    requirements TEXT, -- 活动要求
    points_reward INT DEFAULT 0, -- 积分奖励
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (organizer_id) REFERENCES users(id)
);
```

#### 积分记录表 (points_records)
```sql
CREATE TABLE points_records (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    activity_id BIGINT,
    points INT NOT NULL,
    type ENUM('activity', 'system', 'redeem') NOT NULL,
    description VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (activity_id) REFERENCES activities(id)
);
```

### 2.3 索引设计
```sql
-- 用户表索引
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_organization ON users(organization_id);
CREATE INDEX idx_users_points ON users(points);

-- 活动表索引
CREATE INDEX idx_activities_organizer ON activities(organizer_id);
CREATE INDEX idx_activities_time ON activities(start_time, end_time);
CREATE INDEX idx_activities_location ON activities(latitude, longitude);
CREATE INDEX idx_activities_status ON activities(status);

-- 积分记录表索引
CREATE INDEX idx_points_user ON points_records(user_id);
CREATE INDEX idx_points_time ON points_records(created_at);
```

## 3. API设计

### 3.1 RESTful API规范

#### 认证相关接口
```
POST   /api/auth/login          # 用户登录
POST   /api/auth/register       # 用户注册
POST   /api/auth/logout         # 用户登出
POST   /api/auth/refresh        # 刷新令牌
POST   /api/auth/reset-password # 重置密码
```

#### 用户相关接口
```
GET    /api/users/profile       # 获取用户信息
PUT    /api/users/profile       # 更新用户信息
GET    /api/users/activities    # 获取用户参与的活动
GET    /api/users/points        # 获取用户积分记录
```

#### 活动相关接口
```
GET    /api/activities          # 获取活动列表
POST   /api/activities          # 创建活动
GET    /api/activities/{id}     # 获取活动详情
PUT    /api/activities/{id}     # 更新活动
DELETE /api/activities/{id}     # 删除活动
POST   /api/activities/{id}/register  # 报名活动
```

### 3.2 API响应格式
```json
{
    "code": 200,
    "message": "success",
    "data": {
        "user": {
            "id": 1,
            "username": "volunteer1",
            "email": "user@example.com"
        }
    },
    "timestamp": "2026-01-10T10:00:00Z"
}
```

### 3.3 错误处理
```json
{
    "code": 400,
    "message": "参数验证失败",
    "errors": [
        {
            "field": "email",
            "message": "邮箱格式不正确"
        }
    ],
    "timestamp": "2026-01-10T10:00:00Z"
}
```

## 4. 前端UI设计思路

### 4.1 整体布局设计
```
┌─────────────────────────────────────────┐
│             头部导航栏                    │
├─────────────────────────────────────────┤
│ 侧边栏 │          主内容区域              │
│       │                                  │
│ 菜单  │          页面内容                │
│       │                                  │
└─────────────────────────────────────────┘
```

### 4.2 核心页面设计

#### 首页设计
- 活动推荐轮播图
- 热门活动列表
- 快速报名入口
- 统计数据展示

#### 活动详情页
- 活动基本信息展示
- 参与人员列表
- 地图位置显示
- 报名按钮和状态

#### 个人中心
- 个人信息管理
- 活动参与记录
- 积分和等级展示
- 证书下载

### 4.3 组件设计规范

#### 通用组件
- 按钮组件 (Button)
- 表单组件 (Form, Input, Select)
- 表格组件 (Table)
- 弹窗组件 (Modal)
- 提示组件 (Toast)

#### 业务组件
- 活动卡片 (ActivityCard)
- 用户头像 (UserAvatar)
- 积分显示 (PointsDisplay)
- 地图组件 (MapView)

## 4.4 前端搭建设计

### 4.4.1 项目初始化流程

#### 技术选型确认
- **核心框架**: Vue.js 3.0 + TypeScript
- **构建工具**: Vite (快速构建和热重载)
- **样式框架**: Tailwind CSS (原子化CSS框架，基于Tailwind构建自定义组件)
- **状态管理**: Pinia (轻量级状态管理)
- **路由管理**: Vue Router (单页面应用路由)
- **HTTP客户端**: Axios (HTTP请求库)
- **图标库**: Lucide Vue (轻量级图标组件)
- **日期处理**: date-fns (日期格式化工具)
- **表单验证**: VeeValidate (表单验证库)
- **工具函数**: Lodash (实用工具函数库)
- **动画库**: VueUse (Vue组合式API工具集)

#### 项目结构规划
```
frontend/
├── public/           # 静态资源
├── src/
│   ├── api/          # API接口封装
│   ├── assets/       # 静态资源
│   ├── components/   # 通用组件
│   ├── pages/        # 页面组件
│   ├── router/       # 路由配置
│   ├── store/        # 状态管理
│   ├── utils/        # 工具函数
│   ├── views/        # 视图组件
│   └── types/        # TypeScript类型定义
├── package.json      # 依赖管理
├── vite.config.js    # Vite配置
└── tailwind.config.js # Tailwind配置
```

#### 开发环境配置
- **Node.js环境**: 版本16.0+，确保支持ES6+特性
- **包管理器**: 使用npm或yarn管理依赖
- **IDE配置**: VSCode + Vue.js开发插件
- **浏览器支持**: Chrome DevTools调试工具

### 4.4.2 开发工具链配置

#### 构建工具配置 (Vite)
- **快速启动**: 支持热模块替换(HMR)
- **插件系统**: Vue插件、TypeScript插件支持
- **构建优化**: 代码分割、Tree Shaking
- **开发服务器**: 本地开发环境配置

#### 代码规范配置
- **ESLint**: JavaScript/TypeScript代码检查
- **Prettier**: 代码格式化工具
- **EditorConfig**: 编辑器配置统一
- **Husky**: Git钩子管理

#### 类型检查配置 (TypeScript)
- **严格模式**: 启用严格类型检查
- **类型定义**: 完整的类型支持
- **编译配置**: TSConfig优化配置

#### 样式处理配置 (Tailwind CSS)
- **原子化CSS**: 实用优先的CSS框架
- **主题定制**: 品牌色彩和设计系统
- **响应式设计**: 移动端适配支持

### 4.4.3 核心架构设计

#### 路由架构设计
- **路由分层**: 一级路由、二级路由结构
- **路由守卫**: 权限验证、页面访问控制
- **路由懒加载**: 按需加载路由组件
- **嵌套路由**: 复杂页面结构支持

#### 状态管理架构 (Pinia)
- **Store设计**: 模块化的状态管理
- **数据持久化**: 本地存储数据持久化
- **状态订阅**: 响应式状态变更监听
- **状态调试**: 开发工具调试支持

#### 组件架构设计
- **通用组件**: 基于Tailwind CSS + Lucide图标构建按钮、表单、表格等基础组件
- **表单组件**: 结合VeeValidate实现完整的表单验证功能
- **日期组件**: 使用date-fns处理日期格式化和计算
- **工具组件**: 利用Lodash和VueUse提供常用工具函数
- **业务组件**: 基于Tailwind CSS + 辅助库构建活动卡片、用户信息等业务组件
- **布局组件**: 基于Tailwind CSS + 辅助库构建头部、侧边栏、内容区域等布局组件
- **组件库**: 建立基于Tailwind CSS + 辅助库的自定义组件库，确保设计一致性

#### API集成架构
- **请求封装**: Axios实例封装和配置
- **错误处理**: 统一错误处理机制
- **请求拦截**: Token自动添加、请求日志
- **响应拦截**: 数据格式化、错误提示

### 4.4.4 开发工作流程

#### 组件开发流程
1. **需求分析**: 分析组件功能和交互需求
2. **原型设计**: 基于Tailwind CSS设计组件UI和交互逻辑
3. **组件实现**: 使用Tailwind CSS类名编写组件模板和样式
4. **样式定制**: 根据设计系统定制Tailwind CSS主题和组件样式
5. **单元测试**: 编写组件单元测试用例
6. **文档编写**: 组件使用文档和Tailwind CSS类名说明

#### 页面开发流程
1. **页面规划**: 分析页面功能和数据结构
2. **路由配置**: 配置页面路由和权限
3. **组件组合**: 使用现有组件构建页面
4. **状态管理**: 页面数据状态管理
5. **交互逻辑**: 页面交互和业务逻辑实现

#### 联调测试流程
1. **接口联调**: 与后端API接口联调
2. **功能测试**: 完整功能流程测试
3. **兼容性测试**: 不同浏览器和设备测试
4. **性能测试**: 页面加载和响应性能测试

#### 构建部署流程
1. **代码构建**: 生产环境代码打包构建
2. **静态资源**: 资源优化和CDN部署
3. **版本管理**: 版本控制和发布管理
4. **监控告警**: 线上监控和错误追踪

### 4.4.5 质量保证体系

#### 代码质量管理
- **代码审查**: 团队代码审查机制
- **规范检查**: 自动化代码规范检查
- **类型安全**: TypeScript类型安全检查
- **依赖管理**: 依赖版本和安全检查

#### 测试策略设计
- **单元测试**: 组件和工具函数单元测试
- **集成测试**: 页面功能集成测试
- **E2E测试**: 端到端用户流程测试
- **性能测试**: 页面性能基准测试

#### 性能优化策略
- **打包优化**: 代码分割和懒加载优化
- **缓存策略**: 静态资源缓存配置
- **图片优化**: 图片压缩和响应式处理
- **渲染优化**: 虚拟滚动、列表优化

#### 安全考虑
- **XSS防护**: 输入验证和输出转义
- **CSRF防护**: 请求令牌验证
- **数据安全**: 敏感数据加密处理
- **HTTPS强制**: 生产环境HTTPS启用

## 6. 部署架构

### 6.1 部署环境规划

#### 开发环境
- 本地开发服务器
- 测试数据库
- 开发工具链

#### 测试环境
- 独立的测试服务器
- 测试数据库
- 自动化测试集成

#### 生产环境
- 负载均衡配置
- 数据库主从复制
- CDN静态资源分发
- 监控和日志系统

### 5.2 服务配置

#### 应用服务器配置
```yaml
server:
  port: 8080
  mode: release
  read_timeout: 30s
  write_timeout: 30s

database:
  host: localhost
  port: 3306
  username: app_user
  password: secure_password
  name: volunteer_db
  max_conns: 100
  idle_conns: 10
```

#### 监控配置
- 应用性能监控 (APM)
- 错误日志收集
- 用户行为分析
- 系统资源监控

### 5.3 安全配置

#### 数据安全
- HTTPS强制启用
- 数据加密传输
- 敏感信息脱敏
- 定期安全扫描

#### 访问安全
- API请求频率限制
- 恶意行为检测
- 登录失败锁定
- 会话超时管理

## 7. 技术实现细节

### 7.1 Excel导入导出功能实现

#### 使用第三方包 github.com/xuri/excelize
```go
import "github.com/xuri/excelize/v2"

// 数据导入示例
func ImportUsersFromExcel(filePath string) ([]User, error) {
    f, err := excelize.OpenFile(filePath)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    // 读取Excel数据并转换为模型
    rows, err := f.GetRows("Sheet1")
    if err != nil {
        return nil, err
    }

    var users []User
    for i, row := range rows {
        if i == 0 { // 跳过表头
            continue
        }
        user := User{
            Username: row[0],
            Email:    row[1],
            Phone:    row[2],
            // ...其他字段
        }
        users = append(users, user)
    }

    return users, nil
}

// 数据导出示例
func ExportUsersToExcel(users []User) (*excelize.File, error) {
    f := excelize.NewFile()

    // 设置表头
    headers := []string{"用户名", "邮箱", "电话", "服务时长", "积分"}
    for i, header := range headers {
        cell, _ := excelize.CoordinatesToCellName(i+1, 1)
        f.SetCellValue("Sheet1", cell, header)
    }

    // 填充数据
    for i, user := range users {
        f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+2), user.Username)
        f.SetCellValue("Sheet1", fmt.Sprintf("B%d", i+2), user.Email)
        f.SetCellValue("Sheet1", fmt.Sprintf("C%d", i+2), user.Phone)
        f.SetCellValue("Sheet1", fmt.Sprintf("D%d", i+2), user.TotalHours)
        f.SetCellValue("Sheet1", fmt.Sprintf("E%d", i+2), user.Points)
    }

    return f, nil
}
```

#### 支持的文件格式
- **导入**: XLSX, CSV
- **导出**: XLSX, CSV, PDF

### 7.2 Gens工具生成数据库模型

#### gen.yaml配置文件
```yaml
db:
  # 数据库连接配置
  dsn: "root:password@tcp(localhost:3306)/volunteer_db?charset=utf8mb4&parseTime=True&loc=Local"

tables:
  # 需要生成模型的表
  - name: users
    comment: "用户表"
  - name: activities
    comment: "活动表"
  - name: points_records
    comment: "积分记录表"

gen:
  # 生成配置
  out_path: "./internal/model"
  with_unit_test: false
  model_pkg_name: "model"
  field_nullable: true
  field_signable: false

  # 字段类型映射
  field_type_mapping:
    datetime: "time.Time"
    json: "datatypes.JSON"

model:
  # 模型配置
  field_json_tag: true
  field_with_db_comment: true
```

#### 生成命令
```bash
# 安装gens工具
go install gorm.io/gen/tools/gentool@latest

# 使用gen.yaml配置文件生成模型文件
gentool -c gen.yaml
```

#### 生成的模型文件结构
```go
// 用户模型示例
type User struct {
    ID             int64          `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
    Username       string         `gorm:"column:username;not null" json:"username"`
    Email          string         `gorm:"column:email;not null" json:"email"`
    Phone          string         `gorm:"column:phone" json:"phone"`
    RealName       string         `gorm:"column:real_name;not null" json:"real_name"`
    PasswordHash   string         `gorm:"column:password_hash;not null" json:"password_hash"`
    Role           string         `gorm:"column:role;not null" json:"role"`
    Points         int            `gorm:"column:points;default:0" json:"points"`
    CreatedAt      time.Time      `gorm:"column:created_at" json:"created_at"`
    UpdatedAt      time.Time      `gorm:"column:updated_at" json:"updated_at"`
}

// 表名
tableName() string {
    return "users"
}
```

#### Makefile集成
```makefile
# 生成数据库模型（使用gen.yaml配置文件）
model:
    gentool -c gen.yaml