# `internal/router/register.go`

## 路由

### `POST /api/volunteer/register`

- 鉴权：否
- 身份：游客，志愿者注册
- 功能：创建志愿者账号
- 请求体：`{ name:string, phone:string, email:string, password:string, age:int32, gender:string, userName:string }`
- 返回 `data`：`{}`

### `POST /api/organization/register`

- 鉴权：否
- 身份：游客，组织管理者注册
- 功能：创建组织管理者账号
- 请求体：`{ name:string, phone:string, email:string, password:string, organizationName:string, code:string, userName:string }`
- 返回 `data`：`{}`

## 备注

- 这两个接口都走统一 JSON 包装，业务响应体本身为空对象。

## 数据结构

### `VolunteerRegisterRequest`

- `name:string`
- `phone:string`
- `email:string`
- `password:string`
- `age:int32`
- `gender:string`
- `userName:string`

### `OrganizationRegisterRequest`

- `name:string`
- `phone:string`
- `email:string`
- `password:string`
- `organizationName:string`
- `code:string`
- `userName:string`

### `RegisterResponse`

- 空对象 `{}`
