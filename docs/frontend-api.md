# Frontend API Implementation Guide

## Base URL

```
http://localhost:9099/api/v1
```

## Authentication

### Register

```
POST /auth/register
Content-Type: application/json

{
  "username": "user",
  "password": "pass123",
  "name": "John",
  "surname": "Doe"
}
```

Response `201`:
```json
{
  "id": 1,
  "username": "user",
  "name": "John",
  "surname": "Doe",
  "active": true,
  "created_at": "...",
  "updated_at": "..."
}
```

### Login

```
POST /auth/login
Content-Type: application/json

{
  "username": "user",
  "password": "pass123"
}
```

Response `200`:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Using the token

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

> Protected routes require this header. Only image upload/list/delete are protected.  
> Studio routes (keys, generate, etc.) and file routes use their own auth.

---

## Image Management (old system, filesystem only)

All `GET` image routes have rate limiting (10 req/s). Upload/List/Delete require JWT.

### Upload (multipart)

```
POST /images/upload
Authorization: Bearer <token>
Content-Type: multipart/form-data

image: <file>
```

Response `201`:
```json
{
  "filename": "uuid.jpg",
  "url": "http://localhost:9099/api/v1/images/uuid.jpg",
  "thumbnail_url": "http://localhost:9099/api/v1/images/thumbnails/uuid.jpg",
  "size": 12345
}
```

### Upload (base64)

```
POST /images/upload
Authorization: Bearer <token>
Content-Type: application/json

{
  "filename": "photo.png",
  "data": "data:image/png;base64,iVBORw0KGgo..."
}
```

### List images

```
GET /images/list
Authorization: Bearer <token>
```

```json
[
  {
    "filename": "uuid.jpg",
    "url": "http://localhost:9099/api/v1/images/uuid.jpg",
    "thumbnail_url": "http://localhost:9099/api/v1/images/thumbnails/uuid.jpg"
  }
]
```

### Serve image

```
GET /images/:filename       → raw file
GET /images/thumbnails/:filename  → 300×300 thumbnail
```

### Delete

```
DELETE /images/:filename
Authorization: Bearer <token>
```

```json
{ "message": "image deleted" }
```

---

## Studio (BytePlus ModelArk)

No JWT required. Auth via `/api/v1/keys`.

### API Keys management

#### List keys

```
GET /keys
```

```json
{
  "active": "a1b2c3d4e5f6",
  "endpoints": {
    "byteplus_ap": { "label": "BytePlus · Singapore", "url": "..." },
    "volcengine_cn": { "label": "Volcengine · China", "url": "..." }
  },
  "keys": [
    {
      "id": "a1b2c3d4e5f6",
      "name": "my-key",
      "preview": "ABCD••••xyz1",
      "endpoint": "byteplus_ap",
      "hasAkSk": false,
      "akPreview": null,
      "createdAt": "2026-05-12T00:00:00Z"
    }
  ]
}
```

#### Add key

```
POST /keys
Content-Type: application/json

{
  "name": "my-key",
  "value": "your-byteplus-api-key",
  "endpoint": "byteplus_ap",
  "ak": "AKLT... (optional)",
  "sk": "... (optional)"
}
```

#### Activate key

```
POST /keys/:id/activate
```

#### Delete key

```
DELETE /keys/:id
```

#### Update key (rename / change endpoint)

```
PATCH /keys/:id
Content-Type: application/json

{
  "name": "new-name",
  "endpoint": "volcengine_cn"
}
```

---

### Presets

```
GET /presets
```

Returns the full `presets.json` content (lens, camera, motion, color grading, genre, aspect ratio, resolution).

---

### Compile Prompt

Build the cinematographic text block without calling BytePlus.

```
POST /compile-prompt
Content-Type: application/json

{
  "userPrompt": "A woman walks through neon rain",
  "camera": { "id": "c1", "prompt": "Blade Runner 2049 aesthetic" },
  "lens": { "id": "l1", "prompt": "Anamorphic 35mm" },
  "cameraMotion": { "id": "m1", "prompt": "Slow dolly forward" },
  "colorGrading": { "id": "g1", "prompt": "Teal-orange grade" },
  "genre": { "id": "gen1", "prompt": "Cyberpunk noir" },
  "aspectRatio": { "value": "16:9" },
  "duration": 5,
  "soundOn": true,
  "firstFrame": { "dataUrl": "data:image/png;base64,..." },
  "lastFrame": { "dataUrl": "data:image/png;base64,..." },
  "refImages": [{ "dataUrl": "data:image/png;base64,..." }],
  "refVideos": [{ "dataUrl": "data:video/mp4;base64,..." }],
  "refAudios": [{ "dataUrl": "data:audio/mp3;base64,..." }]
}
```

