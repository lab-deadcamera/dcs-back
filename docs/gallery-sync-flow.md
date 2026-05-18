# Gallery Sync — SeedanceGalleryGenerator

## ¿Qué es?

El `SeedanceGalleryGenerator` (modelo `dreamina-seedance-2-0-gallery`) extiende el generador Seedance estándar con sincronización automática de activos a la **galería privada del modelo** (BytePlus Asset Library).

Cuando un usuario envía referencias (imágenes, videos, audio) para generar un video, el sistema:

1. Verifica si cada activo ya está sincronizado con la galería del modelo.
2. Si NO lo está, lo sube automáticamente a la galería antes de generar.
3. Si el activo pertenece a un personaje, sincroniza **todos los activos de ese personaje** como un grupo.
4. Reemplaza la URL pública por una `asset://<AssetId>` para que el modelo use su propia galería.

---

## Flujo completo

```
Cliente
  │
  ▼
POST /studio/video/generate  (model: "dreamina-seedance-2-0-gallery-*")
  │
  ▼
video/service.go ── GenerateVideo()
  │
  ├─ ¿El modelo contiene "gallery"?
  │      │
  │      ├─ NO  → llama a core.GenerateUnified() directamente
  │      │
  │      └─ SÍ  → llama a core.GallerySyncContent(content, modelName)
  │                  │
  │                  ▼
  │             Por cada item en content[]:
  │                  │
  │                  ├─ type == "text" ──→ saltar
  │                  │
  │                  ├─ ¿Ya sync? (AssetSyncStore)
  │                  │      │
  │                  │      ├─ SÍ → usar asset://<AssetId>
  │                  │      │
  │                  │      └─ NO → continuar
  │                  │
  │                  ├─ ¿Pertenece a un personaje? (character_files)
  │                  │      │
  │                  │      ├─ SÍ → SyncCharacterAssets()
  │                  │      │       └─ Crea grupo → sube todos los assets del personaje
  │                  │      │       └─ asset://<AssetId>
  │                  │      │
  │                  │      └─ NO → SyncAsset()
  │                  │              └─ Sube el asset individual
  │                  │              └─ asset://<AssetId>
  │                  │
  │                  └─ ContentItem.DataURL = "asset://<AssetId>"
  │
  └─ core.GenerateUnified() con content resuelto
       │
       └─ Generador recibe asset:// URIs y las pasa al API de BytePlus
```

---

## Componentes

### 1. Disparador — `video/service.go`

```go
func (s *videoService) GenerateVideo(req *GenerateRequest) (*GenerateResponse, error) {
    unified := toStudioRequest(req)

    // Si el modelo es gallery, sincroniza activos primero
    if studio.IsGalleryModel(req.Model) {
        synced, err := s.core.GallerySyncContent(unified.Content, req.Model)
        if err == nil {
            unified.Content = synced  // reemplaza URLs con asset://
        }
    }

    result, err := s.core.GenerateUnified(unified)
    // ...
}
```

### 2. Sincronizador — `studio.Service.GallerySyncContent()`

Método en `studio/service.go` que orquesta toda la sincronización:

```
Entrada:  []ContentItem (con file IDs y URLs públicas)
Salida:   []ContentItem (con DataURL = "asset://<AssetId>")
```

Para cada ítem no-text:

1. **Lookup en AssetSyncStore** — consulta `model_assets` DB:
   ```sql
   SELECT * FROM model_assets WHERE model_id = ? AND file_id = ? AND status = 'active'
   ```

2. **Si no está sincronizado** — determina estrategia:
   - **Personaje**: llama a `charService.FindCharactersByFileID(fileID)` → obtiene character IDs
   - Si hay personajes, llama a `SyncCharacterAssets()` que:
     - Crea un **AssetGroup** con el nombre del personaje
     - Sube **todos los archivos** del personaje a ese grupo
     - Retorna los Asset IDs
   - **Sin personaje**: llama a `SyncAsset()` que:
     - Sube el archivo individual al grupo por defecto del modelo
     - Espera hasta que esté `Active`

3. **Reemplaza DataURL** con `asset://<AssetId>`

### 3. Subida a BytePlus — `AssetAPI` (`studio/signer.go`)

Las llamadas reales a la API de BytePlus usan **AK/SK** (Access Key / Secret Key) del modelo:

```
CreateAssetGroup(name, description)
  → POST SignedFetch (HMAC-SHA256)
  → Response: { "Id": "group-20260318-xxxxx" }

CreateAsset(publicURL, name, assetType, projectName)
  → POST SignedFetch
  → Body: { GroupId, URL, AssetType, Name }
  → Response: { "Id": "asset-20260318-xxxxx" }
  → NOTA: Es asíncrono. Hay que poll hasta Status == "Active"

GetAsset(assetId, projectName)
  → POST SignedFetch
  → Body: { Id, ProjectName }
  → Response: { "Status": "Active"|"Processing"|"Failed", "URL": "..." }
```

