# Image Upload Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add image upload endpoint (`POST /api/upload`) for volunteer avatars, organization logos, and activity cover images, served via Nginx static files.

**Architecture:** Local storage with abstracted `Storage` interface. Upload handler receives multipart file, validates type/size, saves to `./uploads/{type}/uuid.{ext}`, returns URL. Nginx serves `/uploads/` directly.

**Tech Stack:** Go 1.24, Hertz, GORM

---

### Task 1: Storage Interface + Local Implementation

**Files:**
- Create: `pkg/storage/storage.go`
- Create: `pkg/storage/local.go`

- [ ] **Step 1: Create `pkg/storage/storage.go`**

```go
package storage

import (
	"context"
	"io"
)

type Storage interface {
	Save(ctx context.Context, filePath string, reader io.Reader) error
	GetURL(filePath string) string
	Delete(ctx context.Context, filePath string) error
}
```

- [ ] **Step 2: Create `pkg/storage/local.go`**

```go
package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	baseDir string
	baseURL string
}

func NewLocalStorage(baseDir, baseURL string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir, baseURL: baseURL}
}

func (s *LocalStorage) Save(_ context.Context, filePath string, reader io.Reader) error {
	fullPath := filepath.Join(s.baseDir, filePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return os.WriteFile(fullPath, data, 0644)
}

func (s *LocalStorage) GetURL(filePath string) string {
	return s.baseURL + "/" + filePath
}

func (s *LocalStorage) Delete(_ context.Context, filePath string) error {
	return os.Remove(filepath.Join(s.baseDir, filePath))
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./pkg/storage/...`
Expected: no errors

---

### Task 2: Upload Service

**Files:**
- Create: `internal/service/upload.go`

- [ ] **Step 1: Create `internal/service/upload.go`**

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"volunteer-system/pkg/storage"
)

var (
	ErrInvalidUploadType = errors.New("无效的上传类型")
	ErrFileTooLarge      = errors.New("文件大小超出限制")
	ErrInvalidExtension  = errors.New("不支持的文件格式")
)

type UploadType string

const (
	UploadTypeAvatar    UploadType = "avatar"
	UploadTypeActivity  UploadType = "activity"
	UploadTypeOrg       UploadType = "org"
)

var allowedUploadTypes = map[UploadType]string{
	UploadTypeAvatar:   "avatar",
	UploadTypeActivity: "activity",
	UploadTypeOrg:      "org",
}

type UploadService struct {
	storage            storage.Storage
	maxFileSizeMB      int
	allowedExtensions  []string
}

func NewUploadService(st storage.Storage, maxFileSizeMB int, allowedExts []string) *UploadService {
	return &UploadService{
		storage:           st,
		maxFileSizeMB:     maxFileSizeMB,
		allowedExtensions: allowedExts,
	}
}

