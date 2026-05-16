# Qiniu Kodo（七牛云对象存储）集成方案

> 本方案采用七牛标准编程模型：**服务端签凭证，客户端直传七牛**。后端不接收文件，只负责生成上传凭证和存储 URL，前端将文件直传七牛后把返回的 URL 写入数据库。

---

## 一、架构总览

### 1.1 与本地存储对比

| | 本地存储 | 七牛云存储 |
|--|---------|-----------|
| **后端角色** | 接收文件 → 存磁盘 → 返回 URL | 生成上传凭证，不碰文件 |
| **文件流向** | 前端 → 后端 → 磁盘 | 前端 → 七牛（不经过后端） |
| **访问方式** | `h.Static()` 或 Nginx | CDN 域名直出，全球加速 |
| **URL 格式** | `/uploads/avatar/uuid.jpg` | `https://cdn.your-domain.com/avatar/uuid.jpg` |

### 1.2 上传流程

```
┌─────────┐   ① POST /api/upload/token    ┌──────────┐
│         │ ──────────────────────────────→ │          │
│  前端   │   ② 返回 {token, domain, key}   │  后端服务  │
│ (浏览器) │ ←────────────────────────────── │ (Go+Hertz)│
│         │                                  └──────────┘
│         │   ③ 带 token 直传文件            ┌──────────┐
│         │ ──────────────────────────────→ │          │
│         │   ④ 返回 {key, hash}             │  七牛 Kodo│
│         │ ←────────────────────────────── │          │
│         │                                  └──────────┘
│         │   ⑤ PUT /api/me/volunteer/profile  ┌───────┐
│         │     {avatar_url: "https://..."}     │  数据库 │
│         │ ──────────────────────────────→ │       │
└─────────┘                                  └───────┘
```

### 1.3 展示流程

```
┌─────────┐   GET /api/volunteer/profile     ┌──────────┐
│  前端   │ ──────────────────────────────→   │  后端服务  │
│ (浏览器) │ ←── {avatar_url: "https://..."}  │   数据库   │
│         │                                  └──────────┘
│  <img src="https://cdn.xxx.com/avatar/..." /> │
│                                                  │
│              直接请求 CDN（不回源后端）          │
│         │ ──────────────────────────────→   ┌──────────┐
│         │                                   │  七牛 CDN │
│         │ ←──────────── 图片数据 ────────── │ (边缘缓存)│
│                                              └──────────┘
```

**关键原则：**
- AK/SK 绝对不可以在客户端出现，仅在后端使用
- SecretKey 不得在任何场景公网传输
- 客户端只和七牛做上传/下载，不做管理操作
- 数据库 URL 字段存的是完整的 CDN 地址，展示时直接使用

### 1.4 前置条件

