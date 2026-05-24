# dcs-back

Backend de Dead Camera Server — plataforma de generación de medios (imagen, video, audio, texto) vía proveedores IA externos (Seedance, Seedream, Gemini, etc.).

Module: `dcs-back-v0`, Go 1.21, Gin v1.10, PostgreSQL.

## Entrypoint

`main.go` — ensambla Handler → Service → Store para cada módulo y los registra vía `module.Registry`.

## Comandos

```bash
go run main.go              # iniciar servidor
go build ./...              # compilar
go fmt ./...                # formatear código
go test ./...               # ejecutar tests
```

## Arquitectura

Capa de datos: PostgreSQL con migraciones en `migrations/`. Capa de presentación: JSON estandarizado (ver `internal/utils/responses.go`).

| Módulo | Ruta API | DB | Propósito |
|--------|----------|----|-----------|
| Auth | `/api/v1/auth/*`, `/api/v1/admin/*` | `users`, `roles` | Registro, login, JWT, roles (admin=rol 1), CRUD de usuarios |
| Image | `/api/v1/images/*` | Filesystem (`./uploads`) | Subida/servicio de imágenes, thumbnails vía `disintegration/imaging` |
| File | `/api/v1/files/*` | `files` | Upload genérico, soft/hard delete, trash, restore, purge automático, thumbnails |
| Character | `/api/v1/characters/*` | `characters`, `character_files` | CRUD de personajes con archivos asociados y enriquecimiento de modelos sincronizados |
| Provider | `/api/v1/providers/*` | `providers` | CRUD de proveedores IA |
| Model | `/api/v1/models/*` | `models` | CRUD de modelos por proveedor, favoritos, tipos |
| Project | `/api/v1/projects/*` | `projects`, `scenes`, `takes` | Proyectos con escenas y takes (jerarquía de 3 niveles) |
| Studio | `/api/v1/studio/*` | `generation_logs`, `server_communications`, `generated_assets` | Orquestación de generación (legacy, video, image, audio, text), sync de assets |

## Studio — Generación de medios

El módulo `studio` es el núcleo del sistema. Soporta generación asíncrona con polling de estado.

> Documentación detallada de la arquitectura: [`internal/studio/ARCHITECTURE.md`](internal/studio/ARCHITECTURE.md)

| Subtipo | Handler | Generators |
|---------|---------|------------|
| Legacy | `studio/handler.go` | Seedance, Seedream (handlers directos) |
| Video | `studio/video/` | SeedanceGenerator, SeedanceGalleryGenerator |
| Image | `studio/image/` | SeedreamGenerator, GeminiNanoGenerator |
| Audio | `studio/audio/` | (vía service genérico) |
| Text | `studio/text/` | (vía service genérico) |

Cada generador implementa una interfaz común. Los handlers de video/image/audio/text comparten el patrón: `POST /generate`, `GET /status/:taskId`, `DELETE /task/:taskId`, `POST /preview`.

## API endpoints

### Auth (`/api/v1/auth`)
- `POST /register` — registro de usuario
- `POST /login` — login, devuelve JWT
- `GET /profile` — perfil del usuario autenticado

### Admin (`/api/v1/admin`) — requiere rol 1
- `POST /users`, `GET /users`, `GET /roles`

### Images (`/api/v1/images`)
- `POST /upload` — multipart o base64 JSON
- `GET /list` — listar imágenes
- `GET /:filename` — servir original
- `GET /thumbnails/:filename` — thumbnail 300×300
- `DELETE /:filename` — eliminar

### Files (`/api/v1/files`)
- `POST /upload` — subir archivo
- `GET /trash` — papelera
- `GET` — listar archivos
- `GET /:id` — metadata
- `GET /:id/serve` — servir archivo
- `GET /:id/thumbnail` — servir thumbnail
- `DELETE /:id` — soft delete
- `POST /:id/restore` — restaurar
- `POST /:id/recover-temp` — recuperar temp
- `DELETE /:id/hard` — hard delete

### Characters (`/api/v1/characters`)
- `POST`, `GET`, `GET /:id`, `PATCH /:id`, `DELETE /:id`
- `POST /:id/files`, `GET /:id/files`, `DELETE /:id/files/:fileId`

### Providers (`/api/v1/providers`)
- `POST`, `GET`, `GET /:id`, `PATCH /:id`, `DELETE /:id`
- `GET /:id/models`

### Models (`/api/v1/models`)
- `POST`, `GET`, `GET /:id`, `PATCH /:id`, `DELETE /:id`
- `GET /favorite`, `POST /:id/favorite`