Response `200`:
```json
{
  "prompt": "A woman walks through neon rain. Blade Runner 2049 aesthetic. Anamorphic 35mm. Slow dolly forward. Teal-orange grade. Cyberpunk noir."
}
```

---

### Generate (Seedance video)

Creates a video generation task on BytePlus.

```
POST /generate
Content-Type: application/json

// Same body as compile-prompt
```

Response `200`:
```json
{
  "taskId": "byteplus-task-id",
  "prompt": "...",
  "model": "dreamina-seedance-2-0-fast-260128"
}
```

### Poll status

Frontend must poll every few seconds.

```
GET /status/:taskId
```

While running:
```json
{
  "status": "running",
  "raw": { ... }
}
```

On success (video downloaded locally):
```json
{
  "status": "succeeded",
  "videoUrl": "https://...original-byteplus-url.mp4",
  "localUrl": "/outputs/seedance_123456_abcde123.mp4",
  "raw": { ... }
}
```

On success (no URL found):
```json
{
  "status": "succeeded_no_url",
  "error": "Job succeeded but no video URL was found in the response.",
  "raw": { ... }
}
```

> `localUrl` is accessible at `http://localhost:9099/outputs/seedance_xxx.mp4`

### Cancel task

```
DELETE /task/:taskId
```

---

### Seedream (image generation)

#### List trusted assets

```
GET /seedream/assets
```

```json
{
  "assets": [
    {
      "id": "...",
      "url": "https://...",
      "prompt": "...",
      "model": "seedream-4-0-250828",
      "seed": 42,
      "size": "2K",
      "createdAt": 1712345678000,
      "expiresAt": 1714937678000
    }
  ]
}
```

#### Generate image

```
POST /seedream/generate
Content-Type: application/json

{
  "prompt": "A cinematic portrait",
  "model": "seedream-4-0-250828",
  "size": "2K",
  "seed": 42,
  "referenceImages": ["data:image/png;base64,..."]
}
```

> Trusted assets are valid for 30 days from creation (in-memory, lost on restart).

---

### Assets API (BytePlus management plane)

Requires AK/SK configured on the active key.

#### Create asset group

```
POST /assets/groups
Content-Type: application/json

{
  "name": "My Characters",
  "description": "Character reference library",
  "projectName": "default"
}
```

#### List asset groups

```
GET /assets/groups
```

#### Create asset (add URL to group)

```
POST /assets
Content-Type: application/json

{
  "groupId": "...",
  "url": "https://publicly-accessible-image.jpg",
  "name": "Character ref",
  "assetType": "Image",
  "moderationStrategy": "Skip",
  "projectName": "default"
}
```

#### Get single asset

```
GET /assets/:id?projectName=default
```

#### List assets

```
GET /assets?groupId=...&statuses=Processing,Succeeded&projectName=default
```

#### Delete asset

```
DELETE /assets/:id?projectName=default
```

---

## File Management (DB-backed, all categories)

### Upload

```
POST /files/upload
Content-Type: multipart/form-data

file: <binary>
category: "images" | "videos" | "audio" | "temp"
storage: "persistent" | "temp" (default: persistent)
```

Response `201`:
```json
{
  "id": "uuid",
  "filename": "original-name.png",
  "url": "http://localhost:9099/api/v1/files/uuid",
  "size": 12345,
  "mime_type": "image/png",
  "format": "png",
  "category": "images"
}
```

### List files

```
GET /files?category=images&storage=persistent&trashed=false
```

All params optional. Filters server-side.

```json
[
  {
    "id": "uuid",
    "filename": "...",
    "path": "images/uuid.png",
    "size": 12345,
    "mime_type": "image/png",
    "category": "images",
    "format": "png",
    "storage": "persistent",
    "trashed": false,
    "created_at": "...",
    "updated_at": "...",
    "deleted_at": null
  }
]
```

### Get file metadata

```
GET /files/:id
```