- 一个已备案的域名用于绑定 CDN
- 七牛账号的 AccessKey 和 SecretKey（[查看地址](https://portal.qiniu.com/user/key)）

---

## 二、七牛控制台配置（一次性）

> 以下步骤在七牛控制台完成，只需配置一次。

### 2.1 创建 Bucket

**路径：** [对象存储](https://portal.qiniu.com/kodo/bucket) → 新建空间

| 字段 | 填写内容 |
|------|---------|
| 存储区域 | 选距离你服务器最近的区域（华东 / 华北 / 华南） |
| 空间名称 | `volunteer-system` |
| 访问控制 | **公开**（头像和活动图是公开资源） |

### 2.2 绑定 CDN 域名

**路径：** 空间管理 → 点击 `volunteer-system` → 域名管理 → 绑定域名

| 字段 | 填写内容 |
|------|---------|
| 加速域名 | `cdn.你的域名.com`（例 `cdn.example.com`，需已备案） |
| 源站配置 | 默认即可（对象存储源站） |

**绑定后还需在 DNS 管理处添加 CNAME 记录：** 将 `cdn.你的域名.com` CNAME 到七牛分配的加速域名。绑定完成后文件访问 URL 格式为 `https://cdn.你的域名.com/avatar/uuid.jpg`。

### 2.3 配置 CORS（必须，否则前端直传被拦截）

**路径：** 空间管理 → `volunteer-system` → 空间设置 → 跨域资源共享 → 添加规则

| 字段 | 值 |
|------|----|
| 来源 Origin | `https://你的前端域名.com`（本地开发可填 `*`） |
| 允许方法 Methods | `GET`, `PUT`, `POST` |
| 允许 Headers | `Content-Type`, `Authorization` |

### 2.4 配置 Referer 防盗链（可选，建议开启）

**路径：** 空间管理 → `volunteer-system` → 空间设置 → Referer 防盗链

| 字段 | 值 |
|------|----|
| 白名单 | `你的前端域名.com` |
| 允许空 Referer | 勾选（小程序等场景需要） |

### 2.5 获取 AK/SK

**路径：** [密钥管理](https://portal.qiniu.com/user/key)

记下 **AccessKey** 和 **SecretKey**，后续填入项目配置。

---

## 三、后端代码改动

### 3.1 新增文件

| 文件 | 职责 |
|------|------|
| `internal/handler/upload_token.go` | `POST /api/upload/token` HTTP 处理器 |
| `internal/service/upload_token.go` | 生成上传凭证的业务逻辑 |
| `internal/router/upload_token.go` | 路由注册 |

### 3.2 修改文件

| 文件 | 改动 |
|------|------|
| `internal/router/router.go` | 注册上传凭证路由（需要 JWT 认证） |
| `config/config.go` | 新增 `QiniuConfig` 结构体 |
| `config/config.prod.yaml` | 新增七牛配置段 |
| `.env` / `.env.example` | 新增 AK/SK 环境变量 |

### 3.3 配置文件

**`config/config.go` 新增结构体：**

```go
type QiniuConfig struct {
    AccessKey string `mapstructure:"access_key"`
    SecretKey string `mapstructure:"secret_key"`
    Bucket    string `mapstructure:"bucket"`
    Domain    string `mapstructure:"domain"`
}

type Config struct {
    // ... 已有字段
    Qiniu *QiniuConfig `mapstructure:"qiniu"`
}
```

**`config/config.prod.yaml` 新增：**

```yaml
qiniu:
  access_key: "${QINIU_ACCESS_KEY}"
  secret_key: "${QINIU_SECRET_KEY}"
  bucket: "volunteer-system"
  domain: "https://cdn.你的域名.com"
```

**`.env` 新增：**

```
QINIU_ACCESS_KEY=你的AccessKey
QINIU_SECRET_KEY=你的SecretKey
```

### 3.4 后端 API 设计

#### `POST /api/upload/token` — 获取上传凭证

需要 JWT 认证。

**请求：**
```json
{ "type": "avatar" }
```

`type` 取值范围：`avatar`（头像）、`activity`（活动封面）、`org`（组织 Logo）

**响应：**
```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "token": "MY_ACCESS_KEY:xxxx:xxxxx",
    "domain": "https://cdn.你的域名.com",
    "key": "avatar/Ftgm-CkWePC9fzMBTRNmPMhGBcSV.jpg"
  }
}
```

前端拼接完整 URL：`data.domain + "/" + data.key`，然后调现有 update 接口入库。

### 3.5 后端核心代码

```go
package service

import (
	"context"
	"time"

	"github.com/qiniu/go-sdk/v7/storagev2/credentials"
	"github.com/qiniu/go-sdk/v7/storagev2/uptoken"
)

type UploadTokenService struct {
	accessKey string
	secretKey string
	bucket    string
	domain    string
}

type UploadTokenResponse struct {
	Token  string `json:"token"`
	Domain string `json:"domain"`
	Key    string `json:"key"`
}

func (s *UploadTokenService) Generate(ctx context.Context, uploadType string) (*UploadTokenResponse, error) {
	scope, saveKey := generateScopeAndKey(uploadType)

	putPolicy, err := uptoken.NewPutPolicyWithKey(scope, "", time.Now().Add(1*time.Hour))
	if err != nil {
		return nil, err
	}
	putPolicy.SetFsizeLimit(10 * 1024 * 1024)
	putPolicy.SetMimeLimit("image/jpeg;image/png;image/gif")
	putPolicy.SetInsertOnly(1)
	putPolicy.ForceSaveKey = true
	putPolicy.SaveKey = saveKey
	putPolicy.SetReturnBody(`{"key":"$(key)","hash":"$(etag)","size":$(fsize)}`)

	creds := credentials.NewCredentials(s.accessKey, s.secretKey)
	token, err := uptoken.NewSigner(putPolicy, creds).GetUpToken(ctx)
	if err != nil {
		return nil, err
	}
	return &UploadTokenResponse{Token: token, Domain: s.domain, Key: saveKey}, nil
}

func generateScopeAndKey(uploadType string) (scope, saveKey string) {
	switch uploadType {
	case "avatar":
		return "volunteer-system:avatar/", "avatar/$(etag)$(ext)"
	case "activity":
		return "volunteer-system:activity/", "activity/$(etag)$(ext)"
	case "org":
		return "volunteer-system:org/", "org/$(etag)$(ext)"
	}
	return "", ""
}
```

---

## 四、上传策略字段说明

根据[上传策略文档](https://developer.qiniu.com/kodo/1206/put-policy)：

| 字段 | 值 | 作用 |
|------|----|------|
| `scope` | `volunteer-system:avatar/` | 限制上传目录前缀（根据 type） |
| `deadline` | 当前时间 + 1 小时 | 凭证有效期 |
| `fsizeLimit` | `10 * 1024 * 1024`（10MB） | 限制文件大小 |
| `mimeLimit` | `image/jpeg;image/png;image/gif` | 限制只允许图片 |
| `saveKey` | `avatar/$(etag)$(ext)` | 自动命名，文件 hash + 扩展名 |
| `forceSaveKey` | `true` | 强制使用 saveKey 命名 |
| `insertOnly` | `1` | 禁止覆盖已有文件 |
| `fileType` | `0` | 标准存储 |
| `returnBody` | `{"key":"$(key)","hash":"$(etag)","size":$(fsize)}` | 七牛返回给前端的信息 |

**目录结构（自动按 type 存放）：**

```
七牛 bucket: volunteer-system
├── avatar/         # 头像（etag 去重，同名不会重复上传）
├── activity/       # 活动封面
└── org/            # 组织 Logo
```

---

## 五、前端改动

### 5.1 安装 SDK

```bash
npm install qiniu-js
```

### 5.2 上传代码示例

```javascript
import * as qiniu from 'qiniu-js'

async function uploadFile(file, uploadType) {
  // 1. 从后端获取上传凭证
  const res = await fetch('/api/upload/token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: 'Bearer xxx' },
    body: JSON.stringify({ type: uploadType }),
  })
  const { data } = await res.json()

  // 2. 直传七牛
  const observable = qiniu.upload(file, data.key, data.token)
  
  return new Promise((resolve, reject) => {
    observable.subscribe({
      next: (res) => console.log('上传进度', res.total.percent),
      error: (err) => {
        if (err.code === 401) {
          // token 过期，重新获取并重试
        }
        reject(err)
      },
      complete: (res) => {
        // 3. 拼接完整 URL
        const url = data.domain + '/' + res.key
        resolve(url)
      },
    })
  })
}

// 上传头像示例
const avatarUrl = await uploadFile(fileInput.files[0], 'avatar')
await fetch('/api/me/volunteer/profile', {
  method: 'PUT',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ avatar_url: avatarUrl }),
})
```

### 5.3 凭证过期处理

token 有效期 1 小时，前端处理策略：

- 上传前检查 token 生成时间是否超过 55 分钟，超过则重新请求
- 或在上传失败时检查状态码，若是 `401` 则重新获取 token 重试

### 5.4 注意事项

- 七牛 JS SDK 自动处理分片上传，大文件无需额外处理
- 上传进度可做进度条展示
- 生产环境建议封装成通用上传组件，统一处理 token 刷新和错误重试

---

## 六、图片处理（七牛实时样式）

七牛支持 URL 后加参数实时处理图片，无需提前处理。

### 6.1 常用指令

| 指令 | 示例 URL | 效果 |
|------|---------|------|
| `imageView2/1/w/200/h/200` | `...jpg?imageView2/1/w/200/h/200` | 裁剪为 200x200 正方形 |
| `imageView2/2/w/400/h/300` | `...jpg?imageView2/2/w/400/h/300` | 缩放至 400x300 以内 |
| `imageMogr2/auto-orient` | `...jpg?imageMogr2/auto-orient` | 自动修正手机照片方向 |
| `imageMogr2/format/webp` | `...jpg?imageMogr2/format/webp` | 转为 WebP 格式（体积更小） |

### 6.2 头像推荐用法

```javascript
// 列表中展示小头像
`${avatarUrl}?imageView2/1/w/200/h/200`
// 个人主页展示大头像
`${avatarUrl}?imageView2/1/w/400/h/400`
```

数据库 `avatar_url` 存原图 URL，前端展示时拼接处理参数。

### 6.3 样式别名（可选）

在七牛控制台「图片样式」设置别名后 URL 可简写：

| 样式名 | 规则 |
|--------|------|
| `avatar` | `imageView2/1/w/200/h/200` |

原链：`.../avatar.jpg?imageView2/1/w/200/h/200`
简写：`.../avatar.jpg-avatar`

---

## 七、扩展方案

### 7.1 回调（Callback）

备选方案：七牛上传完后直接回调后端，自动写入数据库，省一次前端请求。

| | 回调 | 前端主动调 update |
|--|------|------------------|
| 前端工作量 | 少，上传完就结束 | 多一步请求 |
| 后端工作量 | 多一个回调接口 + VerifyCallback 验证 | 无新增 |
| 可靠性 | 七牛自动重试 3 次 | 前端需处理重试 |

**推荐用前端主动调 update**，更简单透明。

### 7.2 文件删除（可选）

用户换头像时可删除旧文件：

```go
import "github.com/qiniu/go-sdk/v7/storagev2/objects"

objectsManager := objects.NewObjectsManager(...)
err := objectsManager.Bucket("volunteer-system").
    Object("avatar/oldfile.jpg").
    Delete().
    Call(ctx)
```

---

## 八、注意事项

### 8.1 测试域名

七牛新建 Bucket 会分配一个**测试域名**（如 `volunteer-system.s3-cn-east-1.qiniucs.com`），有 **30 天有效期**和**每日流量限制**，仅可用于开发测试。生产环境必须绑定已备案的**自定义 CDN 域名**。

### 8.2 双模式共存

配置 `upload.driver` 开关可同时支持两种模式：

```yaml
upload:
  driver: "local"    # "local" 走本地存储；"qiniu" 走七牛
  max_file_size_mb: 10
```

对前端透明（返回的都是完整 URL）。

### 8.3 迁移路径

1. 在七牛控制台完成**第二章节**的配置
2. 新增七牛配置段和 AK/SK 环境变量
3. 实现 `POST /api/upload/token` 接口
4. 前端安装 `qiniu-js` SDK 并改上传逻辑
5. 本地开发继续用本地存储，生产环境切七牛
