# Generación de Video vía Studio

Flujo completo para generar videos usando el endpoint unificado.

> Consulta la arquitectura completa del sistema en [`internal/studio/ARCHITECTURE.md`](../../internal/studio/ARCHITECTURE.md).

## Endpoints

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/v1/studio/video/generate` | Iniciar generación de video |
| GET | `/api/v1/studio/video/status/:taskId` | Consultar estado de una tarea |
| DELETE | `/api/v1/studio/video/task/:taskId` | Cancelar tarea en ejecución |
| POST | `/api/v1/studio/video/preview` | Previsualizar payload sin enviarlo |

## Campos obligatorios de sesión

Todos los endpoints de generación requieren `project_id`, `scene_id`, `scene_code` y `take_number` para registrar el contexto en los logs:

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `project_id` | string | ID del proyecto |
| `scene_id` | string | ID de la escena |
| `scene_code` | string | Código de la escena |
| `take_number` | int | Número de take |
| `user_id` | int | ID del usuario (opcional, para logs) |

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

## Paso 2: Generar video

```javascript
const generation = await fetch('/api/v1/studio/video/generate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    model: "dreamina-seedance-2-0-260128",
    content: [
      {
        type: "text",
        text: "perro caminando por la playa, atardecer, cámara lenta"
      },
      {
        type: "image",
        id: "3553d9a0-81ee-4829-8106-b7f54c5780f0",
        name: "ref.png",
        text: "referencia de estilo"
      }
    ],
    ratio: "16:9",
    duration: 5,
    camerafixed: false,
    seed: "22",
    quality: "standard",
    quantity: 1,
    watermark: false,
    resolution: "720p",
    generate_audio: true,
    // Session tracking — obligatorio
    project_id: "uuid-del-proyecto",
    scene_id: "uuid-de-la-escena",
    scene_code: "ESC-001",
    take_number: 3,
    user_id: 1
  })
});
```

### Campos del payload

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `model` | string | sí | Nombre del modelo (ej: `dreamina-seedance-2-0-260128`) |
| `content` | array | sí | Array de items de contenido |
| `ratio` | string | no | Relación de aspecto: `16:9`, `9:16`, `1:1`, etc. |
| `duration` | number | no | Duración en segundos (1-60, default: 5) |
| `camerafixed` | bool | no | Cámara fija / estática |
| `seed` | string | no | Semilla para reproducibilidad |
| `quality` | string | no | `standard`, `high` |
| `quantity` | int | no | Cantidad de outputs |
| `watermark` | bool | no | Incluir marca de agua |
| `resolution` | string | no | `480p`, `720p`, `1080p` |
| `generate_audio` | bool | no | Generar pista de audio |
| `project_id` | string | **sí** | ID del proyecto |
| `scene_id` | string | **sí** | ID de la escena |
| `scene_code` | string | **sí** | Código de escena |
| `take_number` | int | **sí** | Número de take |
| `user_id` | int | no | ID del usuario |

### Items de content

Cada item en `content[]`:

**Tipo `text`** — prompt:
```json
{ "type": "text", "text": "descripción de la escena" }
```

**Tipo `image` / `video` / `audio`** — referencia:
```json
{
  "type": "image",
  "id": "3553d9a0-81ee-4829-8106-b7f54c5780f0",
  "name": "ref.png",
  "text": "descripción visual"
}
```

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `type` | string | sí | `text`, `image`, `video` o `audio` |
| `text` | string | no | Prompt textual o descripción |
| `id` | string | no | UUID del archivo en file store |
| `name` | string | no | Nombre original del archivo |

### Respuesta inmediata (tarea asíncrona)

```json
{
  "data": {
    "taskId": "cgt-20260515-abc123def456",
    "model": "dreamina-seedance-2-0-260128",
    "status": "running",
    "outputs": []
  },
  "success": true,
  "message": "created"
}
```

Los modelos Seedance devuelven `status: "running"` porque la generación es asíncrona. Hay que hacer polling hasta que el estado sea `"succeeded"` o `"failed"`.

## Paso 3: Hacer polling del estado

```javascript
async function pollStatus(taskId) {
  while (true) {
    const res = await fetch(`/api/v1/studio/video/status/${taskId}`);
    const data = await res.json();
    const { status, outputs } = data.data;

    if (status === 'succeeded') {
      return outputs;
    }
    if (status === 'failed') {
      throw new Error(data.data.error);
    }
    await new Promise(r => setTimeout(r, 3000));
  }
}