### Serve file (raw binary)

```
GET /files/:id/serve
```

### Soft delete (move to trash)

```
DELETE /files/:id
```

Response:
```json
{ "message": "file moved to trash" }
```

File is moved from `uploads/{category}/` to `uploads/trash/{category}/`.

### Restore from trash

```
POST /files/:id/restore
```

### List trash

```
GET /files/trash
```

### Recover temp file

If a temp file was soft-deleted and is in trash, recover it back to `temp/`.

```
POST /files/:id/recover-temp
```

### Hard delete (permanent)

```
DELETE /files/:id/hard
```

Removes file from disk and DB record.

---

## Characters

### Create

```
POST /characters
Content-Type: application/json

{
  "name": "Neon Samurai",
  "description": "Cyberpunk protagonist",
  "metadata": "{\"age\": 28, \"style\": \"cyberpunk\"}"
}
```

`metadata` is a JSONB field — pass a JSON string.

Response `201`:
```json
{
  "id": "uuid",
  "name": "Neon Samurai",
  "description": "Cyberpunk protagonist",
  "metadata": "{\"age\": 28, \"style\": \"cyberpunk\"}",
  "created_at": "...",
  "updated_at": "...",
  "deleted_at": null
}
```

### List

```
GET /characters
```

### Get with files

```
GET /characters/:id
```

```json
{
  "character": { ... },
  "files": [
    {
      "file_id": "uuid",
      "role": "portrait",
      "created_at": "..."
    }
  ]
}
```

### Update

```
PATCH /characters/:id
Content-Type: application/json

{
  "name": "New Name",
  "description": "Updated description",
  "metadata": "{\"age\": 30}"
}
```

Only send the fields you want to change.

### Soft delete

```
DELETE /characters/:id
```

### Link file to character

```
POST /characters/:id/files
Content-Type: application/json

{
  "file_id": "uuid-of-existing-file",
  "role": "portrait"
}
```

`role`: `reference`, `portrait`, `asset`, or any custom string.

### List character's files

```
GET /characters/:id/files
```

### Unlink file

```
DELETE /characters/:id/files/:fileId
```

> This does NOT delete the file. Only removes the relationship.

---

## Health & Debug

### Health

```
GET /health
```

```json
{
  "ok": true,
  "keysCount": 2,
  "activeKey": true,
  "activeEndpoint": "byteplus_ap",
  "activeEndpointLabel": "BytePlus · Singapore (ap-southeast)",
  "defaultModel": "dreamina-seedance-2-0-fast-260128"
}
```

### Debug

```
GET /debug
```

Returns full debug info including masked keys, endpoints, and config.

---

## Standard Error Format

All errors follow this shape:

```json
{
  "error": "description of what went wrong"
}
```

| HTTP | Common scenarios |
|------|-----------------|
| 400 | Missing field, invalid value, validation error |
| 401 | Invalid credentials |
| 404 | Resource not found |
| 409 | Duplicate (e.g. username exists) |
| 410 | File was soft-deleted |
| 500 | Internal / upstream API error |

## Frontend Implementation Notes

### File preview from `/files/:id/serve`

Use the `id` directly in an `<img>`, `<video>`, or `<audio>` tag:

```html
<img src="http://localhost:9099/api/v1/files/{id}/serve" />
<video src="http://localhost:9099/api/v1/files/{id}/serve" controls />
<audio src="http://localhost:9099/api/v1/files/{id}/serve" controls />
```

### Polling `GET /status/:taskId`

Poll every 2–3 seconds. Stop when `status` is `succeeded` or `failed`.  
Timeout after 10 minutes.

```javascript
async function poll(taskId) {
  const res = await fetch(`/api/v1/status/${taskId}`);
  const data = await res.json();
  if (data.status === 'succeeded') {
    // show video
    return;
  }
  if (data.status === 'failed') {
    // show error
    return;
  }
  setTimeout(() => poll(taskId), 2000);
}
```

### Media size limits (recommended frontend validation)

| Category | Max |
|----------|-----|
| images   | 10 MB |
| videos   | 200 MB |
| audio    | 50 MB |

### Data URI for Studio assets

BytePlus Studio expects base64 data URIs. Convert files client-side:

```javascript
function toDataUrl(file) {
  return new Promise((resolve) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result);
    reader.readAsDataURL(file);
  });
}
```
