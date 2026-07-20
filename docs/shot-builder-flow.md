# Shot Builder — Flujo detallado

## 1. Creación de proyecto, episodio y escena

**Actor:** Director

- El director crea un proyecto con capítulos (episodios) y escenas mediante la interfaz de proyectos (`/projects`).
- A cada escena se le pueden asignar **personajes** mediante el sistema de assignments.
- No requiere código del shot builder.

---

## 2. Asignación de personajes a la escena

**Actor:** Director

- El director asigna personajes existentes a una escena específica.
- También puede asignar **assets** (archivos de imagen, video, audio) y **presets de cinematografía** a la escena.
- Los datos se persisten en el backend via los endpoints de assignments del proyecto.

---

## 3. Ingreso al Studio — Selección de escena sin shot

**Actor:** Usuario

- El usuario navega al **Studio** (`/studio`).
- Selecciona **Proyecto → Episodio → Escena** en el breadcrumb.
- Si la escena **no tiene shots**, el layout principal muestra `<app-shot-builder-panel>` en lugar de los componentes de edición de shot.

**Código relevante:**
```html
<!-- index-studio.html -->
@if (navSelectedSceneId() && !navSelectedShotId()) {
  <app-shot-builder-panel ... />
} @else {
  <app-character-assets />
  <app-proncer />
  <app-output-format />
}
```

### Assets de escena cargados automáticamente

Al seleccionar la escena, `handleSceneSelected()` ejecuta:

```
GET /projects/{projectId}/chapters/{chapterId}/scenes/{sceneId}/assignments
```

La respuesta contiene:
- **Presets** — IDs de presets de cinematografía asignados a la escena
- **Characters** — personajes asignados con `character_id` y `name`
- **Assets** — archivos libres con `file_id`, `filename`, `mime_type`

`setSceneAssignments()` procesa la respuesta y puebla el `StudioStore` con:

| Store signal | Contenido |
|-------------|-----------|
| `sceneCharacterIds` | Set de character_ids |
| `sceneCharacterData` | Array de `{ id, name }` de personajes asignados |
| `sceneAssetIds` | Set de file_ids de assets |
| `freeAssets` | Array de `ReferenceAsset` con `id`, `kind`, `filename` (para mostrar thumbnails) |
| `scenePresetIds` | Set de preset_ids |
| `assignmentsLoaded` | `true` |

### Preview visual de assets en el shot builder

Dentro del panel izquierdo del shot builder, entre el textarea y los botones de acción, hay una sección colapsable **"Scene Assets"** que muestra:

- **Characters** — chips con nombres de personajes asignados (icono `pi pi-user` + nombre)
- **Free Assets** — thumbnails en grid de 48×48px:
  - Imágenes: renderizadas con `sourceAsset` pipe → `{API_URL}/files/{fileId}/serve`
  - Videos: icono `pi-video`
  - Audio: icono `pi-volume-up`
  - Fallback a icono si la imagen no carga (broken thumb tracking)

La sección solo aparece cuando `assignmentsLoaded()` es `true`.

---

## 4. Uso del Shot Builder — Prompt y archivos

**Actor:** Usuario

El Shot Builder Panel tiene dos paneles:

### Panel izquierdo — Chat + Input

- **Chat history:** muestra el historial de interacción con Claude
- **Scene Assets (colapsable):** preview de personajes y assets asignados a la escena
- **Textarea:** el usuario escribe una descripción de la escena, un guion o instrucciones para los shots
- **File upload:** se pueden adjuntar archivos (PDF, imágenes, documentos Office, texto) como referencia para Claude
- **Settings gear:** permite seleccionar el modelo Claude y si se generan prompts en chino (`generateZh`)
- **Botón "Generate Shots":** envía el prompt + archivos + contexto de escena al backend

### Panel derecho — Preview

- Muestra tabs: **Shots** (vista del resultado), y tabs por cada archivo subido
- Cuando se recibe un resultado de Claude, se muestra el **Sequence Viewer nativo** (grid de shots) o un **artifact HTML**

### Scene Context en la generación

Cuando el usuario hace clic en "Generate Shots", el método `send()`:

