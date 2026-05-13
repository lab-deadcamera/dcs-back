# dcs-back-v0

Module: `dcs-back-v0`, Go 1.21, Gin v1.10.

## Entrypoint

`main.go` — wires Handler → Service → Store (3-layer, same `internal/image` package).

## Commands

```bash
go run main.go
go build ./...
```

No tests, linters, or CI config present.

## Architecture

All image logic lives in `internal/image/`. No database — purely filesystem-backed.

| Layer | File | Role |
|-------|------|------|
| Store | `store.go` | FS read/write/delete, thumbnail generation via `disintegration/imaging` |
| Service | `service.go` | Validation (ext, size), UUID naming, orchestrates store |
| Handler | `handler.go` | Gin handlers for all endpoints |

## API (`/api/v1/images`)

- **POST `/upload`**: multipart field `"image"` (form-data) **or** JSON `{"filename":"...","data":"base64..."}`. Auto-detecta por `Content-Type`. Acepta data URIs (`data:image/png;base64,...`). Returns `{filename, url, thumbnail_url, size}`
- **GET `/list`**: returns array of `{filename, url, thumbnail_url}`
- **GET `/:filename`**: serves original file
- **GET `/thumbnails/:filename`**: serves 300×300 Lanczos thumbnail
- **DELETE `/:filename`**: deletes original + thumbnail

## Config (env vars)

| Var | Default |
|-----|---------|
| `PORT` | `9099` |
| `UPLOAD_DIR` | `./uploads` |
| `BASE_URL` | `http://localhost:{PORT}` |

Hardcoded: max 10MB, allowed exts `.jpg/.jpeg/.png/.gif/.webp`, thumbnails always in `{UPLOAD_DIR}/thumbnails/`.

## Standardized Response Format

All endpoints return responses with the following envelope:

```json
{"data": <object|array|null>, "success": true|false, "message": "..."}
```

- **`data`**: the payload — can be a single object, an array of objects, or `null`.
- **`success`**: `true` for 2xx responses, `false` for error responses.
- **`message`**: human-readable status description (`"success"`, `"created"`, `"character not found"`, etc.).

### Helper functions (`internal/utils/responses.go`)

| Function | HTTP Status | `success` | Typical `message` |
|----------|-------------|-----------|-------------------|
| `Success(c, data)` | 200 | `true` | `"success"` |
| `Created(c, data)` | 201 | `true` | `"created"` |
| `Message(c, msg)` | 200 | `true` | custom message |
| `BadRequest(c, msg)` | 400 | `false` | validation error |
| `Unauthorized(c, msg)` | 401 | `false` | auth error |
| `NotFound(c, msg)` | 404 | `false` | not found |
| `Conflict(c, msg)` | 409 | `false` | duplicate |
| `Gone(c, msg)` | 410 | `false` | resource deleted |
| `InternalError(c, msg)` | 500 | `false` | server error |

This applies to **all** endpoints (Auth, Images, Studio, Files, Characters).

## Notable

- Filenames are random UUIDs — original name is discarded.
- Thumbnails use `imaging.Fit` with Lanczos resampling.
- Delete ignores thumbnail removal failure; only original removal error propagates.
- No auth, no rate limiting, no EXIF handling.