func (s *UploadService) Upload(ctx context.Context, file io.Reader, filename string, fileSize int64, uploadType string) (string, error) {
	ut := UploadType(uploadType)
	if _, ok := allowedUploadTypes[ut]; !ok {
		return "", ErrInvalidUploadType
	}

	if s.maxFileSizeMB > 0 && fileSize > int64(s.maxFileSizeMB)*1024*1024 {
		return "", ErrFileTooLarge
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !s.isExtensionAllowed(ext) {
		return "", ErrInvalidExtension
	}

	savedPath := fmt.Sprintf("%s/%s%s", allowedUploadTypes[ut], newUUID(), ext)
	if err := s.storage.Save(ctx, savedPath, file); err != nil {
		return "", err
	}
	return s.storage.GetURL(savedPath), nil
}

func (s *UploadService) isExtensionAllowed(ext string) bool {
	if len(s.allowedExtensions) == 0 {
		return true
	}
	for _, allowed := range s.allowedExtensions {
		if strings.EqualFold(allowed, ext) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Create UUID utility if it doesn't exist** — check `pkg/util/` for existing UUID generation

Run: `grep -r "uuid\|UUID\|NewUUID" pkg/util/`
If none exists, add to `pkg/util/uuid.go`:

```go
package util

import "github.com/google/uuid"

func NewUUID() string {
	return uuid.New().String()
}
```

Then add `github.com/google/uuid` to go.mod:
Run: `go get github.com/google/uuid`

- [ ] **Step 3: Verify build**

Run: `go build ./internal/service/...`
Expected: no errors

---

### Task 3: Upload Handler

**Files:**
- Create: `internal/handler/upload.go`

- [ ] **Step 1: Create `internal/handler/upload.go`**

```go
package handler

import (
	"context"

	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func Upload(ctx context.Context, c *app.RequestContext) {
	uploadType := c.PostForm("type")
	if uploadType == "" {
		response.Fail(c, service.ErrInvalidUploadType)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, err)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.Fail(c, err)
		return
	}
	defer file.Close()

	svc := service.NewUploadService(
		// Storage will be injected — see Task 4
	)
	url, err := svc.Upload(ctx, file, fileHeader.Filename, fileHeader.Size, uploadType)
	if err != nil {
		response.FailWithCode(c, consts.StatusBadRequest, err)
		return
	}
	response.Success(c, map[string]string{"url": url})
}
```

Note: The `NewUploadService` call will be updated in Task 4 when we wire up the Storage dependency.

---

### Task 4: Wire Up Dependencies (server.go + storage init)

**Files:**
- Create: `pkg/storage/init.go`
- Modify: `cmd/cli/server.go`

- [ ] **Step 1: Create `pkg/storage/init.go`**

```go
package storage

import "sync"

var (
	globalStorage Storage
	once          sync.Once
)

func Init(baseDir, baseURL string) {
	once.Do(func() {
		globalStorage = NewLocalStorage(baseDir, baseURL)
	})
}

func GetStorage() Storage {
	return globalStorage
}
```

- [ ] **Step 2: Modify `cmd/cli/server.go`** — add storage init after DB init

Insert after the `initDatabases` call block and before `service.InitNotificationDispatcher`:

```go
import "volunteer-system/pkg/storage"
```

Insert:
```go
// 初始化文件存储
if cfg.Upload != nil {
    storage.Init(cfg.Upload.Dir, "/uploads")
    appLog.Info("文件存储初始化成功: dir=%s", cfg.Upload.Dir)
}
```

- [ ] **Step 3: Update upload handler** to use global storage

Modify `internal/handler/upload.go` to use `storage.GetStorage()`:

```go
import "volunteer-system/pkg/storage"

// inside Upload function:
svc := service.NewUploadService(
    storage.GetStorage(),
    cfg.Upload.MaxFileSizeMB,
    cfg.Upload.AllowedExtensions,
)
```

But wait — the handler doesn't have access to config. Let me instead have the service check the config, or better yet, create the service with the config values in server.go and store it globally.

Actually, the simplest approach: make the handler get config from a global. Or even simpler: pass config through the handler closure.

Let me revise: The best pattern for this codebase is to store config-global values in the `storage` package itself:

- [ ] **Step 3 (revised): Modify `pkg/storage/init.go`** to also store config

```go
package storage

import "sync"

var (
	globalStorage     Storage
	globalMaxFileSize int
	globalExtensions  []string
	once              sync.Once
)

type Config struct {
	Dir               string
	BaseURL           string
	MaxFileSizeMB     int
	AllowedExtensions []string
}

func Init(cfg Config) {
	once.Do(func() {
		globalStorage = NewLocalStorage(cfg.Dir, cfg.BaseURL)
		globalMaxFileSize = cfg.MaxFileSizeMB
		globalExtensions = cfg.AllowedExtensions
	})
}

func GetStorage() Storage {
	return globalStorage
}

func GetMaxFileSizeMB() int {
	return globalMaxFileSize
}

func GetAllowedExtensions() []string {
	return globalExtensions
}
```

- [ ] **Step 4: Update `cmd/cli/server.go`** init call

```go
if cfg.Upload != nil {
    storage.Init(storage.Config{
        Dir:               cfg.Upload.Dir,
        BaseURL:           "/uploads",
        MaxFileSizeMB:     cfg.Upload.MaxFileSizeMB,
        AllowedExtensions: cfg.Upload.AllowedExtensions,
    })
    appLog.Info("文件存储初始化成功: dir=%s", cfg.Upload.Dir)
}
```

- [ ] **Step 5: Update upload handler** to use global storage + config

```go
package handler

import (
	"context"

	"volunteer-system/internal/response"
	"volunteer-system/internal/service"
	"volunteer-system/pkg/storage"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func Upload(ctx context.Context, c *app.RequestContext) {
	uploadType := c.PostForm("type")
	if uploadType == "" {
		response.Fail(c, service.ErrInvalidUploadType)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, err)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.Fail(c, err)
		return
	}
	defer file.Close()

	svc := service.NewUploadService(
		storage.GetStorage(),
		storage.GetMaxFileSizeMB(),
		storage.GetAllowedExtensions(),
	)
	url, err := svc.Upload(ctx, file, fileHeader.Filename, fileHeader.Size, uploadType)
	if err != nil {
		response.FailWithCode(c, consts.StatusBadRequest, err)
		return
	}
	response.Success(c, map[string]string{"url": url})
}
```

- [ ] **Step 6: Verify build**

Run: `go build ./...`
Expected: no errors

---

### Task 5: Upload Route + Registration

**Files:**
- Create: `internal/router/upload.go`
- Modify: `internal/router/router.go`

- [ ] **Step 1: Create `internal/router/upload.go`**

```go
package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterUploadRouter(r *route.RouterGroup) {
	r.POST("/upload", handler.Upload)
}
```

- [ ] **Step 2: Register in `internal/router/router.go`**

Add after `RegisterImportRouter(authApi)`:
```go
RegisterUploadRouter(authApi)
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: no errors

---

### Task 6: Nginx Static File Serving

- [ ] **Step 1: Already done** — `nginx.conf` was updated during design phase

- [ ] **Step 2: Verify** the config

File: `nginx.conf`
```nginx
location /uploads/ {
    alias /app/uploads/;
    expires 30d;
    add_header Cache-Control "public, immutable";
}
```

---

### Task 7: Integration Test

**Files:**
- Create: `internal/handler/upload_test.go`

- [ ] **Step 1: Create test file**

```go
package handler

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"testing"

	"volunteer-system/pkg/storage"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
)

func init() {
	storage.Init(storage.Config{
		Dir:               t.TempDir(),  // note: can't use t.TempDir() in init
		BaseURL:           "/uploads",
		MaxFileSizeMB:     10,
		AllowedExtensions: []string{".jpg", ".jpeg", ".png"},
	})
}

func TestUploadAvatar(t *testing.T) {
	// Use temp dir for test
	tmpDir := t.TempDir()
	storage.Init(storage.Config{
		Dir:               tmpDir,
		BaseURL:           "/uploads",
		MaxFileSizeMB:     10,
		AllowedExtensions: []string{".jpg", ".jpeg", ".png"},
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.jpg")
	part.Write([]byte("fake-image-data"))
	writer.WriteField("type", "avatar")
	writer.Close()

	h := server.New()
	router.RegisterUploadRouter(h.Group("/api"))

	w := ut.PerformRequest(h, http.MethodPost, "/api/upload", &ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: writer.FormDataContentType()},
	)

	resp := w.Result()
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode())
	}
	// Verify response contains URL
	bodyBytes := resp.Body()
	if !bytes.Contains(bodyBytes, []byte("/uploads/avatar/")) {
		t.Fatalf("response missing URL: %s", bodyBytes)
	}
}

func TestUploadInvalidType(t *testing.T) {
	tmpDir := t.TempDir()
	storage.Init(storage.Config{
		Dir:               tmpDir,
		BaseURL:           "/uploads",
		MaxFileSizeMB:     10,
		AllowedExtensions: []string{".jpg"},
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.jpg")
	part.Write([]byte("data"))
	writer.WriteField("type", "invalid")
	writer.Close()

	h := server.New()
	router.RegisterUploadRouter(h.Group("/api"))

	w := ut.PerformRequest(h, http.MethodPost, "/api/upload", &ut.Body{Body: body, Len: body.Len()},
		ut.Header{Key: "Content-Type", Value: writer.FormDataContentType()},
	)

	if w.Result().StatusCode() != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid type")
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/handler/ -run TestUpload -v`
Expected: PASS

---

### Summary: All Changes

| Action | File | What |
|--------|------|------|
| Create | `pkg/storage/storage.go` | `Storage` interface |
| Create | `pkg/storage/local.go` | `LocalStorage` implementation |
| Create | `pkg/storage/init.go` | Global init, `GetStorage()`, config accessors |
| Create | `internal/service/upload.go` | Upload business logic with validation |
| Create | `internal/handler/upload.go` | HTTP handler |
| Create | `internal/router/upload.go` | Route registration |
| Modify | `cmd/cli/server.go` | Add storage init |
| Modify | `internal/router/router.go` | Register upload route |
| Modify | `nginx.conf` | Add `/uploads/` static serving (already done) |
| Create | `internal/handler/upload_test.go` | Integration tests |

**No existing APIs need modification.** The existing `avatar_url`, `logo_url`, `cover_url` fields remain string-based. Frontend uploads → gets URL → passes it to existing update APIs (`UpdateVolunteer`, `UpdateOrganization`, `UpdateActivity`).
