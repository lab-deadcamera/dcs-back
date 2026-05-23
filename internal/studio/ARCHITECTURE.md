# Arquitectura del Sistema de Generación (Studio)

## Índice

1. [Visión General](#1-visión-general)
2. [Generators y Pipeline](#2-generators-y-pipeline)
3. [Flujo de Generación de Video](#3-flujo-de-generación-de-video)
4. [Flujo de Generación de Imagen](#4-flujo-de-generación-de-imagen)
5. [Flujo de Generación de Audio](#5-flujo-de-generación-de-audio)
6. [Flujo de Generación de Texto](#6-flujo-de-generación-de-texto)
7. [Sistema de Logs](#7-sistema-de-logs)
8. [Project / Scene / Take — Relación con Generaciones](#8-project--scene--take--relación-con-generaciones)
9. [Asset Sync (Galería BytePlus)](#9-asset-sync-galería-byteplus)
10. [Legacy vs Unified Pipeline](#10-legacy-vs-unified-pipeline)
11. [Registro de Generators](#11-registro-de-generators)
12. [Diagrama de Flujo General](#12-diagrama-de-flujo-general)

---

## 1. Visión General

El paquete `internal/studio` es el núcleo del sistema de generación de medios. Orquesta la comunicación con APIs de IA externas (Seedance, Seedream, Gemini, etc.) para generar video, imagen, audio y texto.

### Estructura de directorios

```
internal/studio/
├── ARCHITECTURE.md                 <- Este documento
├── service.go                      <- Servicio central (orquestación)
├── handler.go                      <- Handlers HTTP (legacy + sync + logs)
├── types.go                        <- Tipos compartidos (request, response, log, etc.)
├── gallery_models.go               <- Modelos que requieren sync con BytePlus gallery
├── assetsync_store.go              <- Store para model_assets (sync file ↔ modelo)
├── generation_log_store.go         <- Store para generation_logs
├── server_communication_store.go   <- Store para server_communications
├── generated_assets_service.go     <- Guarda assets generados al completarse una tarea
├── generated_assets_store.go       <- Store para generated_assets
├── signer.go                       <- Firma HMAC-SHA256 para API BytePlus (AK/SK)
├── seedance_handler.go             <- Handler legacy Seedance (video)
├── seedream_handler.go             <- Handler legacy Seedream (imagen)
├── audio/
│   ├── domain.go                   <- Interfaz AudioGenerator (sin implementar)
│   ├── handler.go                  <- Handler HTTP (devuelve "not implemented")
│   ├── types.go                    <- Tipos de request/response
│   └── generators/                 <- (directorio listo para futuros generadores)
├── image/
│   ├── domain.go                   <- Interfaz ImageGenerator
│   ├── handler.go                  <- Handler HTTP
│   ├── service.go                  <- Service adapter (convierte a unified)
│   ├── types.go                    <- Tipos de request/response
│   └── generators/
│       ├── gentypes.go            <- Resoluciones válidas
│       ├── seedream.go            <- SeedreamGenerator (implementación concreta)
│       └── gemini_nano.go         <- GeminiNanoGenerator (implementación concreta)
├── text/
│   ├── domain.go                   <- Interfaz TextGenerator (sin implementar)
│   ├── handler.go                  <- Handler HTTP (devuelve "not implemented")
│   ├── types.go                    <- Tipos de request/response
│   └── generators/                 <- (directorio listo para futuros generadores)
└── video/
    ├── domain.go                   <- Interfaz VideoGenerator
    ├── handler.go                  <- Handler HTTP
    ├── service.go                  <- Service adapter (convierte a unified)
    ├── types.go                    <- Tipos de request/response
    └── generators/
        ├── shared.go               <- Validaciones y constantes compartidas
        ├── dreamina_seedance.go    <- SeedanceGenerator (video, async)
        └── dreamina_seedance_gallery.go <- SeedanceGalleryGenerator (video + gallery sync)
```

### Capas

```
Handler (HTTP) → Service Adapter → core.Service (orquestación) → Generator (API externa)
                                                      ↕
                                              Stores (DB)
```

---

## 2. Generators y Pipeline

### Interfaces

Todos los generadores (video, imagen, audio, texto) implementan la interfaz `PipelineRunner` definida en `service.go`:

```go
type PipelineRunner interface {
    Match(modelName string) bool
    Validate(req *GeneratorRequest) error
    Generate(req *GeneratorRequest) (*GeneratorResult, error)
    GetStatus(taskID, apiKey, baseURL, endpoint string) (*GeneratorResult, error)
    CancelTask(taskID, apiKey, baseURL, endpoint string) error
    BuildPayload(req *GeneratorRequest) map[string]interface{}
    ContentType() string
    Name() string
}
```

Cada dominio define su propia interfaz específica (e.g. `VideoGenerator` en `video/domain.go`), pero como todas tienen los mismos métodos, estructuralmente son compatibles con `PipelineRunner` y se registran con `RegisterGenerator()`.

### PipelineRequest unificado

El `GeneratorRequest` (en `types.go`) es el formato interno que reciben todos los generadores:

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `Model` | string | Nombre del modelo |
| `Content` | []ContentItem | Array de contenido (texto, imágenes, videos, audios) |
| `Ratio` | string | Relación de aspecto (ej. "16:9", "9:16") |
| `Duration` | int | Duración en segundos (video/audio) |
| `CameraFixed` | bool | Cámara fija (sin movimiento) |
| `Seed` | string | Semilla para reproducibilidad |
| `Quality` | string | Calidad |
| `Quantity` | int | Cantidad de outputs |
| `Watermark` | bool | Incluir marca de agua |
| `Resolution` | string | Resolución ("480p", "720p", "1080p", "2K") |
| `GenerateAudio` | bool | Generar pista de audio (video) |
| `ImageMode` | string | Modo de imagen |
| `APIKey` | string | API key del modelo (desde DB) |
| `BaseURL` | string | URL base del proveedor (desde DB) |
| `Endpoint` | string | Endpoint del modelo (desde DB) |

### ContentItem

```go
type ContentItem struct {
    Type    string // "text", "image", "video", "audio"
    Text    string // prompt o descripción
    Name    string // nombre original del archivo
    ID      string // UUID del archivo en file store
    DataURL string // URL resuelta (populada por el servicio, no del cliente)
}
```

El `DataURL` se resuelve en `resolveContent()`:
- Si el archivo está sincronizado con el modelo → usa `ReferenceURI` (asset://id o URL directa)
- Si no → usa la URL pública de serve (`/api/v1/files/:id/serve`)

---

## 3. Flujo de Generación de Video

### Endpoints

| Método | Ruta | Handler | Función |
|--------|------|---------|---------|
| POST | `/api/v1/studio/video/generate` | `video.Handler.Generate` | Iniciar generación |
| GET | `/api/v1/studio/video/status/:taskId` | `video.Handler.GetStatus` | Consultar estado |
| DELETE | `/api/v1/studio/video/task/:taskId` | `video.Handler.CancelTask` | Cancelar tarea |
| POST | `/api/v1/studio/video/preview` | `video.Handler.PreviewPayload` | Previsualizar payload (dry-run) |

### Flujo completo

```
1. POST /studio/video/generate
   │
   ├─ video.Handler.Generate()
   │   └─ Bindea JSON a video.GenerateRequest
   │
   ├─ video.videoService.GenerateVideo()
   │   ├─ Convierte video.GenerateRequest → studio.StudioGenerateRequest
   │   ├─ ¿Es modelo gallery?
   │   │   ├─ Sí → core.GallerySyncContent() → sincroniza assets no sincronizados
   │   │   └─ No → continúa
   │   └─ core.GenerateUnified(unifiedReq)
   │
   ├─ core.Service.GenerateUnified()
   │   ├─ VALIDA: project_id, scene_id, scene_code, take_number (obligatorios)
   │   ├─ DEFER: guarda GenerationLog al final (pase lo que pase)
   │   ├─ Busca modelo en DB (providerStore.GetModelByName)
   │   ├─ Resuelve contenido (resolveContent) → archivos → DataURLs
   │   ├─ Convierte a GeneratorRequest (formato interno)
   │   ├─ Busca generator que haga Match (pickGenerator)
   │   ├─ Valida request contra el generator (gen.Validate)
   │   ├─ Construye payload real (gen.BuildPayload) → para loguear
   │   ├─ Ejecuta gen.Generate() → llama a API externa
   │   ├─ LOG: server_communication (request body, response, status, duration)
   │   ├─ LOG: actualiza GenerationLog (taskID, status, outputs, error)
   │   ├─ Trackea tarea en memoria (s.tasks[taskID])
   │   └─ Devuelve StudioGenerateResponse { taskId, model, status, outputs }
   │
   └─ Generadores concretos:
       ├─ SeedanceGenerator (dreamina-seedance-2-0-260128)
       │   ├─ POST a API → recibe taskId
       │   └─ Status inicial: "running" (async)
       │
       └─ SeedanceGalleryGenerator (dreamina-seedance-2-0-gallery)
           ├─ POST a API → recibe taskId
           └─ Status inicial: "running" (async)
```

### Status polling (GET /status/:taskId)

```
2. GET /studio/video/status/:taskId
   │
   ├─ video.Handler.GetStatus()
   │
   ├─ video.videoService.GetVideoStatus()
   │   └─ core.GetStatusUnified(taskID)
   │       └─ core.GetStatus(taskID)
   │
   ├─ core.Service.GetStatus()
   │   ├─ Busca en memoria (s.tasks[taskID])
   │   │   ├─ Encontrado → consulta al generator (gen.GetStatus)
   │   │   └─ NO encontrado → busca en DB (generation_logs)
   │   │       └─ statusFromLog(log) → consulta al generator por taskID
   │   │
   │   ├─ LOG: server_communication (GET request)
   │   │
   │   ├─ ¿Estado final (succeeded/failed)?
   │   │   ├─ Sí → updateLogWithFinalStatus() → actualiza generation_logs
   │   │   ├─ Sí (succeeded) → saveGeneratedAssets() → crea generated_assets
   │   │   └─ No → devuelve estado actual (running/processing)
   │   │
   │   └─ Devuelve StatusResult { status, outputs, error, raw }
   │
   └─ Generador (GetStatus):
       ├─ Consulta API externa GET /tasks/{taskId}
       ├─ "succeeded" → descarga video, guarda en ./outputs/
       │   └─ Output: URL pública + LocalURL (/outputs/seedance_*.mp4)
       ├─ "failed" → devuelve error
       └─ Otro estado → devuelve "running" / "processing"
```

### Gallery Sync (modelos gallery únicamente)

Cuando el modelo es tipo "gallery" (e.g. `dreamina-seedance-2-0-gallery`), antes de generar se ejecuta `GallerySyncContent()`:

1. Por cada `ContentItem` no-texto con un `ID` de archivo:
   - ¿Ya sincronizado? → usa `ReferenceURI` existente
   - ¿No sincronizado?
     - ¿El archivo pertenece a un personaje? → sincroniza personaje completo (`SyncCharacterAssets`)
     - ¿No pertenece a personaje? → sincroniza archivo individual (`SyncAsset`)
2. El `DataURL` se reemplaza con la URI de referencia (`asset://<AssetID>`)

### Generadores de Video

| Generador | Match | Tipo | Modelo |
|-----------|-------|------|--------|
| `SeedanceGenerator` | Contiene "dreamina-seedance-2-0-260128" | Async | Dreamina Seedance 2.0 |
| `SeedanceGalleryGenerator` | Contiene "dreamina-seedance-2-0-gallery" | Async + Gallery | Dreamina Seedance 2.0 Gallery |

---

## 4. Flujo de Generación de Imagen

### Endpoints

| Método | Ruta | Handler | Función |
|--------|------|---------|---------|
| POST | `/api/v1/studio/image/generate` | `image.Handler.Generate` | Iniciar generación |
| GET | `/api/v1/studio/image/status/:taskId` | `image.Handler.GetStatus` | Consultar estado |
| DELETE | `/api/v1/studio/image/task/:taskId` | `image.Handler.CancelTask` | Cancelar tarea |
| POST | `/api/v1/studio/image/preview` | `image.Handler.PreviewPayload` | Previsualizar payload |

### Flujo completo

```
POST /studio/image/generate
   │
   ├─ image.Handler.Generate()
   │
   ├─ image.imageService.GenerateImage()
   │   ├─ Convierte image.GenerateRequest → studio.StudioGenerateRequest
   │   │   NOTA: image.GenerateRequest tiene project_id/scene_id/scene_code/take_number OPCIONALES
   │   │         (a diferencia de video que son REQUIRED)
   │   └─ core.GenerateUnified(unifiedReq)
   │
   └─ core.Service.GenerateUnified()
       ├─ (mismo flujo que video)
       ├─ NOTA: si project_id está vacío, el log se guarda igual pero sin referencia a proyecto
       └─ Generadores:
           ├─ SeedreamGenerator (dreamina-seedream-4-pro-251224)
           │   ├─ POST a API externa → respuesta SÍNCRONA
           │   ├─ Status inmediato: "succeeded" (no async)
           │   └─ Outputs: URLs de imágenes
           │
           └─ GeminiNanoGenerator (gemini-nano-banana)
               ├─ POST a API Gemini → respuesta síncrona
               ├─ Status inmediato: "succeeded"
               ├─ Decodifica base64, guarda en ./outputs/
               └─ Output: LocalURL (/outputs/gemini_nano_*.png)
```

### Diferencia clave: Image vs Video

| Aspecto | Video | Imagen |
|---------|-------|--------|
| Project/Scene/Take | **REQUIRED** (`binding:"required"`) | **OPCIONAL** (sin `binding:"required"`) |
| Naturaleza | Async (Seedance) / Sync (Seedream, Gemini) | Mayormente síncrono |
| Gallery Sync | Sí (modelos gallery) | No |

### Generadores de Imagen

| Generador | Match | Tipo | Status inicial |
|-----------|-------|------|----------------|
| `SeedreamGenerator` | Contiene "dreamina-seedream-4-pro-251224" | Síncrono | "succeeded" |
| `GeminiNanoGenerator` | Contiene "gemini" | Síncrono | "succeeded" |

---

## 5. Flujo de Generación de Audio

**Estado: NO IMPLEMENTADO**

Los handlers en `audio/handler.go` devuelven `"audio generation not yet implemented"` para todos los endpoints.

Los endpoints están definidos en `main.go`:
- `POST /api/v1/studio/audio/generate`
- `GET /api/v1/studio/audio/status/:taskId`
- `DELETE /api/v1/studio/audio/task/:taskId`
- `POST /api/v1/studio/audio/preview`

La interfaz `AudioGenerator` está definida en `audio/domain.go` lista para implementar.

---

## 6. Flujo de Generación de Texto

**Estado: NO IMPLEMENTADO**

Los handlers en `text/handler.go` devuelven `"text generation not yet implemented"` para todos los endpoints.

Los endpoints están definidos en `main.go`:
- `POST /api/v1/studio/text/generate`
- `GET /api/v1/studio/text/status/:taskId`
- `DELETE /api/v1/studio/text/task/:taskId`
- `POST /api/v1/studio/text/preview`

La interfaz `TextGenerator` está definida en `text/domain.go` lista para implementar.

---

## 7. Sistema de Logs

El sistema tiene **tres tablas de log** que trabajan juntas:

### 7.1 Generation Logs (`generation_logs`)

**Propósito:** Registrar cada solicitud de generación con su resultado final.

**Store:** `GenerationLogStore` (`generation_log_store.go`)

**Campos principales:**
| Campo | Tipo | Descripción |
|-------|------|-------------|
| `id` | UUID | ID único |
| `task_id` | string | ID de tarea devuelto por el generador |
| `model_name` | string | Nombre del modelo usado |
| `user_id` | int? | Usuario que inició la generación |
| `project_id` | string | Proyecto asociado |
| `scene_id` | string | Escena asociada |
| `scene_code` | string | Código de escena |
| `take_number` | int | Número de take |
| `request_payload` | text | Payload completo del cliente (JSON) |
| `outputs` | text | URLs generadas (JSON) |
| `status` | string | Estado final ("running", "succeeded", "failed") |
| `error_message` | text | Mensaje de error si falló |
| `created_at` | timestamp | Fecha de creación |
| `updated_at` | timestamp | Fecha de última actualización |

**Mecanismo de guardado (`defer`):**

En `GenerateUnified()`, el log se guarda usando un `defer` que se ejecuta **siempre**, en cualquier camino de retorno (éxito o error):

```go
defer func() {
    if s.logStore == nil { return }
    // ...construye logEntry...
    logEntry := &GenerationLog{
        TaskID:     taskID,     // "<no-task>" si no se generó
        ModelName:  modelName,
        UserID:     intPtrOrNil(req.UserID),
        ProjectID:  req.ProjectID,
        SceneID:    req.SceneID,
        SceneCode:  req.SceneCode,
        TakeNumber: req.TakeNumber,
        Request:    string(reqBytes),
        Outputs:    outputs,
        Status:     status,     // "failed" por defecto
        ErrorMessage: errLog,
    }
    s.logStore.Create(logEntry)
}()
```

Esto garantiza que **toda generación** queda registrada, incluso si falla antes de contactar la API.

**Actualización asíncrona:**

Cuando una tarea asíncrona (video) se completa, el status polling detecta el estado final y actualiza el log:
- `updateLogWithFinalStatus()` → actualiza `outputs`, `status`, `error_message`
- `saveGeneratedAssets()` → crea registros en `generated_assets`

**Listado de logs:**

- `GET /api/v1/studio/logs/generation` → paginado, con filtros
- `GET /api/v1/studio/logs/generation/:id` → detalle completo con payload

El listado usa columnas ligeras (sin `request_payload` para evitar bloat de base64).
El detalle incluye columnas completas con JOIN a `users`, `projects`, `scenes`.

### 7.2 Server Communications (`server_communications`)

**Propósito:** Trazar cada llamada HTTP a APIs externas (request y response completos).

**Store:** `ServerCommunicationStore` (`server_communication_store.go`)

**Campos:**
| Campo | Descripción |
|-------|-------------|
| `task_id` | ID de tarea |
| `model_name` | Modelo al que se llamó |
| `endpoint` | URL completa del endpoint |
| `method` | HTTP method (POST/GET/DELETE) |
| `request_body` | Payload enviado |
| `response_body` | Respuesta recibida |
| `status_code` | HTTP status code |
| `duration_ms` | Duración en milisegundos |
| `error_message` | Error si ocurrió |

**¿Cuándo se crean?**

1. **En generate** (`GenerateUnified`): se loguea el `BuildPayload` + respuesta de `Generate`
2. **En status polling** (`GetStatus`, `statusFromLog`): se loguea cada consulta de estado
3. **En asset sync** (`AssetAPI.logComm`): se loguean las operaciones de la galería BytePlus

### 7.3 Generated Assets (`generated_assets`)

**Propósito:** Guardar las URLs de salida cuando una tarea asíncrona se completa exitosamente.

**Store:** `GeneratedAssetStore` (`generated_assets_store.go`)

**Campos:**
| Campo | Descripción |
|-------|-------------|
| `task_id` | Tarea que generó el asset |
| `model_name` | Modelo usado |
| `user_id`, `project_id`, `scene_id`, `scene_code`, `take_number` | Contexto de sesión |
| `original_url` | URL pública del asset generado |
| `local_path` | Ruta local si se descargó |
| `status` | "pending", "confirmed", "failed" |

**¿Cuándo se crean?** En `saveGeneratedAssets()`, llamado desde `GetStatus()` cuando el estado es "succeeded".

**Flujo completo de logs para una generación de video:**

```
1. POST /generate
   ├─ defer → GenerationLog creado (status="failed" por defecto, se actualiza si todo ok)
   │           taskID = "<no-task>" si falló antes de generar taskID
   └─ server_communication creado (request + response de la API)

2. GET /status/:taskId (polling)
   ├─ server_communication creado (cada consulta)
   ├─ ¿succeeded?
   │   ├─ GenerationLog actualizado (outputs, status="succeeded")
   │   └─ GeneratedAssets creados (status="pending")
   └─ ¿failed?
       └─ GenerationLog actualizado (status="failed", error_message)
```

---

## 8. Project / Scene / Take — Relación con Generaciones

### Datos de sesión obligatorios

**Para video, audio y texto:** `project_id`, `scene_id`, `scene_code` y `take_number` son **obligatorios** (`binding:"required"`).

**Para imagen:** son **opcionales** (sin `binding:"required"`), lo que permite usar el panel de imágenes independiente.

### ¿Dónde se persisten?

En **3 tablas** distintas:

1. **`generation_logs`**: almacena `project_id`, `scene_id`, `scene_code`, `take_number` directamente
2. **`generated_assets`**: almacena los mismos campos para cada asset generado
3. **`server_communications`**: NO almacena project/scene/take

### ¿Cómo se filtran?

- `GET /studio/logs/generation?project_id=X&scene_id=Y` → filtra por proyecto/escena
- `GET /studio/logs/generation?status=failed` → solo fallos

### Validación en GenerateUnified

```go
if req.ProjectID == "" || req.SceneID == "" || req.SceneCode == "" || req.TakeNumber <= 0 {
    return nil, fmt.Errorf("project_id, scene_id, scene_code and take_number are required")
}
```

Si falta alguno, la generación se rechaza **antes** de llamar a la API.

---

## 9. Asset Sync (Galería BytePlus)

### ¿Qué es?

El asset sync sube archivos locales (imágenes, videos, audios) a la **galería de activos de BytePlus** para que los modelos gallery puedan referenciarlos usando `asset://<AssetID>` en lugar de URLs públicas.

### Flujo de Sync

```
1. ¿Modelo gallery? (IsGalleryModel)
   │
   ├─ ¿Archivo ya sincronizado? → usa ReferenceURI existente
   │
   └─ ¿No sincronizado?
       ├─ ¿Pertenece a personaje?
       │   └─ SyncCharacterAssets → crea grupo de activos por personaje
       │       └─ Sube cada archivo del personaje al grupo
       │
       └─ ¿No pertenece a personaje?
           └─ SyncAsset individual
               ├─ Crea registro model_assets (status="syncing")
               ├─ CreateAsset (POST a BytePlus con URL pública)
               ├─ Poll hasta "Active" o "Failed" (~2 minutos, 20 intentos cada 3s)
               ├─ Actualiza model_assets (status="active"/"failed", ReferenceURI)
               └─ ReferenceURI = "asset://<AssetID>" (gallery) | URL directa (otros)
```

### Modelos Gallery

Definidos en `gallery_models.go`:

| Modelo | ReferenceURI |
|--------|-------------|
| `dreamina-seedance-2-0-gallery` | `asset://<AssetID>` |

Para agregar un nuevo modelo gallery, añadir su nombre al slice `GalleryModels`.

### Credenciales

Las credenciales AK/SK se resuelven en `effectiveCredentials()`:
1. Si el modelo tiene `access_key_id` / `secret_access_key` → usa esas
2. Si no → usa las globales `ASSET_ACCESS_KEY_ID` / `ASSET_SECRET_ACCESS_KEY`

### Firmas HMAC-SHA256

Las llamadas a la API BytePlus usan firma HMAC-SHA256 (implementada en `signer.go`), con los parámetros:
- Región: `ap-southeast-1`
- Servicio: `ark`
- Versión: `2024-01-01`
- Host: `open.byteplusapi.com`

---

## 10. Legacy vs Unified Pipeline

### Legacy (Selection-based)

Ruta: `POST /studio/generate-legacy`

Usa `Selection` (tipos anidados: `SelectionField`, `SelectionPrompt`, `DataRef`) y `ModelHandler` interface.

Handlers legacy:
- `SeedanceHandler` — coincide con "seedance" o "dreamina"
- `SeedreamHandler` — coincide con "seedream"

NO crean `GenerationLog` ni `ServerCommunication`. Solo trackean en memoria.

### Unified Pipeline

Rutas: `POST /studio/video/generate`, `POST /studio/image/generate`, etc.

Usa `GeneratorRequest` unificado y `PipelineRunner` interface.

**Siempre** crea:
- `GenerationLog` (via defer)
- `ServerCommunication` (request/response de API)
- `GeneratedAssets` (cuando se completa exitosamente)

### Estado actual

| Sistema | Handlers/Generators | Logs | Project/Scene/Take |
|---------|-------------------|------|-------------------|
| Legacy (Selection) | SeedanceHandler, SeedreamHandler | NO | NO |
| Unified (Pipeline) | SeedanceGenerator, SeedanceGalleryGenerator, SeedreamGenerator, GeminiNanoGenerator | SÍ | SÍ (video obligatorio, image opcional) |

La intención es migrar completamente de Legacy a Unified.

---

## 11. Registro de Generators

En `main.go`, los generators se registran en el `studio.Service`:

```go
studioSvc := studio.NewService(...)

// Handlers legacy
studioSvc.RegisterHandler(studio.NewSeedanceHandler(cfg.OutputsDir))
studioSvc.RegisterHandler(studio.NewSeedreamHandler(cfg.OutputsDir))

// Generators unificados
studioSvc.RegisterGenerator(videogens.NewSeedanceGenerator(cfg.OutputsDir))
studioSvc.RegisterGenerator(videogens.NewSeedanceGalleryGenerator(cfg.OutputsDir))
studioSvc.RegisterGenerator(studioimage.NewSeedreamGenerator(cfg.OutputsDir))
studioSvc.RegisterGenerator(studioimage.NewGeminiNanoGenerator(cfg.OutputsDir))
```

Para agregar un nuevo generador:
1. Crear struct que implemente `PipelineRunner`
2. En `Match()`, definir qué nombres de modelo activan este generador
3. Registrar con `RegisterGenerator()` en `main.go`

---

## 12. Diagrama de Flujo General

```
Cliente
   │
   ▼
POST /studio/{video,image,audio,text}/generate
   │
   ▼
domain.Handler (HTTP)
   │ Bindea JSON → domain.GenerateRequest
   ▼
domain.Service (adapter)
   │ Convierte → studio.StudioGenerateRequest
   ▼
core.Service.GenerateUnified()
   │
   ├─ VALIDAR project_id, scene_id, scene_code, take_number
   │
   ├─ DEFER → guardar GenerationLog (pase lo que pase)
   │
   ├─ Obtener modelo de DB (providerStore.GetModelByName)
   │
   ├─ Resolver contenido (resolveContent)
   │   └─ archivos → DataURLs (URL pública o asset://)
   │
   ├─ ¿Modelo gallery? → GallerySyncContent()
   │   └─ Sincroniza assets no sincronizados
   │
   ├─ Convertir a GeneratorRequest
   │
   ├─ Seleccionar generator (pickGenerator)
   │
   ├─ Validar (gen.Validate)
   │
   ├─ Construir payload (gen.BuildPayload)
   │
   ├─ EJECUTAR gen.Generate()
   │   │
   │   ├─ LOG: server_communication (request + response + duración)
   │   │
   │   ├─ Síncrono (Seedream, Gemini):
   │   │   └─ Outputs disponibles inmediatamente
   │   │
   │   └─ Asíncrono (Seedance, Gallery):
   │       └─ TaskID, status="running"
   │
   ├─ Actualizar GenerationLog (taskID, status, outputs)
   │
   └─ Devolver respuesta al cliente
       └─ { taskId, model, status, outputs }

CLIENTE (polling si async):
   │
   ▼
GET /studio/{domain}/status/:taskId
   │
   ▼
core.Service.GetStatus()
   │
   ├─ ¿En memoria? → gen.GetStatus (API externa)
   ├─ ¿No en memoria? → buscar en generation_logs → gen.GetStatus
   │
   ├─ LOG: server_communication (cada consulta)
   │
   ├─ ¿succeeded/failed?
   │   ├─ Actualizar GenerationLog (outputs, status)
   │   └─ saveGeneratedAssets() → GeneratedAssets (solo succeeded)
   │
   └─ Devolver estado actual
```

---

## Resumen de tablas DB involucradas

| Tabla | Propósito | Creada por |
|-------|-----------|------------|
| `generation_logs` | Log de cada generación | `GenerateUnified` (defer) |
| `server_communications` | Traza de llamadas a API externas | `GenerateUnified`, `GetStatus`, `AssetAPI` |
| `generated_assets` | Assets generados exitosamente | `saveGeneratedAssets` (en status polling) |
| `model_assets` | Archivos sincronizados con galería BytePlus | `SyncAsset`, `uploadAndTrackAsset` |
