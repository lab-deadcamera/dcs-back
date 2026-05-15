# Generators — Modelos generadores para Studio

## ¿Qué es un Generator?

Un **Generator** es un plugin que sabe cómo comunicarse con un modelo de IA específico (o familia de modelos) para generar contenido (video, imagen, audio). Cada Generator implementa la interfaz `Generator` y se registra en el `Service` de studio.

Los Generators **no** saben de HTTP ni de archivos — reciben todo ya resuelto: los file IDs vienen convertidos a data URLs, el modelo ya está resuelto con su API key, URL y endpoint.

## Interfaz Generator

```go
// internal/studio/generators/generator.go
type Generator interface {
    Name() string
    Match(modelName string) bool
    Generate(req *GeneratorRequest) (*GeneratorResult, error)
    GetStatus(taskID, apiKey, baseURL, endpoint string) (*GeneratorResult, error)
    CancelTask(taskID, apiKey, baseURL, endpoint string) error
}
```

| Método | Propósito |
|--------|-----------|
| `Name()` | Nombre legible del generador (ej: `"seedance"`) |
| `Match(modelName)` | Devuelve `true` si este generador debe manejar el modelo |
| `Generate(req)` | Envía la solicitud al modelo y devuelve el resultado |
| `GetStatus(...)` | Consulta el estado de una tarea asíncrona |
| `CancelTask(...)` | Cancela una tarea en ejecución |

## GeneratorRequest

Es el payload unificado que recibe **todo** Generator:

```go
type GeneratorRequest struct {
    Model         string        // nombre del modelo (ej: "dreamina-seedance-2-0-260128")
    Content       []ContentItem // prompts de texto y referencias a archivos
    Ratio         string        // relación de aspecto
    Duration      int           // duración en segundos
    CameraFixed   bool          // cámara fija
    Seed          string        // semilla
    Quality       string        // "standard" | "high"
    Quantity      int           // cantidad de outputs
    Watermark     bool          // incluir marca de agua
    Resolution    string        // "480p" | "720p" | "1080p" | "2K"
    GenerateAudio bool          // generar audio
    ImageMode     string        // modo de imagen
    // Credenciales (ya resueltas por el service layer)
    APIKey        string
    BaseURL       string
    Endpoint      string
}
```

Los items de `Content` ya vienen resueltos:

```go
type ContentItem struct {
    Type    string // "text" | "image" | "video" | "audio"
    Text    string // prompt textual o descripción
    Name    string // nombre original del archivo
    ID      string // UUID del archivo en file store (vació si es tipo text)
    DataURL string // data URL ya resuelta: "data:image/png;base64,..."
}
```

> El service layer resuelve los file IDs a data URLs **antes** de llamar al Generator. El Generator nunca necesita acceder al file store ni al disco.

## GeneratorResult

**Siempre** contiene `Outputs` como un slice, incluso si el modelo genera un solo recurso:

```go
type GeneratorResult struct {
    TaskID  string           // ID de la tarea (para polling)
    Model   string           // nombre del modelo
    Status  string           // "running" | "succeeded" | "failed" | "queued"
    Outputs []OutputResource // recursos generados
    Raw     interface{}      // respuesta cruda de la API
    Error   string           // mensaje de error
}

type OutputResource struct {
    URL      string // URL pública del recurso
    LocalURL string // URL local (si se descargó)
    Type     string // "video" | "image" | "audio"
}
```

## Cómo crear un nuevo Generator

### 1. Crear el archivo

`internal/studio/generators/<nombre>.go`:

```go
package generators

import (
    "fmt"
    "strings"
)

type MiModeloGenerator struct {
    httpClient *http.Client
    outputsDir string
}

func NewMiModeloGenerator(outputsDir string) *MiModeloGenerator {
    return &MiModeloGenerator{
        httpClient: &http.Client{Timeout: 60 * time.Second},
        outputsDir: outputsDir,
    }
}

func (g *MiModeloGenerator) Name() string { return "mi-modelo" }

func (g *MiModeloGenerator) Match(modelName string) bool {
    return strings.Contains(strings.ToLower(modelName), "mi-modelo")
}
```

### 2. Implementar Generate

```go
func (g *MiModeloGenerator) Generate(req *GeneratorRequest) (*GeneratorResult, error) {
    // 1. Construir el payload específico del modelo
    payload := g.buildPayload(req)

    // 2. Llamar a la API externa
    resp, err := g.callAPI(req.BaseURL+req.Endpoint, payload, req.APIKey)
    if err != nil {
        return nil, err
    }

    // 3. Si es síncrono, devolver outputs directo
    return &GeneratorResult{
        TaskID:  resp.TaskID,
        Model:   req.Model,
        Status:  "succeeded",
        Outputs: []OutputResource{{URL: resp.URL, Type: "image"}},
    }, nil
}
```

### 3. Implementar GetStatus (para modelos asíncronos)