El `SignedFetch` firma cada request con HMAC-SHA256 usando el algoritmo BytePlus Signature V4.

### 4. Almacenamiento local — `AssetSyncStore`

La tabla `model_assets` en la DB local mantiene el mapping entre:

| Campo | Descripción |
|---|---|
| `id` | UUID del registro de sync |
| `model_id` | ID del modelo (BytePlus) |
| `file_id` | UUID del archivo en file store |
| `asset_id` | ID del asset en BytePlus Gallery |
| `asset_group_id` | Grupo al que pertenece (si aplica) |
| `status` | `syncing` → `active` / `failed` |
| `error_message` | Mensaje de error si falló |

---

## Requisitos del modelo

Para que la galería funcione, el modelo debe tener configurado:

```
access_key_id         → AK para autenticación BytePlus
secret_access_key     → SK para autenticación BytePlus
default_asset_group_id → Grupo por defecto para assets sin personaje
```

Sin AK/SK, `GallerySyncContent()` retorna error y la generación cae al flujo normal (URLs públicas).

---

## Sincronización por personaje

Cuando un archivo pertenece a un personaje (tabla `character_files`):

```
1. Buscar personajes para el file:
   SELECT DISTINCT character_id FROM character_files WHERE file_id = ?

2. Para cada personaje:
   a. Obtener datos del personaje (name, description)
   b. Llamar a SyncCharacterAssets:
      - Busca grupo existente para ese personaje + modelo
      - Si no existe: CreateAssetGroup(name=character.Name)
      - Sube CADA archivo del personaje al grupo
      - Poll hasta Active
      - Guarda mapping en model_assets

3. El asset del item actual queda con asset://<AssetId>
```

Esto asegura que **todos los assets de un personaje** estén disponibles en la galería bajo un mismo grupo, permitiendo al modelo Seedance usarlos como referencias coherentes.

---

## Modelo de datos (DB)

La tabla `server_communications` registra cada llamada HTTP al API de BytePlus para trazabilidad:

```sql
CREATE TABLE server_communications (
    id              UUID PRIMARY KEY,
    task_id         TEXT NOT NULL,
    model_name      TEXT NOT NULL,
    endpoint        TEXT NOT NULL,
    method          TEXT NOT NULL,
    request_body    TEXT,
    response_body   TEXT,
    status_code     INT DEFAULT 0,
    duration_ms     BIGINT DEFAULT 0,
    error_message   TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

Endpoints de consulta:
```
GET /studio/logs/server-communications            → Lista paginada
GET /studio/logs/server-communications?task_id=X  → Filtrado por tarea
GET /studio/logs/server-communications?model_name=X → Filtrado por modelo
GET /studio/logs/server-communications/:id        → Detalle individual
```

---

## Registro de modelos Gallery

Los modelos que realizan auto-sync se definen en `internal/studio/gallery_models.go`:

```go
var GalleryModels = []string{
    "dreamina-seedance-2-0-gallery",
}

func IsGalleryModel(modelName string) bool { ... }
```

Para agregar un nuevo modelo a la sincronización, basta con añadir su nombre al slice `GalleryModels`. La función `IsGalleryModel()` usa `strings.Contains` para matching parcial, por lo que un modelo llamado `dreamina-seedance-2-0-gallery-260128` matchea correctamente.

---

## Resumen del flujo de datos

```
┌──────────┐     POST /studio/video/generate     ┌──────────────────┐
│ Cliente  │ ──────────────────────────────────→  │ video/service.go │
│          │     { model: "gallery",              │ GenerateVideo()  │
│          │       content: [                      │                  │
│          │         { type:"text", text:"..." },  │  ¿gallery?       │
│          │         { type:"image", id:"..." },   │     │            │
│          │         { type:"video", id:"..." }    │     ├─ SÍ        │
│          │       ]                               │     │            │
│          │     }                                 │     ▼            │
└──────────┘                                       │ GallerySync     │
                                                   │   Content()      │
                                                   │     │            │
                                                   │     ▼            │
                                                   │ ¿Ya sync?        │
                                                   │ ├─ SÍ → asset:// │
                                                   │ └─ NO            │
                                                   │      ├─ ¿Char?   │
                                                   │      │  ├─ SÍ →  │
                                                   │      │  │  Sync  │
                                                   │      │  │  Group │
                                                   │      │  └─ NO →  │
                                                   │      │     Sync  │
                                                   │      │     File  │
                                                   │      ▼           │
                                                   │ asset://AssetId  │
                                                   │                  │
                                                   │ GenerateUnified()│
                                                   │     │            │
                                                   │     ▼            │
                                                   │ BytePlus API     │
                                                   │ (Seedance model) │
                                                   │     │            │
                                                   │     ▼            │
                                                   │ Video generado   │
                                                   └──────────────────┘
                                                            │
                                                            ▼
                                                   ┌──────────────────┐
                                                   │    Cliente       │
                                                   │ { taskId,        │
                                                   │   outputs: [...] │
                                                   │ }                │
                                                   └──────────────────┘
```
