# Generación de contenido vía Studio (unified payload)

Flujo completo usando el payload unificado de `/api/v1/studio/generate`.

## Endpoints disponibles

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/v1/studio/generate` | Nuevo payload unificado |
| POST | `/api/v1/studio/generate-legacy` | Legacy (basado en `Selection`) |
| GET | `/api/v1/studio/status/{taskId}` | Polling de estado con `outputs[]` |
| GET | `/api/v1/studio/status-legacy/{taskId}` | Polling formato legacy |
| DELETE | `/api/v1/studio/task/{taskId}` | Cancelar tarea |
| POST | `/api/v1/studio/sync-asset` | Sincronizar un archivo a la galería del modelo |
| POST | `/api/v1/studio/sync-character-assets` | Sincronizar TODOS los archivos de un personaje a un modelo |
| GET | `/api/v1/studio/synced-assets?model_id=x` | Listar assets sincronizados de un modelo |
| GET | `/api/v1/studio/files-with-sync?category=&storage=&trashed=` | Listar archivos con sus modelos sincronizados |
| GET | `/api/v1/studio/characters/:id/files-with-sync` | Listar archivos de un personaje con sus modelos sincronizados |

## Arquitectura de generadores

```
POST /api/v1/studio/generate
         │
         ▼
  Handler.GenerateUnified
         │
         ▼
  Service.GenerateUnified
         │
         ├── 1. providerStore.GetModelByName(name) → API key, URL, endpoint
         │
         ├── 2. resolveContent(content, modelID)
         │      │
         │      ├── ¿File sincronizado? (model_assets: active)
         │      │     └── Sí → DataURL = "asset://asset-id"  (asset library URI)
         │      │
         │      └── No → Lee archivo del disco → data:...;base64,...  (data URL)
         │
         ├── 3. pickGenerator(modelName) → Seedance | Seedream | ...
         │
         └── 4. Generator.Generate(req)
                  │
                  └── BytePlus ModelArk API
```

### Asset sync (pre-carga a la galería del modelo)

```
POST /api/v1/studio/sync-asset
         │
         ▼
  Service.SyncAsset
         │
         ├── 1. Busca modelo → AK/SK + default_asset_group_id
         ├── 2. Crea registro en model_assets (status: "syncing")
         ├── 3. CreateAsset(fileURL) a BytePlus asset library
         ├── 4. Poll GetAsset hasta "Active" (~20s-60s)
         └── 5. Actualiza model_assets (status: "active" | "failed")
```

Cuando el archivo está sincronizado, `resolveContent` usa `asset://asset-id` en vez de la data URL. Esto permite que BytePlus use el asset directamente de su galería sin re-subirlo cada vez.

## Paso 1: Subir archivos de referencia (opcional)

Antes de generar, puedes subir imágenes, videos o audio de referencia:

```javascript
async function uploadFile(file, category = 'images') {
  const form = new FormData();
  form.append('file', file);
  form.append('category', category);
  form.append('storage', 'persistent');
  const res = await fetch('/api/v1/files/upload', { method: 'POST', body: form });
  return res.json();
  // → { id: "file-uuid", filename: "...", url: "...", ... }
}
```

## Paso 1b: Sincronizar archivo a la galería del modelo (opcional pero recomendado)

Si el modelo tiene AK/SK configurados y un asset group, puedes sincronizar el archivo para usar `asset://` URIs:

```javascript
async function syncAsset(modelId, fileId) {
  const res = await fetch('/api/v1/studio/sync-asset', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ model_id: modelId, file_id: fileId })
  });
  return res.json();
}

const syncResult = await syncAsset(model.id, file.id);
// syncResult.data.status → "active"
// syncResult.data.asset_id → "asset-20260318-xxxxx"
```

### Requisitos para sync

Para sincronizar archivos, el modelo debe tener configurado:

| Campo | Ejemplo | Descripción |
|-------|---------|-------------|
| `access_key_id` | `AKLTxxx...` | Access Key de BytePlus (NO es el API key Bearer) |
| `secret_access_key` | `YzIx...` | Secret Key correspondiente |
| `default_asset_group_id` | `group-20260318-xxxxx` | Asset group creado en BytePlus Console |

> **Nota**: El API Key (Bearer) y el AK/SK son credenciales diferentes. El API Key se usa para la generación, el AK/SK para la API de asset library (CreateAsset, GetAsset, etc.).

