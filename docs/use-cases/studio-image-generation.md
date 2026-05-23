# Generación de Imagen vía Studio

Flujo para generar imágenes usando el endpoint unificado de imagen.

> Consulta la arquitectura completa del sistema en [`internal/studio/ARCHITECTURE.md`](../../internal/studio/ARCHITECTURE.md).

## Endpoints

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/v1/studio/image/generate` | Generar imagen |
| GET | `/api/v1/studio/image/status/:taskId` | Consultar estado |
| DELETE | `/api/v1/studio/image/task/:taskId` | Cancelar tarea |
| POST | `/api/v1/studio/image/preview` | Previsualizar payload |

## Contexto de sesión

Todos los endpoints requieren `project_id`, `scene_id`, `scene_code` y `take_number` para registrar el contexto de la generación:

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `project_id` | string | ID del proyecto |
| `scene_id` | string | ID de la escena |
| `scene_code` | string | Código de la escena |
| `take_number` | int | Número de take |
| `user_id` | int | ID del usuario (opcional) |

## Generar imagen

Los modelos de imagen son **síncronos** — la respuesta incluye los outputs inmediatamente, no requiere polling.

### Seedream

```javascript
const result = await fetch('/api/v1/studio/image/generate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    model: "dreamina-seedream-4-pro-251224",
    content: [
      { type: "text", text: "Cyberpunk city street at night, neon lights, rain" }
    ],
    resolution: "2K",
    watermark: false,
    // Session tracking
    project_id: "uuid-del-proyecto",
    scene_id: "uuid-de-la-escena",
    scene_code: "ESC-001",
    take_number: 2,
    user_id: 1
  })
});

// Respuesta:
// {
//   "data": {
//     "taskId": "seedream_1747246991123",
//     "model": "dreamina-seedream-4-pro-251224",
//     "status": "succeeded",
//     "outputs": [
//       { "url": "https://...", "type": "image" }
//     ]
//   }
// }
```

### Gemini Nano Banana

```javascript
const result = await fetch('/api/v1/studio/image/generate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    model: "gemini-nano-banana",
    content: [
      { type: "text", text: "un bosque mágico con luces brillantes" },
      {
        type: "image",
        id: "file-uuid",
        name: "referencia.png"
      }
    ],
    resolution: "1080p",
    project_id: "uuid-del-proyecto",
    scene_id: "uuid-de-la-escena",
    scene_code: "ESC-001",
    take_number: 2,
    user_id: 1
  })
});

// Respuesta:
// {
//   "data": {
//     "taskId": "gemini_1747246991123",
//     "model": "gemini-nano-banana",
//     "status": "succeeded",
//     "outputs": [
//       { "url": "/outputs/gemini_6991123_0.png", "type": "image" }
//     ]
//   }
// }
```

### Campos del payload

| Campo | Tipo | Requerido | Descripción |
|-------|------|-----------|-------------|
| `model` | string | sí | `dreamina-seedream-4-pro-251224` o `gemini-nano-banana` |
| `content` | array | sí | Items de contenido (texto + imágenes referencia) |
| `ratio` | string | no | Relación de aspecto |
| `seed` | string | no | Semilla para reproducibilidad |
| `quality` | string | no | `standard`, `high` |
| `quantity` | int | no | Cantidad de outputs |
| `watermark` | bool | no | Incluir marca de agua |
| `resolution` | string | no | `2K`, `1080p`, `720p` |
| `project_id` | string | **sí** | ID del proyecto |
| `scene_id` | string | **sí** | ID de la escena |
| `scene_code` | string | **sí** | Código de escena |
| `take_number` | int | **sí** | Número de take |
| `user_id` | int | no | ID del usuario |

## Sincronización de assets

Los modelos de imagen **no** requieren gallery sync. Los archivos de referencia se envían como URLs públicas.

## Logs

Cada generación se registra automáticamente:

```javascript
const logs = await fetch('/api/v1/studio/logs/generation?model_name=seedream&project_id=X')
  .then(r => r.json());
```

## Generadores de imagen disponibles

| Generador | Match | Tipo | Status inicial |
|-----------|-------|------|----------------|
| `SeedreamGenerator` | Contiene "dreamina-seedream-4-pro-251224" | Síncrono | `succeeded` |
| `GeminiNanoGenerator` | Contiene "gemini" | Síncrono | `succeeded` |

## Referencia de API externa

### BytePlus (Seedream)

| Parámetro | Valor |
|-----------|-------|
| Auth | `Authorization: Bearer <ark-api-key>` |
| Endpoint | `POST /images/generations` |
| Output | `image/png` o `image/jpeg` |

### Google Gemini (Gemini Nano)

| Parámetro | Valor |
|-----------|-------|
| Auth | `x-goog-api-key` header |
| Output | `image/png` (guardado en `/outputs/`) |
