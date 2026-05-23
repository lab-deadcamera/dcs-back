# Flujo de Logs — `/studio/**/generate`

## Tablas involucradas

| Tabla | Propósito | Creada por |
|-------|-----------|------------|
| `generation_logs` | Log de cada solicitud de generación | `GenerateUnified` (defer) y actualizado por `updateLogWithFinalStatus` |
| `server_communications` | Traza de cada llamada HTTP a APIs externas | `GenerateUnified`, `GetStatus`, `statusFromLog`, `AssetAPI.logComm` |
| `generated_assets` | URLs de salida de tareas completadas exitosamente | `saveGeneratedAssets` (desde `GetStatus`) |

---

## 1. Flujo para generación asíncrona (video)

### POST /studio/video/generate

```
POST /studio/video/generate (modelo Seedance, async)
  │
  ├─ 1. core.Service.GenerateUnified()
  │      │
  │      ├─ Valida project_id, scene_id, scene_code, take_number
  │      │
  │      ├─ DEFER: crea generation_log (status="failed" por defecto)
  │      │     └─ Campos: task_id="<no-task>", model_name, user_id,
  │      │         project_id, scene_id, scene_code, take_number,
  │      │         request_payload (JSON), status="failed", error_message
  │      │
  │      ├─ Busca modelo en DB → obtiene API key, URL, endpoint
  │      │
  │      ├─ Resuelve contenido (resolveContent)
  │      │   └─ Archivos → DataURLs (URL pública o asset://)
  │      │
  │      ├─ Busca generator (pickGenerator)
  │      │
  │      ├─ Construye payload real (BuildPayload) → guardado para server_communication
  │      │
  │      ├─ gen.Generate() → llama a API externa
  │      │   ├─ LOG: server_communication
  │      │   │   └─ model_name, endpoint, method="POST",
  │      │   │      request_body (BuildPayload), response_body,
  │      │   │      status_code, duration_ms
  │      │   │
  │      │   └─ Obtiene taskId de la respuesta
  │      │
  │      ├─ Actualiza generation_log (vía defer):
  │      │   └─ task_id="cgt-xxx", status="running" (o "failed" si error)
  │      │
  │      └─ Devuelve { taskId, model, status: "running", outputs: [] }
```

**Registros creados hasta aquí:**
- `generation_logs` → 1 registro (task_id asignado, status="running")
- `server_communications` → 1 registro (POST a API externa)

### GET /studio/video/status/:taskId (polling)

```
GET /studio/video/status/:taskId (polling, cada ~3s)
  │
  ├─ core.Service.GetStatus(taskID)
  │   │
  │   ├─ Busca en memoria (s.tasks)
  │   │   ├─ ¿Encontrado? → consulta gen.GetStatus()
  │   │   └─ ¿No encontrado? (reinicio) → busca en generation_logs
  │   │       └─ statusFromLog(log) → consulta gen.GetStatus()
  │   │
  │   ├─ LOG: server_communication (cada consulta)
  │   │   └─ task_id, model_name, endpoint, method="GET",
  │   │      request_body={"task_id":"..."}, response_body,
  │   │      status_code, duration_ms
  │   │
  │   ├─ ¿Estado "succeeded"?
  │   │   ├─ updateLogWithFinalStatus(taskID, result)
  │   │   │   └─ UPDATE generation_logs SET
  │   │   │       outputs='[{url, localUrl, type}]',
  │   │   │       status='succeeded',
  │   │   │       error_message=''
  │   │   │
  │   │   └─ saveGeneratedAssets(taskID, result)
  │   │       └─ Por cada output:
  │   │           INSERT INTO generated_assets
  │   │           (task_id, model_name, user_id, project_id,
  │   │            scene_id, scene_code, take_number,
  │   │            original_url, status='pending')
  │   │
  │   ├─ ¿Estado "failed"?
  │   │   └─ updateLogWithFinalStatus(taskID, result)
  │   │       └─ UPDATE generation_logs SET
  │   │           outputs='[]', status='failed',
  │   │           error_message='motivo del error'
  │   │
  │   └─ ¿Estado "running"/"processing"?
  │       └─ Devuelve estado actual (sin cambios en DB)
  │
  └─ Devuelve StatusResult { status, outputs, error, raw }
```

**Registros creados/actualizados durante polling:**
- `server_communications` → 1 registro por cada consulta de estado
- `generation_logs` → 1 UPDATE cuando la tarea se completa
- `generated_assets` → N registros (uno por output) cuando succeeded

---

## 2. Flujo para generación síncrona (imagen)

### POST /studio/image/generate

