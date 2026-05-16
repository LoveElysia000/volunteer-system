# Image Upload Feature Design

## Overview
Add image upload functionality for volunteer avatars, organization logos, and activity cover images. Uses local storage with abstracted `Storage` interface for future migration to object storage.

## Architecture

```
Client Upload → POST /api/upload → Handler → Service → Storage.Save()
                                                          ↓
                                                    Local Disk (./uploads/)
Client Access → GET /uploads/xxx.jpg → Nginx static (no Go overhead)
```

## Storage Interface

```go
type Storage interface {
    Save(ctx context.Context, filePath string, reader io.Reader) error
    GetURL(filePath string) string
    Delete(ctx context.Context, filePath string) error
}
```

- `LocalStorage` implements via `os.WriteFile`
- Future `OSSStorage` implements via S3 SDK
- Business code depends only on interface

## Directory Layout

```
./uploads/
├── avatar/          # Volunteer avatars
├── activity/        # Activity cover images
└── org/             # Organization logos
```

## API Design

### `POST /api/upload`
- **Auth**: Required (JWT)
- **Content-Type**: `multipart/form-data`
- **Parameters**:
  - `file` — the image file (required)
  - `type` — `avatar` | `activity` | `org` (required)
- **Validation**:
  - File size ≤ `max_file_size_mb` from config (10MB)
  - Extension in `allowed_extensions` (.jpg, .jpeg, .png, .gif)
- **Response**:
  ```json
  {
    "code": 200,
    "msg": "success",
    "data": { "url": "/uploads/avatar/uuid.jpg" }
  }
  ```

### How Existing APIs Consume the URL
- Frontend uploads image → gets URL → passes it to existing update APIs
- **`avatar_url`**: Already in `VolunteerUpdateRequest`, `UpdateVolunteer` handler
- **`logo_url`**: Already in `OrganizationUpdateRequest`, `UpdateOrganization` handler
- **`cover_url`**: Already in `UpdateActivityRequest`, `UpdateActivity` handler
- **No proto/API changes needed** — image fields already exist

## Files to Create

| File | Purpose |
|------|---------|
| `pkg/storage/storage.go` | `Storage` interface |
| `pkg/storage/local.go` | Local filesystem implementation |
| `internal/handler/upload.go` | Upload HTTP handler |
| `internal/service/upload.go` | Upload business logic |
| `internal/router/upload.go` | Route registration |

## Files to Modify

| File | Change |
|------|--------|
| `nginx.conf` | Add `location /uploads/` for static serving |
| `internal/router/router.go` | Register upload route group |
| `cmd/cli/server.go` | Initialize Storage and inject into handler |

## Nginx Config

```nginx
location /uploads/ {
    alias /app/uploads/;
    expires 30d;
    add_header Cache-Control "public, immutable";
}
```

## Files NOT to Change

| File | Reason |
|------|--------|
| `config/config.prod.yaml` | Already has `upload` config section |
| `config/config.yaml` | Already has `upload` config section |
| `docker-compose.prod.yml` | Already mounts `./uploads:/app/uploads` |
| `Dockerfile` | Already creates `/app/uploads` |
| Any `.proto` file | URL fields (`avatar_url`, `logo_url`, `cover_url`) already exist |
| Any existing handler/service | Upload is separate; existing APIs consume URL as string |

## Future Extensibility

- To switch to OSS: implement `OSSStorage`, change 1 init line in `server.go`
- To add file types: extend `allowed_extensions` config, add subdirectory
- To add image processing: insert processing step between save and return
