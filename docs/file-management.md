# File Management

## Storage Structure

```
UPLOAD_DIR (./uploads)
├── images/          → imágenes estáticas persistentes
├── videos/          → videos estáticos persistentes
├── audio/           → audios estáticos persistentes
├── temp/            → archivos temporales (purga automática > 1 mes)
└── trash/           → papelera de reciclaje (soft delete)
    ├── images/
    ├── videos/
    ├── audio/
    └── temp/
```

- **Persistent**: archivos que permanecen hasta ser eliminados explícitamente.
- **Temp**: archivos temporales. Se purgan automáticamente vía cron si superan 1 mes de antigüedad. También pueden eliminarse manualmente (soft delete → trash) y recuperarse desde trash.
- **Trash**: papelera de reciclaje para soft deletes. Archivos movidos aquí pueden restaurarse a su ubicación original o purgarse definitivamente.

## Database Schema

### Migration: `00002_create_files_table.sql`

```sql
-- +goose Up
CREATE TABLE files (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename    VARCHAR(255) NOT NULL,              -- nombre original
    path        TEXT NOT NULL,                       -- ruta relativa desde UPLOAD_DIR
    size        BIGINT NOT NULL,                     -- bytes
    mime_type   VARCHAR(127) NOT NULL,               -- image/png, video/mp4, audio/wav
    category    VARCHAR(31) NOT NULL,                -- image | video | audio
    format      VARCHAR(15) NOT NULL,                -- png, mp4, wav, etc.
    storage     VARCHAR(15) NOT NULL DEFAULT 'persistent',  -- persistent | temp
    trashed     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_files_category ON files (category);
CREATE INDEX idx_files_storage ON files (storage);
CREATE INDEX idx_files_deleted_at ON files (deleted_at);

-- +goose Down
DROP TABLE IF EXISTS files;
```

### Campos

| Campo       | Tipo      | Descripción |
|-------------|-----------|-------------|
| `id`        | UUID      | Identificador único |
| `filename`  | VARCHAR   | Nombre original del archivo |
| `path`      | TEXT      | Ruta relativa (ej. `images/uuid.png`) |
| `size`      | BIGINT    | Tamaño en bytes |
| `mime_type` | VARCHAR   | MIME type (ej. `image/png`, `video/mp4`) |
| `category`  | VARCHAR   | `image`, `video`, o `audio` |
| `format`    | VARCHAR   | Extensión sin punto (ej. `png`, `mp4`, `wav`) |
| `storage`   | VARCHAR   | `persistent` o `temp` |
| `trashed`   | BOOLEAN   | Indica si está en la papelera |
| `created_at` | TIMESTAMPTZ | Fecha de creación |
| `updated_at` | TIMESTAMPTZ | Fecha de última modificación |
| `deleted_at` | TIMESTAMPTZ | Fecha de soft delete (NULL si activo) |

## Operations

### Upload

1. Validar extensión y tamaño según configuración.
2. Generar UUID + extensión como nombre.
3. Guardar archivo en `UPLOAD_DIR/{category}/uuid.ext`.
4. Insertar registro en `files`.
5. Devolver `{ id, filename, url, size, mime_type, format, category }`.

### Serve

- `GET /api/v1/files/{category}/{filename}` → archivo estático.
- `GET /api/v1/files/{id}` → redirect a la ruta física (permite cambiar la ruta sin invalidar URLs).

### Delete (soft)

1. Mover el archivo de `UPLOAD_DIR/{category}/` a `UPLOAD_DIR/trash/{category}/`.
2. Actualizar `trashed = TRUE`, `deleted_at = NOW()`.
3. NO borrar el registro.

### Restore

1. Mover el archivo de `UPLOAD_DIR/trash/{category}/` de vuelta a `UPLOAD_DIR/{category}/`.
2. Actualizar `trashed = FALSE`, `deleted_at = NULL`.

### Purge (hard delete)

1. Eliminar físicamente el archivo de `trash/`.
2. Eliminar el registro de `files`.

### Temp purge (cron — midnight)

```
CRON: 0 0 * * *
```

1. Buscar archivos con `storage = 'temp'` y `created_at < NOW() - INTERVAL '1 month'`.
2. Hard delete: eliminar archivo físico + registro de BD.
3. Ejecutar una vez al día a la medianoche (configurable vía env).

### Temp recovery

Los archivos temporales eliminados (soft delete) van a `trash/temp/`. Se pueden recuperar:

1. `GET /api/v1/files/trash` → listar archivos en papelera (incluye temp).
2. `POST /api/v1/files/{id}/restore` → restaurar a su ubicación original.
3. `POST /api/v1/files/{id}/recover-temp` → restaurar un temp específicamente a `temp/` aunque su storage original fuera otro (útil para recuperar previsualizaciones eliminadas).

## Config (env vars)

| Variable              | Default        | Descripción |
|-----------------------|----------------|-------------|
| `UPLOAD_DIR`          | `./uploads`    | Directorio raíz de archivos |
| `FILE_MAX_SIZE_IMAGE` | `10MB`         | Máximo para imágenes |
| `FILE_MAX_SIZE_VIDEO` | `200MB`        | Máximo para videos |
| `FILE_MAX_SIZE_AUDIO` | `50MB`         | Máximo para audios |
| `FILE_TEMP_MAX_AGE`   | `720h`         | Edad máxima de temp antes de purga (720h = 30 días) |
| `FILE_PURGE_CRON`     | `0 0 * * *`    | Expresión cron para purga de temporales |
| `ALLOWED_EXTS`        | imágenes/video/audio comunes | Extensiones permitidas por categoría |

## Package structure (propuesta)

```
internal/file/
├── handler.go     → Gin handlers (upload, serve, delete, restore, purge, list, recover-temp)
├── service.go     → Validación, lógica de negocio
├── store.go       → Filesystem operations (save, move, delete)
├── db.go          → CRUD sobre tabla files (PostgreSQL)
└── cron.go        → Tarea programada: purga de temporales > 1 mes a medianoche
```

Reemplaza/absorbe al actual `internal/image/` extendiendo su lógica a múltiples categorías.