1. Construye el contenido del prompt (texto + archivos)
2. **Recolecta scene context** del StudioStore usando SOLO datos disponibles a nivel de escena (no dependientes de un shot):

```typescript
const sceneContext: SceneContext = {
  description: this.studio.rawDescription() || undefined,
  characters: this.studio.sceneCharacterData().map(c => ({ name: c.name })),
  assets: this.studio.freeAssets().map(a => ({
    id: a.id,
    filename: a.filename,
    mimeType: a.kind === 'image' ? 'image/png' : a.kind === 'video' ? 'video/mp4' : 'audio/mpeg',
  })),
};
```

3. Llama a `shotBuilderService.generate()` con todo el contexto:

```typescript
this.shotBuilderService.generate({
  projectId, sceneId, prompt: content,
  systemPrompt: '', model, skillID, userName, generateZh,
  sceneContext,   // ← personajes + assets de la escena
});
```

**Nota:** Los presets de cinematografía (`shotPresets`) NO se incluyen en el scene context del shot builder porque solo están disponibles cuando hay un shot seleccionado. A nivel de escena solo tenemos los IDs de presets (`scenePresetIds`), no los datos completos.

Esto permite a Claude generar shots más relevantes teniendo conocimiento de los personajes y recursos de la escena.

---

## 5. Recepción del grid de shots — Modificación y selección de idioma

**Actor:** Usuario

Claude responde con un `ShotBuilderResult` que contiene:

- `shots: ShotBuilderShot[]` — lista simple de `{ number, name, description }`
- `rawText: string` — respuesta cruda del modelo
- `sequence?: Sequence` — datos enriquecidos del Sequence Viewer

Cuando se recibe la respuesta:

1. `this.rawResponse.set(result.rawText)` — guarda la respuesta cruda
2. `this.shots.set(result.shots)` — guarda la lista parseada de shots
3. `this.sequenceData.set(result.sequence)` — si hay datos Sequence, se guardan para el visor nativo

### Sequence Viewer

El componente `<app-shot-sequence-viewer>` renderiza:

- **Header:** nombre del proyecto, episodio, escena, duración total, modo
- **Timeline strip:** barra de tiempo interactiva con segmentos por shot
- **Shot cards:** cada shot muestra:
  - ID (A, B, C...)
  - Título y descripción
  - Cámara, composición, blocking, acting
  - **Prompt editable** en EN y ZH (selector de idioma por shot)
  - Botón de aprobación individual

El usuario puede:

- **Editar** el prompt de cada shot (EN o ZH)
- **Seleccionar el idioma** (EN/ZH) por cada shot mediante un toggle
- **Aprobar/rechazar** shots individualmente
- Modificar la descripción del shot

---

## 6. Click en "Crear listado de pre-prompts"

**Actor:** Usuario

Cuando el usuario hace clic en el botón **"Crear listado de pre-prompts"** dentro del Sequence Viewer:

1. El Sequence Viewer emite `createPrePromptsClicked` con un array:

```typescript
[
  { shotId: 'A', lang: 'en', prompt: 'Scene & Mood: A violent intrusion...' },
  { shotId: 'B', lang: 'zh', prompt: '第一次交涉。画面布局...' },
  // ...
]
```

2. El `ShotBuilderPanelComponent.onCreatePrePrompts()` procesa cada shot:

   - Toma `shotId` para identificar el shot (solo referencia visual)
   - Toma `lang` para saber qué prompt usar (EN o ZH)
   - Toma `prompt` (el texto completo del prompt en el idioma seleccionado)
   - Combina con `this.shots()` para obtener `number` y `name`
   - Crea cada shot secuencialmente via `POST /projects/.../shots`

3. **Aplica el output format del Sequence** al StudioStore:

```typescript
const seq = this.sequenceData();
if (seq) {
  if (seq.aspectRatio) {
    this.studio.patchOutput({ aspectRatio: seq.aspectRatio });
  }
  if (seq.duration && seq.duration > 0) {
    this.studio.patchOutput({ durationSeconds: Math.min(seq.duration, 15) });
  }
}
```

Esto asegura que el output format (aspect ratio, duración) herede los datos generados por el shot builder (Claude), dando prioridad sobre la configuración por defecto del usuario.

---

## 7. Creación de shots en backend

**Backend**