El asset group debe crearse primero desde la [BytePlus Console](https://console.byteplus.com/ark/region:ark+ap-southeast-1/experience/vision) o vía API. Una vez creado, el ID del grupo se asigna al campo `default_asset_group_id` del modelo.

## Paso 2: Generar contenido (video)

El payload unificado permite referenciar archivos por su UUID:

```javascript
const generation = await fetch('/api/v1/studio/generate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    model: "dreamina-seedance-2-0-fast-260128",
    content: [
      {
        type: "text",
        text: "haz un perro caminando por la playa. shot on a 24mm wide lens, expansive framing with subtle edge distortion, deep depth of field. slow deliberate dolly-in toward the subject, smooth mechanical motion, tension building as the frame compresses, no handheld shake."
      },
      {
        type: "image",
        id: "3553d9a0-81ee-4829-8106-b7f54c5780f0",
        name: "image1.png",
        text: "Cyberpunk city street at night, neon lights, rain"
      }
    ],
    ratio: "16:9",
    duration: 5,
    camerafixed: false,
    seed: "22",
    quality: "standard",
    quantity: 1,
    watermark: false,
    resolution: "480p",
    generate_audio: true,
    image_mode: "PIL"
  })
});
```

Respuesta inmediata:

```json
{
  "data": {
    "taskId": "cgt-20260515-abc123def456",
    "model": "dreamina-seedance-2-0-fast-260128",
    "status": "running",
    "outputs": []
  },
  "success": true,
  "message": "created"
}
```

> Si el archivo está sincronizado, el content se envía como `"image_url": { "url": "asset://asset-20260318-xxxxx" }` en vez de una data URL con base64. Esto reduce el payload y permite a BytePlus usar el asset desde su galería interna.

### Campos del payload

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `model` | string | sí | Nombre del modelo registrado (ej: `dreamina-seedance-2-0-fast-260128`) |
| `content` | array | sí | Array de items de contenido (texto, imágenes, videos, audio) |
| `ratio` | string | no | Relación de aspecto: `16:9`, `9:16`, `1:1`, etc. |
| `duration` | number | no | Duración en segundos (default: 5) |
| `camerafixed` | bool | no | Cámara fija / estática |
| `seed` | string | no | Semilla para reproducibilidad |
| `quality` | string | no | Calidad: `standard`, `high` |
| `quantity` | int | no | Cantidad de outputs a generar |
| `watermark` | bool | no | Incluir marca de agua |
| `resolution` | string | no | Resolución: `480p`, `720p`, `1080p` |
| `generate_audio` | bool | no | Generar audio para el video |
| `image_mode` | string | no | Modo de imagen: `PIL` |

### Items de content

Cada item en `content[]` tiene esta estructura:

**Tipo `text`** — prompt de texto:
```json
{
  "type": "text",
  "text": "descripción de la escena"
}
```

**Tipo `image` / `video` / `audio`** — archivo de referencia:
```json
{
  "type": "image",
  "id": "3553d9a0-81ee-4829-8106-b7f54c5780f0",
  "name": "image1.png",
  "text": "descripción visual de esta referencia"
}
```

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `type` | string | sí | `text`, `image`, `video` o `audio` |
| `text` | string | no | Prompt textual o descripción del asset |
| `id` | string | no | UUID del archivo en file store (solo para `image`/`video`/`audio`) |
| `name` | string | no | Nombre original del archivo |

## Paso 3: Hacer polling del estado

```javascript
async function pollStatus(taskId) {
  while (true) {
    const res = await fetch(`/api/v1/studio/status/${taskId}`);
    const data = await res.json();
    const { status, outputs } = data.data;
    
    if (status === 'succeeded') {
      // outputs → [{ url, localUrl, type }]
      return outputs;
    }
    if (status === 'failed') {
      throw new Error(data.data.error);
    }
    // 'running' | 'queued' — seguir esperando
    await new Promise(r => setTimeout(r, 3000));
  }
}

const outputs = await pollStatus(taskId);
// outputs → [
//   { url: "https://...mp4", localUrl: "/outputs/seedance_xxx.mp4", type: "video" }
// ]
```

Respuesta cuando la tarea se completa:

```json
{
  "data": {
    "status": "succeeded",
    "outputs": [
      {
        "url": "https://ark-content-generation-v2-ap.tos-ap-southeast-1.bytepluses.com/seedance/.../output.mp4?X-Tos-Algorithm=...",
        "localUrl": "/outputs/seedance_1747246991123_def456ab.mp4",
        "type": "video"
      }
    ]
  },
  "success": true,
  "message": "success"
}
```

> La respuesta **siempre** contiene `outputs` como array, incluso para modelos que generan un solo recurso. Cada output tiene `type`: `"video"`, `"image"` o `"audio"`.

## Seedream (generación de imágenes síncrona)

Modelos Seedream son **síncronos** — la respuesta de `POST /studio/generate` ya incluye outputs:

```javascript
const result = await fetch('/api/v1/studio/generate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    model: "dreamina-seedream-4-pro-251224",
    content: [
      { type: "text", text: "Cyberpunk city street at night, neon lights" }
    ],
    resolution: "2K",
    watermark: false
  })
});
// → { data: { taskId: "seedream_xxx", model: "dreamina-seedream-4-pro-251224",
//     status: "succeeded", outputs: [{ url: "https://...", type: "image" }] } }
```

## Generadores disponibles

| Generador | Modelos | Match | Tipo de output |
|-----------|---------|-------|----------------|
| Seedance | `dreamina-seedance-2-0-fast-260128`, `dreamina-seedance-2-0-260128` | Nombre contiene "dreamina-seedance-2-0-fast-260128" | video (asíncrono) |
| Seedream | `dreamina-seedream-4-pro-251224` | Nombre contiene "dreamina-seedream-4-pro-251224" | image (síncrono) |

## Listar archivos con info de sincronización

Los endpoints `/api/v1/studio/files-with-sync` y `/api/v1/studio/characters/:id/files-with-sync` devuelven los mismos archivos que los endpoints regulares de `/api/v1/files` y `/api/v1/characters/:id/files`, pero incluyen el campo `synced_models` con los modelos con los que cada archivo está sincronizado.

### Files with sync

```javascript
const files = await fetch('/api/v1/studio/files-with-sync?category=images').then(r => r.json());
// files.data → [
//   {
//     "id": "uuid",
//     "filename": "photo.png",
//     "mime_type": "image/png",
//     "synced_models": [
//       { "model_id": "uuid", "name": "dreamina-seedance-2-0-fast-260128" }
//     ]
//   }
// ]
```

### Character files with sync

```javascript
const files = await fetch(`/api/v1/studio/characters/${charId}/files-with-sync`).then(r => r.json());
// files.data → [
//   {
//     "file_id": "uuid",
//     "role": "portrait",
//     "url": "http://.../serve",
//     "synced_models": [
//       { "model_id": "uuid", "name": "dreamina-seedance-2-0-260128" }
//     ]
//   }
// ]
```

## Sincronizar todos los archivos de un personaje

Para sincronizar todas las imágenes de un personaje a un modelo de una sola vez:

```javascript
const result = await fetch('/api/v1/studio/sync-character-assets', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    character_id: "uuid-del-personaje",
    model_id: "uuid-del-modelo"
  })
});

// result.data → {
//   "model_id": "uuid",
//   "total": 5,
//   "successful": 4,
//   "failed": 1,
//   "results": [
//     { "file_id": "uuid", "status": "active", "asset_id": "asset-xxx" },
//     { "file_id": "uuid", "status": "failed", "error_message": "..." },
//     ...
//   ]
// }
```

Para sincronizar solo una imagen específica, usa el endpoint individual:

```javascript
await fetch('/api/v1/studio/sync-asset', {
  method: 'POST',
  body: JSON.stringify({ model_id: modelId, file_id: fileId })
});
```

## Generadores disponibles

| Generador | Modelos | Match | Tipo de output |
|-----------|---------|-------|----------------|
| Seedance | `dreamina-seedance-2-0-fast-260128`, `dreamina-seedance-2-0-260128` | Nombre contiene "dreamina-seedance-2-0-fast-260128" | video (asíncrono) |
| Seedream | `dreamina-seedream-4-pro-251224` | Nombre contiene "dreamina-seedream-4-pro-251224" | image (síncrono) |

Para agregar un nuevo generador, crea un archivo en `internal/studio/generators/` que implemente la interfaz `Generator` y regístralo en `main.go`.

## Referencia de APIs externas

### BytePlus ModelArk (inferencia)

| Parámetro | Seedance (video) | Seedream (imagen) |
|-----------|-----------------|-------------------|
| Base URL | `https://ark.ap-southeast.bytepluses.com/api/v3` | Misma |
| Auth | `Authorization: Bearer <ark-api-key>` | Misma |
| Endpoint create | `POST /contents/generations/tasks` | `POST /images/generations` |
| Endpoint status | `GET /contents/generations/tasks/:id` | N/A (síncrono) |
| Output | `video/mp4` | `image/png` o `image/jpeg` |

### BytePlus Asset Library (sync)

| Parámetro | Valor |
|-----------|-------|
| Host | `open.byteplusapi.com` |
| Auth | AK/SK (HMAC-SHA256 signed request) |
| Region | `ap-southeast-1` |
| Service | `ark` |
| API Version | `2024-01-01` |
| Crear asset group | `CreateAssetGroup` |
| Subir asset | `CreateAsset` (requiere URL pública del archivo) |
| Consultar asset | `GetAsset` (hasta que status = "Active") |
