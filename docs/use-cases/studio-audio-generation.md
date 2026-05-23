# Generación de Audio vía Studio

**Estado: NO IMPLEMENTADO**

Los endpoints están definidos pero devuelven `"audio generation not yet implemented"`.

## Endpoints

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/v1/studio/audio/generate` | Generar audio |
| GET | `/api/v1/studio/audio/status/:taskId` | Consultar estado |
| DELETE | `/api/v1/studio/audio/task/:taskId` | Cancelar tarea |
| POST | `/api/v1/studio/audio/preview` | Previsualizar payload |

## Contexto de sesión

Los campos `project_id`, `scene_id`, `scene_code` y `take_number` son obligatorios (definidos en el tipo `GenerateRequest`).

## Interfaz para implementar

Para agregar un generador de audio, implementar `AudioGenerator` en `internal/studio/audio/generators/`:

```go
type AudioGenerator interface {
    Name() string
    Match(modelName string) bool
    Validate(req *studio.GeneratorRequest) error
    Generate(req *studio.GeneratorRequest) (*studio.GeneratorResult, error)
    GetStatus(taskID, apiKey, baseURL, endpoint string) (*studio.GeneratorResult, error)
    CancelTask(taskID, apiKey, baseURL, endpoint string) error
    BuildPayload(req *studio.GeneratorRequest) map[string]interface{}
    ContentType() string
}
```

Ver [`internal/studio/audio/domain.go`](../../internal/studio/audio/domain.go).

## Payload esperado

```json
{
  "model": "nombre-del-modelo-de-audio",
  "content": [
    { "type": "text", "text": "descripción del audio a generar" }
  ],
  "duration": 30,
  "project_id": "uuid-del-proyecto",
  "scene_id": "uuid-de-la-escena",
  "scene_code": "ESC-001",
  "take_number": 1,
  "user_id": 1
}
```
