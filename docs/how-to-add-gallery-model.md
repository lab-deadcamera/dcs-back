# Cómo agregar un modelo con sincronización de galería

## ¿Qué es la sincronización de galería?

Ciertos modelos de IA (ej: Seedance Gallery) requieren que los assets de referencia (imágenes, videos) estén subidos a su galería privada en BytePlus antes de generar. El sistema sincroniza automáticamente los archivos antes de enviar el payload al modelo, reemplazando las URLs públicas por `asset://<AssetId>`.

---

## Paso a paso

### 1. Agregar el nombre del modelo a `GalleryModels`

Archivo: `internal/studio/gallery_models.go`

```go
var GalleryModels = []string{
    "dreamina-seedance-2-0-gallery",
    "mi-nuevo-modelo-gallery",   // <-- agregar aquí
}
```

`IsGalleryModel()` usa `strings.Contains` para matching parcial, por lo que un modelo llamado `mi-nuevo-modelo-gallery-v2` matchea automáticamente.

### 2. Configurar credenciales del modelo

Cada modelo necesita sus propias credenciales de BytePlus. Se configuran de dos formas:

**Opción A — En la BD (por modelo):**

| Columna en `models` | Ejemplo |
|---|---|
| `access_key_id` | `AKTP0VyX37N...` |
| `secret_access_key` | `SkP0VyX37N...` |
| `default_asset_group_id` | `group-abc123` |

**Opción B — Variables de entorno (global para todos):**

```
ASSET_ACCESS_KEY_ID=AKTP0VyX37N...
ASSET_SECRET_ACCESS_KEY=SkP0VyX37N...
ASSET_DEFAULT_GROUP_ID=group-abc123
```

La resolución está en `effectiveCredentials()`: si el modelo no tiene credenciales propias, usa las de entorno.

### 3. Si el modelo es de tipo image — agregar el hook de sync

`video/service.go` ya tiene el hook. Si tu modelo es de tipo image, agrégalo en `image/service.go`:

```go
func (s *imageService) GenerateImage(req *GenerateRequest) (*GenerateResponse, error) {
    unified := toStudioRequest(req)

    if studio.IsGalleryModel(req.Model) {
        synced, err := s.core.GallerySyncContent(unified.Content, req.Model)
        if err == nil {
            unified.Content = synced
        }
    }

    result, err := s.core.GenerateUnified(unified)
    // ...
}
```

Sin este hook, el modelo image generará con URLs públicas en lugar de `asset://`.

### 4. Asegurar que el generador soporte `asset://` URIs

El generador del modelo (el que implementa `PipelineRunner`) recibe el `GeneratorRequest.Content` con `DataURL` en formato `asset://<AssetId>`. El payload que construye `BuildPayload()` debe pasar ese valor tal cual al API de BytePlus.

Ejemplo en `dreamina_seedance_gallery.go` — no se necesita lógica especial porque `asset://` es el formato que espera BytePlus.

### 5. Correr la migración

```bash
goose up
```

Esto agrega las columnas `asset_url` y `asset_type` a `model_assets` si no existen.

---

## Verificación

### Probar que el modelo se detecta como gallery

```go
studio.IsGalleryModel("mi-nuevo-modelo-gallery-v2")  // → true
```

### Ver assets sincronizados en la BD

```sql
SELECT id, model_id, file_id, asset_id, asset_url, asset_type, status
FROM model_assets
WHERE model_id = '<model-uuid>'
ORDER BY created_at DESC;
```

### Logs de comunicación con BytePlus

```
GET /studio/logs/server-communications?model_name=mi-nuevo-modelo-gallery
```

Muestra cada llamada `CreateAsset`, `GetAsset`, etc.

---

## Resumen del flujo

```
POST /generate (model: "mi-modelo-gallery")
  │
  ├─ IsGalleryModel? ── SÍ ──┐
  │                           ▼
  │                    GallerySyncContent()
  │                      ├─ ¿Asset ya sincronizado? → asset://AssetId
  │                      └─ No → upload a BytePlus → poll Active → asset://AssetId
  │                           │
  │                           └─ Se guarda en model_assets:
  │                                asset_id, asset_url, asset_type (MAYUSCULA)
  │
  └─ GenerateUnified() con asset:// URIs en content
       │
       └─ El payload se envía al API con las asset:// URIs
```

## Lo que NO hay que hacer

- **No** crear un nuevo `*GalleryGenerator` — el sync es independiente del generador. Un modelo puede usar un generador existente y solo agregar el sync.
- **No** modificar `GallerySyncContent()` — ya es agnóstico del modelo y del tipo de contenido (image/video/audio).
- **No** tocar `AssetSyncStore` — la persistencia es genérica.