Se ejecuta `POST /projects/{projectId}/chapters/{chapterId}/scenes/{sceneId}/shots` por cada shot, con:

```json
{
  "number": 1,
  "name": "Mike irrumpe + estalla",
  "description": "Scene & Mood: A violent intrusion..."
}
```

Donde `description` es el **pre-prompt** del shot en el idioma seleccionado por el usuario.

Los shots se crean **secuencialmente** (uno tras otro) para mantener el orden.

---

## 8. Breadcrumb se actualiza — Navegación al primer shot

**Frontend**

Cuando todos los shots se han creado exitosamente:

1. `onCreatePrePrompts()` emite `shotsSaved` con el ID del primer shot y su descripción
2. `IndexStudio.onShotsSaved()` responde:

```typescript
onShotsSaved(event) {
  this.reloadShots();                           // Recarga lista de shots en breadcrumb
  this.navSelectedShotId.set(event.firstShotId); // Selecciona el primer shot
  this.startSessionWithShot({                   // Inicia sesión con el shot
    id: event.firstShotId,
    number: 1,
    name: event.firstShotDescription.slice(0, 40)
  });
  if (event.firstShotDescription) {
    this.studio.setRawDescription(event.firstShotDescription); // Carga pre-prompt
  }
}
```

3. El breadcrumb se actualiza mostrando los nuevos shots (SH01, SH02...)
4. El layout cambia: el **shot builder se oculta** y se muestra la **interfaz de edición de shot** con:
   - CharacterAssets
   - OutputFormat
   - Viewer
   - PromptBuilder
   - TakesReel

---

## 9. Sesión del shot iniciada + Pre-prompt en PromptBuilder

Cuando se navega al primer shot:

1. `startSessionWithShot()`:
   - Carga takes del backend
   - Inicializa el `StudioStore` con `initStudioSession()` (setea projectId, chapterId, sceneId, shotId)
   - Carga assignments de la escena (`setSceneAssignments()`)
   - Carga shot resources (personajes, assets, presets)

2. `setRawDescription(firstShotDescription)` establece el **pre-prompt** en el store

3. El `PromptBuilderComponent` detecta el cambio en `studio.rawDescription()` mediante un `effect()` y actualiza el editor Quill con el texto del pre-prompt

4. **Efecto visual:** el usuario ve el prompt completo cargado en el editor, listo para ser editado o enviado a generar.

---

## 10. Persistencia del output format en backend

### 10.1 Columnas en la tabla shots

Se agregaron dos columnas a la tabla shots (migration 00018_shot_output_format.sql):

| Columna | Tipo | Proposito |
|---------|------|-----------|
| aspect_ratio | VARCHAR(10) DEFAULT NULL | Relacion de aspecto (ej. 9:16, 16:9) |
| duration_seconds | INT DEFAULT NULL | Duracion en segundos |

### 10.2 Persistencia al crear shots

Cuando onCreatePrePrompts() o saveShotsToBackend() terminan de crear los shots:

1. Se aplica patchOutput() al StudioStore (en memoria) con los valores del Sequence
2. Se persiste el formato a cada shot creado via PATCH /projects/{id}/chapters/{id}/scenes/{id}/shots/{shotId}

### 10.3 Restauracion al cargar un shot

Cuando startSessionWithShot() carga un shot via getShot(), se restaura el output format desde
shot.aspect_ratio y shot.duration_seconds, aplicandolos via studio.patchOutput().

### 10.4 Prioridades del output format

| Campo | Fuente | Default | Prioridad |
|-------|--------|---------|-----------|
| aspectRatio | Sequence.aspectRatio o shot.aspect_ratio | 9:16 | Shot builder |
| resolution | No viene del Sequence | 720p | Default |
| durationSeconds | Sequence.duration o shot.duration_seconds (capped a 15s) | 5 | Shot builder |
| sound | No viene del Sequence | true | Default |
| engine | No viene del Sequence | fast | Default |

El OutputFormatComponent refleja estos cambios automaticamente porque esta vinculado al studio.output() signal.

---
## 11. Assets de escena en PromptBuilder

Cuando se selecciona el shot:

