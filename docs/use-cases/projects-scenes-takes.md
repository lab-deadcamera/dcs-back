# Proyectos, Escenas y Tomas

Organiza la generación de videos en **Proyectos** que contienen **Escenas**, y cada escena tiene **Tomas** numeradas (1–100) con su video asociado.

## Endpoints

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/v1/projects` | Crear un proyecto |
| GET | `/api/v1/projects` | Listar proyectos |
| GET | `/api/v1/projects/:id` | Ver proyecto con sus escenas |
| PATCH | `/api/v1/projects/:id` | Actualizar proyecto |
| DELETE | `/api/v1/projects/:id` | Eliminar proyecto (soft delete) |
| POST | `/api/v1/projects/:id/scenes` | Crear escena en un proyecto |
| GET | `/api/v1/projects/:id/scenes` | Listar escenas de un proyecto |
| GET | `/api/v1/projects/:id/scenes/:sceneId` | Ver escena con sus tomas |
| PATCH | `/api/v1/projects/:id/scenes/:sceneId` | Actualizar escena |
| DELETE | `/api/v1/projects/:id/scenes/:sceneId` | Eliminar escena (soft delete) |
| POST | `/api/v1/projects/:id/scenes/:sceneId/takes` | Crear toma en una escena |
| GET | `/api/v1/projects/:id/scenes/:sceneId/takes` | Listar tomas de una escena |
| GET | `/api/v1/projects/:id/scenes/:sceneId/takes/:takeId` | Ver detalle de una toma |
| PATCH | `/api/v1/projects/:id/scenes/:sceneId/takes/:takeId` | Actualizar toma (URL, estado) |
| DELETE | `/api/v1/projects/:id/scenes/:sceneId/takes/:takeId` | Eliminar toma (soft delete) |

## Ejemplo: crear proyecto con escenas y tomas

```javascript
// 1. Crear proyecto
const project = await fetch('/api/v1/projects', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    name: 'Comercial Fruta Fresh',
    description: 'Anuncio de jugo natural'
  })
}).then(r => r.json());
// project.data → { id, name, description, ... }

const projectId = project.data.id;

// 2. Crear escenas
const scene1 = await fetch(`/api/v1/projects/${projectId}/scenes`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ number: 1, name: 'Apertura', description: 'Toma inicial de la fruta' })
}).then(r => r.json());

const scene2 = await fetch(`/api/v1/projects/${projectId}/scenes`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ number: 2, name: 'Preparación', description: 'Cortando la fruta' })
}).then(r => r.json());

const sceneId = scene1.data.id;

// 3. Crear tomas en la escena
const take1 = await fetch(`/api/v1/projects/${projectId}/scenes/${sceneId}/takes`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ number: 1 })
}).then(r => r.json());

const take2 = await fetch(`/api/v1/projects/${projectId}/scenes/${sceneId}/takes`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ number: 2 })
}).then(r => r.json());

// 4. Actualizar toma con el video generado
await fetch(`/api/v1/projects/${projectId}/scenes/${sceneId}/takes/${take1.data.id}`, {
  method: 'PATCH',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    video_url: 'https://cdn.example.com/videos/take1.mp4',
    video_local_url: '/outputs/take1.mp4',
    status: 'completed'
  })
});

// 5. Ver proyecto completo con escenas
const fullProject = await fetch(`/api/v1/projects/${projectId}`).then(r => r.json());
// fullProject.data → { project: {...}, scenes: [...] }

// 6. Ver escena con sus tomas
const fullScene = await fetch(`/api/v1/projects/${projectId}/scenes/${sceneId}`).then(r => r.json());
// fullScene.data → { scene: {...}, takes: [...] }
```

## Estructura de datos

```
Proyecto
 ├── Escena 1: "Apertura"
 │    ├── Toma 1 → video_url, status
 │    ├── Toma 2 → video_url, status
 │    └── ...
 ├── Escena 2: "Preparación"
 │    ├── Toma 1 → video_url, status
 │    └── ...
 └── ...
```

### Project

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `id` | UUID | ID del proyecto |
| `name` | string | Nombre del proyecto |
| `description` | string | Descripción |
| `metadata` | TEXT (JSON) | Metadatos adicionales |
| `created_at` | datetime | Fecha de creación |
| `updated_at` | datetime | Última actualización |

### Scene

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `id` | UUID | ID de la escena |
| `project_id` | UUID | Proyecto al que pertenece |
| `number` | int | Número de escena (único por proyecto) |
| `name` | string | Nombre de la escena |
| `description` | string | Descripción |
| `created_at` | datetime | Fecha de creación |
| `updated_at` | datetime | Última actualización |

### Take

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `id` | UUID | ID de la toma |
| `scene_id` | UUID | Escena a la que pertenece |
| `number` | int | Número de toma (1–100, único por escena) |
| `video_url` | string | URL pública del video generado |
| `video_local_url` | string | URL local del video (si aplica) |
| `status` | string | Estado: `pending`, `processing`, `completed`, `failed` |
| `created_at` | datetime | Fecha de creación |
| `updated_at` | datetime | Última actualización |

## Consideraciones

- El número de escena debe ser único dentro de cada proyecto.
- El número de toma debe ser único dentro de cada escena (rango 1–100).
- Los proyectos, escenas y tomas usan soft delete — se marcan como eliminados sin borrar datos.
- Al eliminar un proyecto, todas sus escenas y tomas se eliminan en cascada.