```
POST /studio/image/generate (modelo Seedream/Gemini, síncrono)
  │
  └─ core.Service.GenerateUnified()
       │
       ├─ (mismo flujo que video hasta gen.Generate())
       │
       ├─ gen.Generate() → respuesta INMEDIATA con outputs
       │   ├─ LOG: server_communication
       │   └─ Resultado: status="succeeded", outputs=[...]
       │
       ├─ DEFER: crea generation_log
       │   └─ Campos: task_id, model_name, user_id, project_id,
       │       scene_id, scene_code, take_number, request_payload,
       │       outputs=[{url, type}], status="succeeded"
       │
       └─ Devuelve { taskId, model, status: "succeeded", outputs }
```

**Registros creados:**
- `generation_logs` → 1 registro (status="succeeded" desde el primer momento)
- `server_communications` → 1 registro (POST a API externa)

No hay polling ni generated_assets porque la respuesta es inmediata.

---

## 3. Flujo para generación legacy (Selection-based)

### POST /studio/generate-legacy

```
POST /studio/generate-legacy (Selection, legacy)
  │
  └─ core.Service.Generate(sel)
       │
       ├─ Busca handler legacy (pickHandler)
       │
       ├─ handler.Generate() → llama a API externa
       │
       ├─ Trackea en memoria (s.tasks)
       │
       └─ Devuelve { taskId, modelId, model, status }
```

**NO se crean:** generation_logs, server_communications, generated_assets.

Solo trackea en memoria. Si el servidor se reinicia, se pierde el seguimiento.

---

## 4. Asset Sync logs

### POST /studio/sync-asset

```
POST /studio/sync-asset
  │
  └─ Service.SyncAsset()
       ├─ Crea model_assets (status="syncing")
       │
       ├─ AssetAPI.CreateAsset() → POST BytePlus SignedFetch
       │   └─ LOG: server_communication (vía AssetAPI.logComm)
       │       └─ endpoint=open.byteplusapi.com/..., method="POST"
       │
       └─ AssetAPI.GetAsset() → polling (hasta 20 intentos, cada 3s)
           └─ LOG: server_communication (cada consulta)
```

---

## 5. Diagrama de estado de generation_logs

```
        POST /generate
              │
              ▼
        ┌───────────┐
        │  failed   │ ←── estado inicial del defer (se sobrescribe si todo ok)
        └─────┬─────┘
              │
              ├─ ¿Error antes de generar taskId?
              │   └─ task_id = "<no-task>", status = "failed"
              │
              └─ ¿Generación exitosa?
                  │
                  ├─ Síncrono (Seedream, Gemini):
                  │   └─ status = "succeeded" (inmediato)
                  │
                  └─ Asíncrono (Seedance):
                      ├─ status = "running" (inicial)
                      │
                      └─ Polling hasta:
                          ├─ "succeeded" → UPDATE outputs, status
                          └─ "failed" → UPDATE error_message, status
```

## 6. Resumen de datos registrados por generación

### generation_logs (1 registro por generación)

| Dato | ¿De dónde viene? |
|------|------------------|
| `task_id` | Respuesta de la API externa o fallback |
| `model_name` | Nombre del modelo desde DB |
| `user_id` | Del request (`StudioGenerateRequest.UserID`) |
| `project_id` | Del request (obligatorio) |
| `scene_id` | Del request (obligatorio) |
| `scene_code` | Del request (obligatorio) |
| `take_number` | Del request (obligatorio) |
| `request_payload` | JSON del `StudioGenerateRequest` completo |
| `outputs` | URLs de salida (seteado al completar) |
| `status` | "running", "succeeded" o "failed" |
| `error_message` | Mensaje de error si falló |
| `created_at` | Auto |
| `updated_at` | Auto (se actualiza al completar) |

### server_communications (1+ registros por generación)

| Dato | ¿De dónde viene? |
|------|------------------|
| `task_id` | ID de tarea (si se obtuvo) |
| `model_name` | Nombre del modelo |
| `endpoint` | URL completa del endpoint llamado |
| `method` | POST (generate) o GET (status) |
| `request_body` | Payload JSON enviado a la API externa |
| `response_body` | Respuesta JSON de la API externa |
| `status_code` | Código HTTP de respuesta |
| `duration_ms` | Duración de la llamada |
| `error_message` | Error si ocurrió |

### generated_assets (N registros por generación, solo succeeded)

| Dato | ¿De dónde viene? |
|------|------------------|
| `task_id` | ID de la tarea completada |
| `model_name` | Del `GenerationLog` asociado |
| `user_id`, `project_id`, `scene_id`, `scene_code`, `take_number` | Del `GenerationLog` asociado |
| `original_url` | URL del output generado |
| `status` | "pending" (se confirma manualmente después) |