```go
func (g *MiModeloGenerator) GetStatus(taskID, apiKey, baseURL, endpoint string) (*GeneratorResult, error) {
    resp, err := g.callAPI(baseURL+endpoint+"/tasks/"+taskID, nil, apiKey)
    if err != nil {
        return nil, err
    }

    if resp.Status == "succeeded" {
        // Descargar el recurso si aplica
        localURL := g.downloadOutput(resp.OutputURL)
        return &GeneratorResult{
            TaskID:  taskID,
            Model:   "mi-modelo",
            Status:  "succeeded",
            Outputs: []OutputResource{{
                URL:      resp.OutputURL,
                LocalURL: localURL,
                Type:     "video",
            }},
        }, nil
    }

    return &GeneratorResult{
        TaskID: taskID,
        Model:  "mi-modelo",
        Status: resp.Status,
    }, nil
}
```

### 4. Registrar en main.go

```go
import "dcs-back-v0/internal/studio/generators"

func main() {
    // ...
    studioSvc := studio.NewService(providerStore, fileSvc, cfg.OutputsDir)
    studioSvc.RegisterGenerator(generators.NewMiModeloGenerator(cfg.OutputsDir))
    // ...
}
```

## Cómo se integran con los modelos y API keys

### Flujo completo

```
POST /api/v1/studio/generate
  │
  ├── Handler.GenerateUnified
  │     └── bindea el JSON → StudioGenerateRequest
  │
  ├── Service.GenerateUnified
  │     ├── 1. Busca el modelo por nombre en provider store
  │     │     → providerStore.GetModelByName("dreamina-seedance-2-0-260128")
  │     │     → devuelve Model{ APIKey, URL, Endpoint, ... }
  │     │
  │     ├── 2. Resuelve file IDs a data URLs
  │     │     → fileService.GetFile(id) → lee el archivo del disco → base64
  │     │
  │     ├── 3. Elige el Generator
  │     │     → recorre generatorsList, llama a Match(modelName)
  │     │     → el primero que devuelve true es el elegido
  │     │
  │     └── 4. Construye GeneratorRequest y llama a gen.Generate(req)
  │           → el Generator usa APIKey, BaseURL, Endpoint del request
  │
  └── Generator.Generate(req)
        → usa req.APIKey para autenticar
        → usa req.BaseURL + req.Endpoint para la URL
        → construye el payload específico del modelo
        → llama a la API externa
        → devuelve GeneratorResult
```

### Resolución de modelos

El campo `model` del payload se usa para:

1. **Buscar credenciales** — `providerStore.GetModelByName(name)` devuelve el `Model` registrado con su `APIKey`, `URL` y `Endpoint`
2. **Elegir Generator** — se itera `generatorsList` llamando a `Match(modelName)` hasta encontrar un match

Esto significa que puedes tener múltiples modelos (ej: `seedance-2-0`, `seedance-2-0-fast`, `seedance-2-0-fast-260128`) todos manejados por el mismo Generator mientras tengan "seedance" en el nombre.

### Ejemplo de mapeo

| Modelo registrado | API Key | Generator que matchea |
|---|---|---|
| `dreamina-seedance-2-0-260128` | `ark-xxx...` | Seedance (contiene "seedance") |
| `dreamina-seedance-2-0-fast-260128` | `ark-yyy...` | Seedance (contiene "seedance") |
| `seedream-4-0-250828` | `ark-zzz...` | Seedream (contiene "seedream") |
| `mi-modelo-personalizado` | `sk-...` | MiModelo (contiene "mi-modelo") |

## Reglas importantes

1. **Los outputs siempre son array** — incluso si el modelo genera un solo recurso, usa `[]OutputResource{...}`
2. **No accedas al file store ni al disco** — los file IDs ya vienen resueltos como data URLs en `ContentItem.DataURL`
3. **No guardes API keys en el Generator** — las recibe en `GeneratorRequest` en cada llamada
4. **Usa `outputsDir` para descargar archivos** — si el modelo devuelve URLs removibles, descárgalas a `outputsDir` y completa el `LocalURL`
5. **Sé específico con Match** — usa `strings.Contains` con el nombre del modelo en minúsculas para evitar falsos positivos

## Diferencia entre Name() y Match()

- `Name()` — etiqueta interna del generador. No se usa para routing ni matching. Solo para logging y debug.
- `Match(modelName)` — es el **único** método que decide si este generador maneja o no un modelo. Puedes usar la lógica que quieras: `strings.Contains`, regex, switch de valores exactos, etc.

Un mismo `Name()` puede matchear múltiples modelos distintos. El nombre del generador **no tiene que coincidir** con el nombre del modelo.

### Buen ejemplo

```go
func (g *SeedanceGenerator) Name() string { return "seedance" }

// Matchea cualquier modelo que contenga "seedance" o "dreamina"
func (g *SeedanceGenerator) Match(modelName string) bool {
    lower := strings.ToLower(modelName)
    return strings.Contains(lower, "seedance") || strings.Contains(lower, "dreamina")
}
```

`Name()` es `"seedance"` pero matchea modelos que se llaman `dreamina-seedance-2-0-260128`, `seedance-2-0`, etc.
