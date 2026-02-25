# humming-wondering-leaf

# 文件上传与导出功能设计文档

## 目录

- [1. 概述](about:blank#1-%E6%A6%82%E8%BF%B0)
- [2. 文件上传功能](about:blank#2-%E6%96%87%E4%BB%B6%E4%B8%8A%E4%BC%A0%E5%8A%9F%E8%83%BD)
- [3. 文件导出功能](about:blank#3-%E6%96%87%E4%BB%B6%E5%AF%BC%E5%87%BA%E5%8A%9F%E8%83%BD)
- [4. 技术栈总结](about:blank#4-%E6%8A%80%E6%9C%AF%E6%A0%88%E6%80%BB%E7%BB%93)
- [5. 设计模式与最佳实践](about:blank#5-%E8%AE%BE%E8%AE%A1%E6%A8%A1%E5%BC%8F%E4%B8%8E%E6%9C%80%E4%BD%B3%E5%AE%9E%E8%B7%B5)

---

## 1. 概述

本文档详细分析了文件上传和文件导出两个核心功能的设计思路、实现逻辑和技术方案。

### 1.1 功能定位

| 功能 | 用途 | 主要场景 |
| --- | --- | --- |
| 文件上传 | 批量导入数据 | 数据批量导入、信息批量更新、记录批量创建等 |
| 文件导出 | 批量导出数据 | 数据报表导出、记录列表导出、统计分析导出等 |

### 1.2 架构层次

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Layer                          │
│                   (前端应用 / API 调用)                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Handler Layer                           │
│              (HTTP 请求处理、参数验证、响应封装)              │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                           │
│              (业务逻辑、数据处理、权限控制)                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Repository Layer                          │
│              (数据访问、数据库操作、外部服务调用)             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Storage / Database                         │
│              (对象存储 / 关系型数据库)                       │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. 文件上传功能

### 2.1 设计思路

文件上传功能采用 **异步处理 + 状态追踪** 的设计模式，核心思路如下：

1. **快速响应**：接收文件后立即返回上传 ID，不阻塞用户操作
2. **异步处理**：使用 goroutine 在后台处理文件内容
3. **状态追踪**：通过数据库记录处理进度和结果
4. **错误收集**：失败记录保存为 Excel 供用户查看
5. **编码兼容**：自动检测并转换多种编码格式

### 2.2 处理流程

```
┌──────────────┐
│  客户端上传  │
└──────┬───────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│  Handler: UploadAttachment                                  │
│  1. 接收 multipart 表单 (FormFile "file")                   │
│  2. 读取文件内容                                             │
│  3. 检测文件编码 (chardet)                                  │
│  4. 转换为 UTF-8                                             │
│  5. 解析 CSV/Excel 为结构体                                  │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│  Service: UploadAttachment                                  │
│  1. 创建上传记录 (元数据)                                    │
│  2. 返回 uploadId 给客户端                                   │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│  异步处理 (goroutine)                                        │
│  processAttachments(uploadID, data)                         │
│                                                              │
│  for each item in data:                                     │
│    ├─> processSingleItem(item)                              │
│    │   ├─> 类型A -> ServiceA                                │
│    │   ├─> 类型B -> ServiceB                                │
│    │   ├─> 类型C -> ServiceC                                │
│    │   └─> ... 其他类型                                      │
│    │                                                         │
│    ├─> 成功: 更新成功计数                                    │
│    └─> 失败: 更新失败计数 (保存错误信息)                     │
└─────────────────────────────────────────────────────────────┘
       │
       ▼
┌──────────────┐
│  客户端轮询   │
│  查询处理状态 │
└──────────────┘
```

### 2.3 核心组件

### 2.3.1 API 定义

```protobuf
service FileUploadService {
  // 上传文件
  rpc UploadFile(UploadFileRequest) returns (UploadFileResponse) {
    option (google.api.http) = {
      post: "/api/file/upload"
      body: "*"
    };
  }

  // 文件处理状态
  rpc FileProcessStatus(FileProcessStatusRequest) returns (FileProcessStatusResponse) {
    option (google.api.http) = {
      get: "/api/file/process/status"
    };
  }

  // 文件处理结果
  rpc FileProcessResult(FileProcessResultRequest) returns (FileProcessResultResponse) {
    option (google.api.http) = {
      get: "/api/file/process/result"
    };
  }

  // 获取上传Token
  rpc UploadToken(UploadTokenReq) returns (UploadTokenRes) {
    option (google.api.http) = {
      get: "/api/upload/token"
    };
  }
}
```

### 2.3.2 Handler 实现

```go
func UploadFile(ctx context.Context, c *app.RequestContext) {
    // 1. 绑定并验证请求参数
    var req api.UploadFileRequest
    if err := c.BindAndValidate(&req); err != nil {
        response.FailWithCode(c, http.StatusBadRequest, err)
        return
    }

    // 2. 读取multipart文件
    file, err := c.FormFile("file")
    if err != nil {
        response.Fail(c, err)
        return
    }

    // 3. 读取文件内容
    src, err := file.Open()
    fileBytes, err := io.ReadAll(src)

    // 4. 使用chardet检测文件编码
    detector := chardet.NewTextDetector()
    detResult, err := detector.DetectBest(fileBytes)

    // 5. 转换为UTF-8编码
    utf8Data, err := utils.ConvertToUTF8(fileBytes, detResult.Charset)

    // 6. 根据uploadType解析文件
    var data []any
    switch req.UploadType {
    case model.UploadType_A:
        var items []*model.UploadTypeA
        utils.Unmarshal(bytes.NewReader(utf8Data), file.Filename, &items)
        data = make([]any, len(items))
        for i, item := range items {
            data[i] = item
        }
    // ... 其他类型
    }

    // 7. 调用service层处理
    result, err := service.NewFileUploadService(ctx, c.Copy()).UploadFile(
        req.UploadType, file.Filename, data)
}
```

### 2.3.3 Service 层逻辑

```go
func (s *FileUploadService) UploadFile(uploadType int32, fileName string, data []any) (*api.UploadFileResponse, error) {
    // 1. 创建上传记录（元数据）
    uploadRecord := &model.FileUpload{
        CreateBy:   operator.Account,
        UpdateBy:   operator.Account,
        TenantID:   operator.TenantId,
        FileName:   fileName,
        UploadType: uploadType,
        Total:      int32(len(data)),
    }
    err = s.repo.CreateFileUpload(s.repo.DB, uploadRecord)

    // 2. 使用goroutine异步处理（非阻塞）
    go func() {
        defer func() {
            if err := recover(); err != nil {
                stackTrace := debug.Stack()
                logger.Errorf(s.ctx, "panic recovered: %v\n%s", err, stackTrace)
            }
        }()
        s.processFiles(uploadRecord.ID, data)
    }()

    return &api.UploadFileResponse{Id: uploadRecord.ID}, nil
}
```

### 2.3.4 数据库模型

```go
type FileUpload struct {
    ID         int64     `json:"id"`
    CreateBy   string    `json:"create_by"`
    CreateTime time.Time `json:"create_time"`
    UpdateBy   string    `json:"update_by"`
    UpdateTime time.Time `json:"update_time"`
    TenantID   int64     `json:"tenant_id"`
    FileName   string    `json:"file_name"`
    UploadType int32     `json:"upload_type"`  // 上传类型
    Total      int32     `json:"total"`         // 数据总量
    Processed  int32     `json:"processed"`     // 已处理数量
    Succeed    int32     `json:"succeed"`       // 成功数量
    Failed     int32     `json:"failed"`        // 失败数量
    Result     string    `json:"result"`        // 结果（报错信息JSON）
    // 可根据业务需求添加其他字段
}
```

### 2.4 文件存储方案

### 2.4.1 对象存储（如七牛云、阿里云OSS等）

```go
// 生成上传令牌（10分钟有效期）
func GetUploadToken(conf *config.FileStoreConfig) (string, string) {
    putPolicy := PutPolicy{
        Scope: conf.Bucket,
    }
    mac := auth.New(conf.AccessKey, conf.SecretKey)
    upToken := putPolicy.UploadToken(mac)
    return upToken, conf.Api
}

// 生成下载链接（带签名，10分钟有效期）
func GetDownloadUrl(conf *config.FileStoreConfig, keyString string) string {
    currentTime := time.Now()
    result := currentTime.Add(10 * time.Minute)

    downloadPath := keyString + "?e=" + cast.ToString(result.Unix())

    mac := auth.New(conf.AccessKey, conf.SecretKey)
    downloadToken := mac.Sign([]byte(downloadPath))

    realDownloadUrl := downloadPath + "&token=" + downloadToken
    return conf.Api + realDownloadUrl
}
```

### 2.4.2 配置

```yaml
file_store:
bucket: your-bucket-name
access_key: your_access_key
secret_key: your_secret_key
api: https://your-storage-api.com/files/
```

### 2.5 文件解析工具

### 2.5.1 统一解析接口

```go
func Unmarshal(in io.Reader, fileName string, obj interface{}) error {
    // 处理 Excel 文件
    if strings.HasSuffix(fileName, ".xlsx") || strings.HasSuffix(fileName, ".xls") {
        return UnmarshalExcel(in, obj)
    }

    // 处理 CSV 文件
    if strings.HasSuffix(fileName, ".csv") {
        return gocsv.Unmarshal(in, obj)
    }

    return fmt.Errorf("unsupported file: %s", fileName)
}
```

### 2.5.2 编码转换

```go
func ConvertToUTF8(data []byte, encoding string) ([]byte, error) {
    var decoder *transform.Reader

    switch encoding {
    case "GBK":
        decoder = transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
    case "GB-18030":
        decoder = transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewDecoder())
    case "ISO-8859-1":
        decoder = transform.NewReader(bytes.NewReader(data), charmap.ISO8859_1.NewDecoder())
    case "UTF-16LE":
        decoder = transform.NewReader(bytes.NewReader(data), unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder())
    case "UTF-16BE":
        decoder = transform.NewReader(bytes.NewReader(data), unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder())
    default:
        return data, nil
    }

    return io.ReadAll(decoder)
}
```

### 2.6 支持的上传类型

上传类型应根据具体业务需求定义，通常包括：

| 类型 | 说明 |
| --- | --- |
| UploadType_A | 数据类型A上传 |
| UploadType_B | 数据类型B上传 |
| UploadType_C | 数据类型C上传 |
| … | … |

**实现建议**：
- 使用常量定义上传类型
- 每种类型对应一个处理 Service
- 使用策略模式根据类型分发处理逻辑

### 2.7 安全性考虑

1. **编码安全**：自动检测文件编码，防止乱码
2. **认证和授权**：文件上传接口需要登录认证
3. **文件大小限制**：配置文件中设置最大文件大小限制
4. **存储安全**：上传 Token 有有效期，下载链接带签名和过期时间
5. **数据验证**：根据上传类型进行不同的数据校验
6. **错误处理**：使用 defer + recover 捕获 panic
7. **租户隔离**：所有上传记录带租户 ID，实现多租户数据隔离

---

## 3. 文件导出功能

### 3.1 设计思路

文件导出功能采用 **同步生成 + 直接返回** 的设计模式，核心思路如下：

1. **权限控制**：导出前进行权限验证
2. **敏感数据脱敏**：根据用户权限对敏感数据进行脱敏处理
3. **大数据量处理**：使用分页查询和并发加载优化性能
4. **格式统一**：统一使用 Excel 格式导出
5. **标签驱动**：通过结构体标签定义导出字段和列名

### 3.2 处理流程

```
┌──────────────┐
│  客户端请求   │
│  导出数据     │
└──────┬───────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│  Handler: ExportData                                        │
│  1. 绑定并验证请求参数                                       │
│  2. 权限检查                                                │
│  3. 调用 Service 层获取导出数据                              │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│  Service: ExportData                                        │
│  1. 获取敏感数据权限                                         │
│  2. 并发获取关联数据 (errgroup)                             │
│  3. 分页查询主数据 (大数据量)                                │
│  4. 构建导出数据结构                                         │
│  5. 敏感数据脱敏                                             │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│  Utils: MarshalExcel                                         │
│  1. 通过反射读取结构体字段和 excel 标签                      │
│  2. 第一行写入列头                                           │
│  3. 后续行写入数据                                           │
│  4. 生成 Excel 文件内容                                      │
└──────┬──────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│  Response                                                    │
│  1. 设置 HTTP 响应头                                         │
│     - Content-Disposition: attachment; filename=xxx.xlsx    │
│     - Content-Type: application/vnd.openxmlformats...       │
│  2. 返回文件内容                                             │
└─────────────────────────────────────────────────────────────┘
```

### 3.3 核心组件

### 3.3.1 API 定义

```protobuf
// 导出数据
rpc ExportData(ExportDataRequest) returns (ExportDataResponse) {
  option (google.api.http) = {
    get: "/api/data/export"
  };
}

message ExportDataRequest {
  // 导出类型
  int32 exportType = 1;
  // 可选参数 支持勾选导出
  repeated int64 idList = 2;
  // 其他可选参数
  string filterParam = 3;
}
```

### 3.3.2 Handler 实现

```go
func ExportData(ctx context.Context, c *app.RequestContext) {
    var req api.ExportDataRequest
    if err := c.BindAndValidate(&req); err != nil {
        response.FailWithCode(c, http.StatusBadRequest, err)
        return
    }
    userRoleService := service.NewUserRoleService(ctx, c)

    var csvContent string
    var filename string
    switch req.ExportType {
    case model.ExportType_A:
        // 权限检查
        auth, err := userRoleService.CheckUserMenuAuth(consts.ExportTypeAPermission)
        if !auth {
            response.FailWithCode(c, http.StatusForbidden, fmt.Errorf("no permission"))
            return
        }
        result, err := service.NewDataService(ctx, c.Copy()).ExportTypeA()
        // Create excel content
        csvContent, err = utils.MarshalExcel(result)
        filename = fmt.Sprintf("数据导出-%s.xlsx", time.Now().Format("2006-01-02 150405"))
    case model.ExportType_B:
        // 类型B导出
        result, err := service.NewDataService(ctx, c.Copy()).ExportTypeB(req.IdList)
        csvContent, err = utils.MarshalExcel(result)
        filename = fmt.Sprintf("类型B导出-%s.xlsx", time.Now().Format("2006-01-02"))
    // ... 其他导出类型
    }

    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", url.PathEscape(filename)))
    c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Header("Content-Transfer-Encoding", "binary")
    c.Header("Access-Control-Expose-Headers", "Content-Disposition")

    c.Response.SetBody([]byte(csvContent))
    c.Response.SetStatusCode(http.StatusOK)
}
```

### 3.3.3 Service 层逻辑

```go
func (s *DataService) ExportTypeA() ([]*model.ExportTypeA, error) {
    tenantUser, err := GetTenantUserInfoFromContext(s.c)
    if err != nil {
        return nil, err
    }
    tenantId := tenantUser.TenantId

    // 获取敏感数据权限
    isAdmin, sensDataPermits, err := s.repo.GetPermissionSensData()
    if err != nil {
        return nil, fmt.Errorf("failed to get sensitive data permission, err: %v", err)
    }

    var eg errgroup.Group
    // 并发获取关联数据
    relatedDataMap := make(map[int64]*model.RelatedData)
    // ... 更多并发查询

    if err := eg.Wait(); err != nil {
        return nil, err
    }

    // 分页获取数据（大数据量处理）
    size := 300
    offset := 0
    dataList := make([]*model.Data, 0)
    for {
        query := map[string]any{"tenant_id = ?": tenantId}
        data, _, err := s.repo.GetDataList(s.repo.DB, query, offset, size, "id asc", true)
        if err != nil {
            return nil, err
        }
        if len(data) == 0 {
            break
        }
        dataList = append(dataList, data...)
        offset += size
    }

    // 构建导出数据
    result := make([]*model.ExportTypeA, 0, len(dataList))
    for _, v := range dataList {
        exportData := s.buildExportData(v, relatedDataMap)
        // 敏感数据脱敏
        if !isAdmin {
            exportData = s.maskOffSensitiveData(sensDataPermits, exportData)
        }
        result = append(result, exportData)
    }

    return result, nil
}
```

### 3.3.4 Excel 生成工具

```go
// MarshalExcel 将结构体切片转换为 Excel 格式字符串
func MarshalExcel(in interface{}) (out string, err error) {
    // 获取切片的反射值
    val := reflect.ValueOf(in)
    if val.Kind() != reflect.Slice {
        return "", fmt.Errorf("input must be a slice")
    }

    // 创建 Excel 文件
    f := excelize.NewFile()

    // 获取切片元素类型
    elemType := val.Type().Elem()
    if elemType.Kind() == reflect.Ptr {
        elemType = elemType.Elem()
    }
    if elemType.Kind() != reflect.Struct {
        return "", fmt.Errorf("slice elements must be struct type or pointers to struct")
    }

    // 获取列头（基于 struct tag "excel"）
    headers := []string{}
    for i := 0; i < elemType.NumField(); i++ {
        field := elemType.Field(i)
        tag := field.Tag.Get("excel")
        if tag == "" {
            tag = field.Name
        }
        headers = append(headers, tag)
    }

    // 写入列头到 Excel 的第一行
    sheetName := "Sheet1"
    for i, header := range headers {
        ss := ""
        if i >= 26 {
            ss = string('A'+i/26-1) + string('A'+i%26)
        } else {
            ss = string('A' + i)
        }
        cell := fmt.Sprintf("%s1", ss)
        f.SetCellValue(sheetName, cell, header)
    }

    // 写入数据
    for rowIndex := 0; rowIndex < val.Len(); rowIndex++ {
        elem := val.Index(rowIndex)
        if elem.Kind() == reflect.Ptr {
            elem = elem.Elem()
        }
        for colIndex := 0; colIndex < elem.NumField(); colIndex++ {
            field := elem.Field(colIndex)
            ss := ""
            if colIndex >= 26 {
                ss = string('A'+colIndex/26-1) + string('A'+colIndex%26)
            } else {
                ss = string('A' + colIndex)
            }
            cell := fmt.Sprintf("%s%d", ss, rowIndex+2)
            f.SetCellValue(sheetName, cell, field.Interface())
        }
    }

    // 将 Excel 内容写入字节缓冲区
    var buf bytes.Buffer
    if err := f.Write(&buf); err != nil {
        return "", fmt.Errorf("failed to write Excel content to buffer: %v", err)
    }

    // 转换缓冲区内容为字符串并返回
    return buf.String(), nil
}
```

### 3.4 导出数据模型

导出数据模型应根据具体业务需求定义，使用结构体标签来指定导出字段和列名：

```go
// 示例：类型A导出模型
type ExportTypeA struct {
    Field1 string `csv:"字段1" excel:"字段1"`
    Field2 string `csv:"字段2" excel:"字段2"`
    Field3 string `csv:"字段3" excel:"字段3"`
    // ... 更多字段
}

// 示例：类型B导出模型
type ExportTypeB struct {
    Code   string `csv:"编码" excel:"编码"`
    Title  string `csv:"名称" excel:"名称"`
    Status string `csv:"状态" excel:"状态"`
    // ... 更多字段
}
```

**实现建议**：
- 使用 `csv` 和 `excel` 标签定义列名
- 字段类型根据业务需求选择
- 支持嵌套结构体和关联数据

### 3.5 敏感数据脱敏

```go
func (s *DataService) maskOffSensitiveData(sensDataPermits map[int64]*repository.SensDataPermItem, data *model.ExportData) *model.ExportData {
    // 根据权限对敏感数据进行脱敏
    if sensDataPermits[repository.SENSDATA_FIELD1].CanExport == 0 {
        data.Field1 = consts.NoPermissionExportData
    }
    if sensDataPermits[repository.SENSDATA_FIELD2].CanExport == 0 {
        data.Field2 = consts.NoPermissionExportData
    }
    // ... 更多敏感字段
    return data
}
```

### 3.6 支持的导出类型

导出类型应根据具体业务需求定义，通常包括：

| 类型 | 说明 |
| --- | --- |
| ExportType_A | 数据类型A导出 |
| ExportType_B | 数据类型B导出 |
| ExportType_C | 数据类型C导出 |
| … | … |

---

## 4. 技术栈总结

### 4.1 文件上传技术栈

| 组件 | 技术选型 | 说明 |
| --- | --- | --- |
| HTTP 框架 | CloudWeGo Hertz / Gin / Echo | 高性能 HTTP 框架 |
| 文件解析 | excelize, gocsv | Excel 和 CSV 解析 |
| 编码检测 | chardet | 自动检测文件编码 |
| 编码转换 | golang.org/x/text | 多编码转换支持 |
| 对象存储 | 七牛云 / 阿里云OSS / 腾讯云COS | 文件存储服务 |
| 数据库 | GORM + MySQL / PostgreSQL | 元数据存储 |
| 并发处理 | goroutine + errgroup | 异步处理和并发控制 |

### 4.2 文件导出技术栈

| 组件 | 技术选型 | 说明 |
| --- | --- | --- |
| HTTP 框架 | CloudWeGo Hertz / Gin / Echo | 高性能 HTTP 框架 |
| Excel 生成 | excelize | Excel 文件生成 |
| 反射 | reflect | 动态读取结构体字段 |
| 数据库 | GORM + MySQL / PostgreSQL | 数据查询 |
| 并发处理 | errgroup | 并发加载关联数据 |
| 权限控制 | 自定义权限系统 | 菜单权限和敏感数据权限 |

### 4.3 关键依赖包

```go
// 文件处理
import (
    "github.com/xuri/excelize/v2"      // Excel 处理
    "github.com/gocarina/gocsv"         // CSV 处理
    "github.com/bodgi/encodingdetector" // 编码检测
    "golang.org/x/text"                 // 编码转换
)

// HTTP 框架（根据实际选择）
import (
    "github.com/cloudwego/hertz"        // Hertz 框架
    // 或
    "github.com/gin-gonic/gin"          // Gin 框架
)

// 数据库
import (
    "gorm.io/gorm"                      // ORM 框架
)

// 并发控制
import (
    "golang.org/x/sync/errgroup"        // 错误组
)
```

---

## 5. 设计模式与最佳实践

### 5.1 设计模式

### 5.1.1 异步处理模式（文件上传）

```go
// 快速返回，后台处理
func UploadFile(...) {
    // 1. 创建记录
    uploadRecord := &model.FileUpload{...}
    repo.CreateFileUpload(uploadRecord)

    // 2. 异步处理
    go func() {
        defer recover() // panic 恢复
        processFiles(uploadRecord.ID, data)
    }()

    // 3. 立即返回
    return &api.UploadFileResponse{Id: uploadRecord.ID}
}
```

### 5.1.2 策略模式（根据类型分发）

```go
// 根据上传类型分发到不同的处理逻辑
func (s *FileUploadService) processSingleItem(item any) error {
    switch s.uploadType {
    case model.UploadType_A:
        return s.serviceA.ProcessUpload(item)
    case model.UploadType_B:
        return s.serviceB.ProcessUpload(item)
    case model.UploadType_C:
        return s.serviceC.ProcessUpload(item)
    // ...
    }
}
```

### 5.1.3 标签驱动模式（Excel 导出）

```go
// 通过结构体标签定义导出字段
type ExportData struct {
    Field1 string `csv:"字段1" excel:"字段1"`
    Field2 string `csv:"字段2" excel:"字段2"`
    // ...
}

// 反射读取标签并生成 Excel
func MarshalExcel(in interface{}) (string, error) {
    // 通过反射读取 excel 标签
    tag := field.Tag.Get("excel")
    // ...
}
```

### 5.2 最佳实践

### 5.2.1 错误处理

```go
// 使用 defer + recover 捕获 panic
go func() {
    defer func() {
        if err := recover(); err != nil {
            stackTrace := debug.Stack()
            logger.Errorf(s.ctx, "panic recovered: %v\n%s", err, stackTrace)
        }
    }()
    s.processFiles(uploadRecord.ID, data)
}()
```

### 5.2.2 并发控制

```go
// 使用 errgroup 并发获取关联数据
var eg errgroup.Group
dataMap1 := make(map[int64]*model.Data1)
dataMap2 := make(map[int64]*model.Data2)

eg.Go(func() error {
    data1, err := s.repo.GetData1(...)
    // ...
})
eg.Go(func() error {
    data2, err := s.repo.GetData2(...)
    // ...
})

if err := eg.Wait(); err != nil {
    return nil, err
}
```

### 5.2.3 大数据量处理

```go
// 分页查询大数据量
size := 300
offset := 0
for {
    data, _, err := s.repo.GetDataList(s.repo.DB, query, offset, size, "id asc", true)
    if err != nil {
        return nil, err
    }
    if len(data) == 0 {
        break
    }
    dataList = append(dataList, data...)
    offset += size
}
```

### 5.2.4 租户隔离

```go
// 所有数据查询都带上租户 ID
query := map[string]any{
    "tenant_id = ?": tenantId,
    // ...
}
```

### 5.2.5 权限控制

```go
// 导出前检查权限
auth, err := userRoleService.CheckUserMenuAuth(consts.ExportPermission)
if !auth {
    response.FailWithCode(c, http.StatusForbidden, fmt.Errorf("no permission"))
    return
}
```

---

## 6. 关键文件路径总结

### 6.1 文件上传相关

| 组件 | 文件路径（示例） |
| --- | --- |
| API 定义 | `internal/api/file.proto` |
| Handler | `internal/handler/file.go` |
| Service | `internal/service/file.go` |
| Repository | `internal/repository/file.go` |
| 模型 | `internal/model/file.gen.go` |
| 常量 | `internal/model/consts.go` |
| 路由 | `internal/router/file.go` |
| 配置 | `config/config.yaml` |
| 配置结构 | `config/config.go` |
| 存储Token | `pkg/store/token/token.go` |
| Auth 凭证 | `pkg/store/auth/credentials.go` |
| 文件解析 | `pkg/utils/file.go` |
| Excel 解析 | `pkg/utils/excel.go` |

### 6.2 文件导出相关

| 组件 | 文件路径（示例） |
| --- | --- |
| API 定义 | `internal/api/export.proto` |
| Handler | `internal/handler/export.go` |
| Service | `internal/service/export.go` |
| 模型 | `internal/model/export.go` |
| Excel 生成 | `pkg/utils/excel.go` |

---

## 7. 总结

### 7.1 文件上传功能特点

1. **异步处理**：使用 goroutine 非阻塞处理，支持轮询查询状态
2. **多格式支持**：支持 CSV、Excel（.xlsx, .xls）格式
3. **多编码支持**：自动检测并转换为 UTF-8
4. **云存储**：使用对象存储服务（七牛云、阿里云OSS等）
5. **安全性**：Token 认证、签名验证、租户隔离、权限控制
6. **可追溯**：完整的处理状态和错误记录
7. **多业务类型**：支持多种不同的上传类型

### 7.2 文件导出功能特点

1. **架构清晰**：采用分层架构（Handler -> Service -> Repository）
2. **功能完善**：支持多种数据类型的导出
3. **性能优化**：使用并发和分页处理大数据量
4. **安全可靠**：完善的权限控制和敏感数据脱敏
5. **易于扩展**：基于标签的映射机制，易于添加新的导出字段
6. **格式统一**：统一使用 Excel 格式导出

### 7.3 设计原则

1. **快速响应**：上传采用异步处理，立即返回
2. **数据安全**：完善的权限控制和敏感数据保护
3. **性能优化**：并发处理、分页查询、内存缓存
4. **易于维护**：清晰的分层架构和统一的接口设计
5. **可扩展性**：基于标签的映射和策略模式，易于添加新功能