- `setSceneAssignments()` carga los personajes, presets y assets de la escena (llamado desde `handleSceneSelected()` y también desde `startSessionWithShot()`)
- `loadShotResources()` carga recursos específicos del shot (personajes del shot, assets del shot, presets del shot)
- El `CharacterAssetsComponent` muestra los personajes asignados en "My Library"
- El `PromptBuilderComponent` muestra los assets usados como **chips** agrupados por tipo (imagen, video, audio):
  - `[Image1]`, `[Image2]`
  - `[Video1]`
  - `[Audio1]`

Estos chips permiten al usuario ver y gestionar qué referencias visuales se incluirán en la generación.

---

## Resumen arquitectura de componentes

```
IndexStudio
  ├── StudioBreadcrumbComponent        (navegación proyecto/episodio/escena/shot)
  │
  ├── [sceneId && !shotId]
  │   └── ShotBuilderPanelComponent    (shot builder completo)
  │       ├── Splitter izquierdo
  │       │   ├── Chat history
  │       │   ├── Scene Assets (colapsable)  ← personajes + free assets
  │       │   ├── Textarea + File upload
  │       │   └── Botones: Mock Seq (SA), Preview (SA), Clear, Generate Shots
  │       ├── Splitter derecho
  │       │   ├── ShotSequenceViewerComponent  (grid nativo)
  │       │   │   ├── ShotTimelineStripComponent
  │       │   │   └── ShotCardPreviewComponent
  │       │   └── Artifact HTML (iframe)
  │       └── ShotBuilderSettingsDialogComponent
  │
  ├── [shotId]
  │   ├── CharacterAssetsComponent     (personajes, assets)
  │   ├── ProncerComponent             (optimizador de prompt)
  │   ├── OutputFormatComponent        (aspect ratio, resolución, duración)
  │   ├── ViewerComponent              (preview de video)
  │   ├── TakeChecklistComponent       (checklist de takes)
  │   ├── RatingComponent              (rating de takes)
  │   ├── PromptBuilderComponent       (editor de prompt + generate)
  │   └── TakesReelComponent           (historial de takes/generaciones)
  │
  └── Dialogs
      ├── Preview payload dialog       (solo SUPER_ADMIN)
      └── Sync dialog
```

## Servicios involucrados

| Servicio | Rol |
|----------|-----|
| `ShotBuilderService` | Comunicación con backend Claude (generate-shots, optimize-prompt) |
| `ShotBuilderArtifact` | Parseo y renderizado de respuestas de Claude (artifact HTML) |
| `StudioStore` | Estado global del estudio (output format, rawDescription, assets, takes) |
| `SessionStore` | Sesión del usuario (roleLevel para permisos, user info) |
| `ProjectsApiService` | CRUD de proyectos, capítulos, escenas, shots |
| `VideoGeneratorService` | Generación de video (Seedance) |
| `PresetsService` | Presets de cinematografía |
| `StudioApiService` | HTTP requests del Studio (assignments, resources, updateShotFormat) |
| `StudioApiService` | HTTP requests del Studio (assignments, resources, updateShotFormat) |

## Endpoints relevantes

| Endpoint | Método | Propósito |
|----------|--------|-----------|
| `/projects/{id}/chapters/{chId}/scenes/{scId}/assignments` | GET | Obtener personajes, presets y assets asignados a la escena |
| `/projects/{id}/chapters/{chId}/scenes/{scId}/shots` | POST | Crear un shot en la escena |
| `/projects/{id}/chapters/{chId}/scenes/{scId}/shots/{shId}` | PATCH | Actualizar shot (incluyendo aspect_ratio y duration_seconds) |
| `/projects/{id}/chapters/{chId}/scenes/{scId}/shots` | GET | Listar shots de la escena |
| `/projects/{id}/chapters/{chId}/scenes/{scId}/shots/{shId}/resources` | GET | Obtener recursos del shot (personajes, assets, presets) |
| `/studio/text/claude/generate-shots` | POST | Enviar prompt + scene context a Claude para generar shots |
| `/projects/{id}/chapters/{chId}/scenes/{scId}/shots/{shId}` | PATCH | Actualizar shot (incluyendo aspect_ratio y duration_seconds) |
| `StudioApiService` | HTTP requests del Studio (assignments, resources, updateShotFormat) |
