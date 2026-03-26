# `internal/router/login.go`

## 路由

### `POST /api/login`

- 鉴权：否
- 身份：游客，志愿者/组织管理者都通过这里登录
- 功能：账号登录并换取访问令牌
- 请求体：`{ loginType:string, identifier:string, password:string, identity:string }`
- 返回 `data`：`LoginResponse`

### `POST /api/logout`

- 鉴权：否，接口本身未挂认证中间件
- 身份：已登录用户
- 功能：注销当前令牌
- 请求体：`{ token:string }`
- 返回 `data`：`LogoutResponse`

### `POST /api/refresh`

- 鉴权：否
- 身份：已登录用户
- 功能：刷新访问令牌
- 请求体：`{ refreshToken:string }`
- 返回 `data`：`RefreshTokenResponse`

## 数据结构

### 请求消息

### `LoginRequest`

- `loginType:string`
- `identifier:string`
- `password:string`
- `identity:string`

### `LogoutRequest`

- `token:string`

### `RefreshTokenRequest`

- `refreshToken:string`

### `LoginResponse`

- `success:bool`
- `message:string`
- `accessToken:string`
- `refreshToken:string`
- `expiresAt:int64`
- `userInfo:UserInfo`

### `RefreshTokenResponse`

- `success:bool`
- `message:string`
- `token:string`
- `refreshToken:string`
- `expiresAt:int64`
- `userInfo:UserInfo`

### `LogoutResponse`

- `success:bool`
- `message:string`

### `UserInfo`

- `userId:string`
- `userName:string`
- `email:string`
- `phone:string`
- `displayName:string`
- `avatarUrl:string`
- `identity:string`
- `createdAt:int64`
- `updatedAt:int64`