const outputs = await pollStatus(taskId);
// outputs → [
//   { url: "https://...mp4", localUrl: "/outputs/seedance_xxx.mp4", type: "video" }
// ]
```

### Respuesta cuando se completa

```json
{
  "data": {
    "status": "succeeded",
    "outputs": [
      {
        "url": "https://...bytepluses.com/seedance/.../output.mp4",
        "localUrl": "/outputs/seedance_1747246991123_def456ab.mp4",
        "type": "video"
      }
    ],
    "progress": null
  },
  "success": true,
  "message": "success"
}
```

## Gallery Sync (modelos gallery)

Para modelos como `dreamina-seedance-2-0-gallery`, los archivos de referencia se sincronizan automáticamente con la galería BytePlus antes de generar:

1. Verifica si cada archivo ya está sincronizado
2. Si no, lo sube a la galería (individualmente o por personaje)
3. Reemplaza la URL por `asset://<AssetId>`

Esto ocurre automáticamente — no requiere acciones adicionales del cliente. Ver [`docs/gallery-sync-flow.md`](../gallery-sync-flow.md) para más detalles.

## Sincronización manual de assets

Si quieres precargar archivos a la galería del modelo antes de generar:

```javascript
const syncResult = await fetch('/api/v1/studio/sync-asset', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    model_id: "uuid-del-modelo",
    file_id: "uuid-del-archivo"
  })
});
// syncResult.data.status → "active"
```

## Logs de generación

Cada generación se registra automáticamente con el contexto de proyecto/escena/take:

```javascript
// Listar logs filtrados por proyecto
const logs = await fetch('/api/v1/studio/logs/generation?project_id=X&page=1&limit=20')
  .then(r => r.json());
// logs.data → { logs: [...], total, page, limit, total_pages }

// Ver detalle de un log
const detail = await fetch(`/api/v1/studio/logs/generation/${logId}`)
  .then(r => r.json());
// detail.data → { id, task_id, model_name, project_id, scene_id,
//                  take_number, request, outputs, status, error_message, ... }
```

### Campos del log

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `id` | UUID | ID del registro |
| `task_id` | string | ID de la tarea |
| `model_name` | string | Modelo usado |
| `project_id` | string | Proyecto asociado |
| `scene_id` | string | Escena asociada |
| `scene_code` | string | Código de escena |
| `take_number` | int | Número de take |
| `request` | TEXT | Payload original del cliente |
| `outputs` | TEXT | Outputs generados (JSON) |
| `status` | string | `running`, `succeeded`, `failed` |
| `error_message` | string | Mensaje de error si falló |
| `user_name` | string | Nombre de usuario (join) |
| `project_name` | string | Nombre del proyecto (join) |
| `scene_name` | string | Nombre de la escena (join) |

## Server communications

Cada llamada a la API externa se registra para trazabilidad:

```javascript
const comms = await fetch('/api/v1/studio/logs/server-communications?task_id=X')
  .then(r => r.json());
// comms.data.logs → [{ model_name, endpoint, method, request_body,
//                       response_body, status_code, duration_ms, ... }]
```

## Generadores de video disponibles

| Generador | Match | Modelo | Tipo |
|-----------|-------|--------|------|
| `SeedanceGenerator` | Contiene "dreamina-seedance-2-0-260128" | Dreamina Seedance 2.0 | Async |
| `SeedanceGalleryGenerator` | Contiene "dreamina-seedance-2-0-gallery" | Dreamina Seedance 2.0 Gallery | Async + Gallery |

## Referencia de API externa

### BytePlus ModelArk (inferencia video)

| Parámetro | Valor |
|-----------|-------|
| Base URL | `https://ark.ap-southeast.bytepluses.com/api/v3` |
| Auth | `Authorization: Bearer <ark-api-key>` |
| Endpoint create | `POST /contents/generations/tasks` |
| Endpoint status | `GET /contents/generations/tasks/:id` |
| Output | `video/mp4` |
