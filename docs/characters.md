# Characters

## Database Schema

### Migration: `00003_create_characters_table.sql`

```sql
-- +goose Up
CREATE TABLE characters (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_characters_name ON characters (name);
CREATE INDEX idx_characters_deleted_at ON characters (deleted_at);

-- +goose Down
DROP TABLE IF EXISTS characters;
```

### Campos

| Campo       | Tipo      | Descripción |
|-------------|-----------|-------------|
| `id`        | UUID      | Identificador único |
| `name`      | VARCHAR   | Nombre del personaje |
| `description` | TEXT    | Descripción o lore del personaje |
| `metadata`  | JSONB     | Metadatos adicionales (edad, género, estilo visual, etc.) |
| `created_at` | TIMESTAMPTZ | Fecha de creación |
| `updated_at` | TIMESTAMPTZ | Fecha de última modificación |
| `deleted_at` | TIMESTAMPTZ | Fecha de soft delete (NULL si activo) |

## File relation

Cada personaje puede tener múltiples archivos asociados. La relación se define mediante una tabla pivot:

### Migration: `00004_create_character_files_table.sql`

```sql
-- +goose Up
CREATE TABLE character_files (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    character_id UUID NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    file_id      UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    role         VARCHAR(63) NOT NULL DEFAULT 'reference',  -- reference | portrait | asset | etc.
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(character_id, file_id, role)
);

CREATE INDEX idx_character_files_character ON character_files (character_id);
CREATE INDEX idx_character_files_file ON character_files (file_id);

-- +goose Down
DROP TABLE IF EXISTS character_files;
```

### Campos

| Campo         | Tipo      | Descripción |
|---------------|-----------|-------------|
| `id`          | UUID      | Identificador único |
| `character_id` | UUID     | FK → characters.id |
| `file_id`     | UUID      | FK → files.id |
| `role`        | VARCHAR   | Rol del archivo: `reference` (referencia visual), `portrait` (retrato), `asset` (asset confiable), etc. |
| `created_at`  | TIMESTAMPTZ | Fecha de asociación |

## Storage

Los archivos de personajes siguen el sistema definido en `file-management.md`:

```
UPLOAD_DIR (./uploads)
└── characters/
    └── {character_id}/
        ├── images/
        ├── videos/
        └── audio/
```

- Los archivos se registran en `files` con `category` según su tipo y se vinculan al personaje via `character_files`.
- El soft delete de un personaje (`characters.deleted_at`) NO elimina automáticamente sus archivos. La limpieza queda a cargo del service layer.

## API Endpoints

| Método | Ruta | Descripción |
|--------|------|-------------|
| `GET`    | `/api/v1/characters` | Listar personajes |
| `POST`   | `/api/v1/characters` | Crear personaje |
| `GET`    | `/api/v1/characters/:id` | Obtener personaje con sus archivos |
| `PATCH`  | `/api/v1/characters/:id` | Actualizar personaje |
| `DELETE` | `/api/v1/characters/:id` | Soft delete personaje |
| `POST`   | `/api/v1/characters/:id/files` | Asociar archivo existente al personaje |
| `GET`    | `/api/v1/characters/:id/files` | Listar archivos del personaje |
| `DELETE` | `/api/v1/characters/:id/files/:fileId` | Desvincular archivo del personaje |

## Package structure (propuesta)

```
internal/character/
├── handler.go     → Gin handlers
├── service.go     → Lógica de negocio
├── store.go       → CRUD PostgreSQL
└── types.go       → Structs y request/response types
```
