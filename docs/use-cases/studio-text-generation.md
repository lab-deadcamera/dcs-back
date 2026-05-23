# Generación de Texto vía Studio

**Estado: NO IMPLEMENTADO**

Los endpoints están definidos pero devuelven `"text generation not yet implemented"`.

## Endpoints

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/v1/studio/text/generate` | Generar texto |
| GET | `/api/v1/studio/text/status/:taskId` | Consultar estado |
| DELETE | `/api/v1/studio/text/task/:taskId` | Cancelar tarea |
| POST | `/api/v1/studio/text/preview` | Previsualizar payload |

## Contexto de sesión

Los campos `project_id`, `scene_id`, `scene_code` y `take_number` son obligatorios (definidos en el tipo `GenerateRequest`).

## Interfaz para implementar

Para agregar un generador de texto, implementar `TextGenerator` en `internal/studio/text/generators/`:

```go
type TextGenerator interface {
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

Ver [`internal/studio/text/domain.go`](../../internal/studio/text/domain.go).

## Payload esperado

```json
{
  "model": "nombre-del-modelo-de-texto",
  "content": [
    { "type": "text", "text": "genera un guión para una escena de..." }
  ],
  "seed": "42",
  "quality": "standard",
  "quantity": 1,
  "project_id": "uuid-del-proyecto",
  "scene_id": "uuid-de-la-escena",
  "scene_code": "ESC-001",
  "take_number": 1,
  "user_id": 1
}
```