### Projects (`/api/v1/projects`)
- `POST`, `GET`, `GET /:id`, `PATCH /:id`, `DELETE /:id`
- `POST /:id/scenes`, `GET /:id/scenes`
- `GET /:id/scenes/:sceneId`, `PATCH /:id/scenes/:sceneId`, `DELETE /:id/scenes/:sceneId`
- `POST /:id/scenes/:sceneId/takes`, `GET /:id/scenes/:sceneId/takes`
- `GET /:id/scenes/:sceneId/takes/:takeId`, `PATCH`, `DELETE`
- `POST /:id/scenes/:sceneId/takes/save-generation`
- `POST /:id/scenes/:sceneId/takes/:takeId/toggle-active`

### Studio (`/api/v1/studio`)
- `POST /generate-legacy`, `GET /status-legacy/:taskId`
- `POST /sync-asset`, `GET /synced-assets`, `GET /files-with-sync`
- `GET /characters/:id/files-with-sync`
- `POST /sync-character-assets`
- `GET /logs/generation`, `GET /logs/generation/:id`
- `GET /logs/server-communications`, `GET /logs/server-communications/:id`
- `POST /video/generate`, `GET /video/status/:taskId`, `DELETE /video/task/:taskId`, `POST /video/preview`
- `POST /image/generate`, `GET /image/status/:taskId`, `DELETE /image/task/:taskId`, `POST /image/preview`
- `POST /audio/generate`, `GET /audio/status/:taskId`, `DELETE /audio/task/:taskId`, `POST /audio/preview`
- `POST /text/generate`, `GET /text/status/:taskId`, `DELETE /text/task/:taskId`, `POST /text/preview`

## Middleware

| Middleware | Archivo | Uso |
|------------|---------|-----|
| JWT Auth | `internal/middleware/auth.go` | Extrae y valida JWT, inyecta `user_id` en contexto |
| Role check | `internal/middleware/auth.go` | `RequireRole(1)` para admin |
| Rate limit | `internal/middleware/ratelimit.go` | Token bucket, usado en rutas públicas de imágenes |

## Config (env vars)

| Var | Default | Descripción |
|-----|---------|-------------|
| `PORT` | `9099` | Puerto del servidor |
| `DATABASE_URL` | `postgres://dcs:dcs_pass@localhost:5432/dcs_db?sslmode=disable` | Conexión PostgreSQL |
| `JWT_SECRET` | `super_secret_jwt_key_development_only` | Secreto JWT |
| `UPLOAD_DIR` | `./uploads` | Directorio de uploads |
| `OUTPUTS_DIR` | `./outputs` | Directorio de outputs generados |
| `URL_PUBLIC` | `http://localhost:{PORT}` | URL pública base |
| `CORS_ALLOW_ORIGINS` | `*` | Orígenes CORS permitidos |
| `SUPER_ADMIN_USERNAME` | `superadmin` | Usuario super admin semilla |
| `SUPER_ADMIN_PASSWORD` | `superadmin_pass_123` | Password super admin |
| `ASSET_ACCESS_KEY_ID` | — | BytePlus asset library AK |
| `ASSET_SECRET_ACCESS_KEY` | — | BytePlus asset library SK |
| `ASSET_DEFAULT_GROUP_ID` | — | BytePlus grupo por defecto |

Hardcoded: max upload 10MB, extensiones `.jpg/.jpeg/.png/.gif/.webp`, thumbnails 300×300.

## Standardized Response Format

Toda respuesta usa el envelope:

```json
{"data": <object|array|null>, "success": true|false, "message": "..."}
```

Helpers en `internal/utils/responses.go`: `Success`, `Created`, `Message`, `BadRequest`, `Unauthorized`, `NotFound`, `Conflict`, `Gone`, `InternalError`.

## Migraciones

21 migraciones SQL en `migrations/` (numeradas `00001`..`00021`). Tablas: `users`, `roles`, `files`, `characters`, `character_files`, `providers`, `models`, `generation_logs`, `server_communications`, `generated_assets`, `projects`, `scenes`, `takes`.

## Tests

- `internal/project/service_test.go`
- `internal/studio/service_test.go`
- `internal/studio/image/generator_test.go`

Sin linter ni CI configurados.

## Módulos

Cada grupo de rutas es un `module.Module` que se registra en el `Registry`. Todos los módulos heredan el middleware de autenticación automáticamente. Los módulos pueden exponer rutas públicas creando sub-grupos sin `authMw`.

```
main.go → module.Registry
              │
              ├─ auth.NewModule()        → /auth/*, /admin/*
              ├─ image.NewModule()       → /images/*
              ├─ file.NewModule()        → /files/*
              ├─ character.NewModule()   → /characters/*
              ├─ provider.NewModule()    → /providers/*, /models/*
              ├─ project.NewModule()     → /projects/*
              └─ studio.NewModule()      → /studio/*
```

Para crear un nuevo módulo:
1. Crear paquete en `internal/` con Handler/Service/Store
2. Crear `module.go` con un struct que implemente `module.Module`
3. En `Register()`, definir las rutas usando `authMw` (protegidas) o grupos sin middleware (públicas)
4. Registrar en `main.go` vía `registry.Register(...)`
