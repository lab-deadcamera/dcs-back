# Shot Builder — Contexto del Backend

> Documento de contexto para trabajar sobre el Shot Builder sin romper lo existente.
> Backend: `dcs-back` (Go + Gin). Última actualización: 2026-08-09.
>
> Complementa a [`docs/shot-builder-flow.md`](./shot-builder-flow.md), que documenta el **flujo de UI** en el frontend. Este documento se enfoca en el **backend** (contratos, lógica de Claude, logs, datos).

---

## 1. ¿Qué es el Shot Builder?

El Shot Builder es el corazón de la plataforma: convierte un **guion / descripción de escena** en un **plan de rodaje** (episodio → escenas → shots), cada shot con su **pre-prompt listo para Seedance** (EN y opcionalmente ZH).

**Flujo de negocio:**

1. El usuario escribe una descripción/guion de escena (o sube archivos) en el panel del Shot Builder.
2. El frontend adjunta el **scene context** (personajes, presets de cinematografía, assets de referencia asignados a la escena).
3. El backend llama a **Claude** (Anthropic) con un system prompt especializado (**DCS-DIRECTION**), que devuelve un JSON estructurado `{ episode, scenes[] }`.
4. El frontend muestra el resultado como **Sequence Viewer** y, al hacer clic en "Crear listado de pre-prompts", crea los **shots** en la base de datos (uno por shot del JSON) y navega al primer shot.

El Shot Builder y el **Proncer** (optimizador de prompts) comparten el mismo módulo backend y el mismo `SceneContext`.

---

## 2. Dónde vive el código (backend)

```
dcs-back/
├── main.go                                    → Wiring: stores, servicios, registro de módulos
└── internal/modules/
    ├── skill/                                 → Módulo "skills" (system prompts nombrados para el Shot Builder)
    │   ├── types.go, service.go, store.go, handler.go, module.go
    ├── studio/
    │   ├── module.go                          → Registro de rutas de /studio (incluye /text/claude/*)
    │   └── text/                              → Módulo de texto: Shot Builder + Proncer (Claude)
    │       ├── handler.go                     → Handlers ClaudeGenerateShots / ClaudeOptimizePrompt + logs de fallos
    │       ├── claude_types.go                → Tipos: SceneContext, request/response, Shot/SceneData/Episode
    │       ├── log_store.go                   → Store de shot_builder_logs / shot_builder_attempts
    │       ├── types.go, domain.go            → Stub genérico de generación de texto (NO usado por el Shot Builder)
    │       ├── handler_test.go                → Tests del prompt, extracción y validación JSON
    │       └── generators/claude_text.go      → Generador TextGenerator genérico (NO usado por el Shot Builder)
    ├── provider/                              → Proveedores/modelos IA: resuelve la API key de Claude
    └── project/                               → CRUD de proyectos/capítulos/escenas/shots/takes
```

> **Importante:** El Shot Builder **NO** pasa por el pipeline unificado de generación (`PipelineRunner` / `GeneratorRequest`). Es un flujo aparte y síncrono: `Handler.ClaudeGenerateShots()` → `callClaude()` → respuesta JSON. El generador `text/generators/claude_text.go` y los stubs de `POST /studio/text/generate` son otra cosa (generación de texto genérica, marcada como "not yet implemented").

---

## 3. Flujo end-to-end

```
Angular (ShotBuilderPanel)
  │  POST /api/v1/studio/text/claude/generate-shots
  │  { project_id, scene_id, prompt, skill_id, model, api_model, generate_zh, scene_context }
  ▼
text.Handler.ClaudeGenerateShots (handler.go)
  │  1. Lee raw body (ground truth para logs de fallo) y hace bind del JSON
  │  2. Si hay scene_context → buildSceneContextBlock() se antepone al prompt
  │  3. Compone system prompt: defaultShotBuilderPrompt + skill + user instructions + reglas (ZH / idioma)
  │  4. Retry loop (hasta 3 intentos):
  │       ├─ callClaude() → Anthropic Messages API (claude-sonnet-4-6 por defecto)
  │       ├─ extractJSON(reply) → objeto JSON balanceado
  │       ├─ validateShotJSON(clean) → formato nuevo (episode+scenes) o legacy (shots[])
  │       └─ si inválido → buildCorrectivePrompt(original) y reintenta
  │  5. Éxito → { taskId, model, status: "succeeded", text: <JSON limpio> }  (NO se loguea)
  │     Fracaso → persistFailure() → shot_builder_logs + shot_builder_attempts  (500)
  ▼
Angular: parsea el JSON, renderiza Sequence Viewer
  │  POST /projects/{id}/chapters/{chId}/scenes/{scId}/shots   (por cada shot, secuencial)
  │  PATCH .../shots/{shotId}  → persiste aspect_ratio y duration_seconds del Sequence
  ▼
DB: shots + output format → sesión del primer shot
```

---

## 4. Contrato API

### 4.1 Generar shots

`POST /api/v1/studio/text/claude/generate-shots` (auth JWT)

**Request** (`ClaudeGenerateShotsRequest` en `claude_types.go`):

```json
{
  "scene_id": "uuid",            // obligatorio
  "project_id": "uuid",          // obligatorio
  "prompt": "texto del guion",   // obligatorio
  "model": "claude-shot-builder",// key del modelo (clave en provider store)
  "api_model": "claude-sonnet-4-6", // id real de Anthropic (opcional; default = model)
  "system_prompt": "",           // instrucciones extra del usuario
  "skill_id": "uuid",            // skill nombrado (opcional)
  "user_id": 0, "user_name": "", // denormalizados; se sobreescriben desde el JWT si existe
  "generate_zh": false,          // si true → Claude incluye prompt.zh
  "scene_context": {             // opcional — personajes, presets, assets de la escena
    "description": "...",
    "characters": [{ "id", "name", "description", "slot" }],
    "presets": [{ "code", "label", "prompt" }],
    "assets": [{ "id", "filename", "mime_type" }]
  }
}
```

**Response éxito** (200):

```json
{
  "success": true,
  "data": {
    "taskId": "claude_<timestamp_ms>",
    "model": "claude-sonnet-4-6",
    "status": "succeeded",
    "text": "{ \"episode\": {...}, \"scenes\": [...] }"
  }
}
```

> `text` es el JSON **crudo** extraído de la respuesta de Claude. El frontend lo parsea y lo enriquece en el Sequence Viewer. Los campos `episode`, `scenes`, `directorNotes`, `aspectRatio`, `mode` existen en el tipo de respuesta pero **no se pueblan** — solo se envía `text`.

**Errores:**
- 400: prompt vacío / body inválido.
- 500: error de API de Claude, o agotados los 3 intentos con JSON inválido (en ambos casos se persiste el log de fallo).

### 4.1b Refinar un breakdown existente

`POST /api/v1/studio/text/claude/refine-shots` (auth JWT) — regenera el breakdown aplicando **solo** los cambios pedidos por el usuario (anti-drift).

**Request** (`ClaudeRefineShotsRequest`):

```json
{
  "scene_id": "uuid",             // obligatorio
  "project_id": "uuid",           // obligatorio
  "previous_response": "{...}",   // obligatorio — data.text crudo del generate-shots anterior
  "change_request": "hacé el tono más oscuro y acortá el shot C",  // obligatorio
  "model": "claude-shot-builder",
  "api_model": "claude-sonnet-4-6",
  "system_prompt": "",
  "skill_id": "",
  "generate_zh": false
}
```

**Response éxito** (200): mismo shape que generate-shots (`taskId`, `model`, `status: "succeeded"`, `text` con el JSON completo regenerado).

**Cómo funciona:**
- El prompt del usuario se compone como `=== Previous Breakdown ===` + JSON previo + `=== Change Request ===` + instrucción. El `scene_context` NO se reenvía (ya está embebido en el breakdown previo vía references/assetAssignments).
- System prompt = `defaultShotBuilderPrompt` + `## Refinement Mode` con `refineModeInstructions` (reglas anti-drift: aplicar solo el cambio pedido, conservar IDs, timestamps, slots `@imageN` y continuidad) + skill/instrucciones/reglas ZH/inglés igual que generate.
- Reutiliza el mismo pipeline: `runClaudeShotBuilder()` (retry 3 intentos, `extractJSON` + `validateShotJSON`), y `persistFailure()` con `mode='refine'`.
- El retry correctivo reenvía el breakdown previo + instrucción (nunca la respuesta truncada).

### 4.2 Optimizar prompt (Proncer)

`POST /api/v1/studio/text/claude/optimize-prompt` (auth JWT)

```json
{
  "scene_id": "uuid", "project_id": "uuid",
  "current_prompt": "...",       // obligatorio
  "user_instructions": "",
  "system_prompt": "",
  "skill_id": "",
  "shot_context": { "shot_name": "", "shot_description": "" },
  "scene_context": { ... }       // mismo SceneContext
}
```

Respuesta: `{ optimized_prompt, suggestions[], changes_made[], raw_text }`. Usa `resolveSystemPromptStrict()` (skill > system_prompt > **defaultProncerPrompt**; nunca cae al prompt del Shot Builder).

### 4.3 Logs de fallos

| Endpoint | Propósito |
|----------|-----------|
| `GET /api/v1/studio/text/claude/generate-shots-logs` | Listado paginado (page, limit ≤ 100) con filtros `project_id`, `scene_id`, `mode` (`generate` \| `refine`), `user_id`, `date_from`, `date_to` |
| `GET /api/v1/studio/text/claude/generate-shots-logs/:id` | Detalle de un log con sus attempts |

### 4.4 Skills

`CRUD /api/v1/skills` (auth JWT): `POST ""`, `GET ""`, `GET /:id`, `PATCH /:id`, `DELETE /:id`. Un skill es solo un `name + description + system_prompt` (texto) que se **apendiza** al prompt del Shot Builder.

### 4.5 Shots (relacionados, módulo project)

| Endpoint | Método | Propósito |
|----------|--------|-----------|
| `/projects/{id}/chapters/{chId}/scenes/{scId}/shots` | POST | Crear shot (usado por el frontend tras el JSON de Claude) |
| `/projects/{id}/chapters/{chId}/scenes/{scId}/shots` | GET | Listar shots de la escena |
| `/projects/{id}/chapters/{chId}/scenes/{scId}/shots/{shId}` | PATCH | Actualizar shot (incluye `aspect_ratio`, `duration_seconds`) |
| `/projects/{id}/chapters/{chId}/scenes/{scId}/assignments` | GET | Personajes, presets y assets asignados a la escena (fuente del scene_context) |

---

## 5. Composición del system prompt

En `ClaudeGenerateShots` (handler.go) el prompt se arma **siempre en este orden**:

```
defaultShotBuilderPrompt            ← DCS-DIRECTION (schema + reglas críticas). Siempre primero.
+ "\n\n## Skill\n" + skill.SystemPrompt          ← solo si skill_id válido
+ "\n\n## User instructions\n" + req.SystemPrompt ← solo si viene del request
+ instrucción "no generar ZH"                     ← solo si generate_zh == false
+ regla de idioma (output SIEMPRE en inglés)      ← siempre
```

Puntos clave del `defaultShotBuilderPrompt`:

- **Misión DCS-DIRECTION**: convertir guiones literarios en prompts Seedance con actuación humana real (nada de caricatura).
- **Filosofía**: realismo analógico, actuación física (verbos transitivos, nunca adjetivos de emoción tipo "angry"), composición bloqueada, golden rule of diffusion (referencias `@Image`/`@Video`).
- **Parseo de guion**: bloques `NN. INT/EXT. LOCACIÓN — TIEMPO` → `scriptNumber`, `scriptLocation`, tipo de escena (present/flashback/fantasy/dream/montage).
- **Asignación de assets a nivel de episodio**: `episode.assetAssignments` con slots `@imageN` y tipos `character | location | prop | audio`.
- **Continuity Engine**: rastrea locación, tiempo, personajes, estado emocional/físico, vestuario, props entre escenas.
- **4 fases**: análisis de subtexto (Weston) → física (Mamet) → cinematografía → ensamblaje multimodal.
- **Anatomy Library**: diccionario de emociones en términos musculares + micro-fidgeting (eye darts, nostril flares...).
- **Cinema Modes M1–M5** con lente/movimiento/grade por modo.
- **Prompt Engine**: el pre-prompt `prompt.en` es un texto plano de 220–360 palabras con secciones fijas en orden: `Scene and Mood / Composition / Space and Mélange / Cross-Shot Rule / Action / Dialogue / Ending Shot / Environmental Base / Sound Layer` + párrafo final de captura.
- **Salida**: solo JSON `{ episode, description, duration, mode, aspectRatio, directorNotes, scenes[] }`, sin texto alrededor, sin markdown fences.
- Los shots del JSON son **slim**: `id, title, description, duration, start, end, references, prompt{en,zh}, notes` — NO camera/composition/blocking/acting como sub-objetos (esa dirección vive dentro de `prompt.en`). *(Los tipos Go `Shot`/`Camera`/`Composition`/etc. en `claude_types.go` conservan los campos ricos por compatibilidad con el formato legacy, pero el prompt actual instruye a Claude a no emitirlos.)*
- `prompt.zh` solo si el usuario pidió generación en chino.
- Regla final: la respuesta debe empezar con `{` y terminar con `}`.

**Validación del prompt** (para no romper el string de Go): hay tests que verifican que `defaultShotBuilderPrompt` no contenga backticks crudos y que tenga todas las secciones requeridas (`handler_test.go`).

---

## 6. Llamada a Claude (`callClaude`)

- **SDK**: `github.com/anthropics/anthropic-sdk-go`.
- **Modelo**: se resuelve con `modelNameMap` (`claude-shot-builder` → `claude-sonnet-4-6`). Mapa actual:
  - `claude-shot-builder`, `claude-assistant`, `claude-sonnet-4-6` → `claude-sonnet-4-6`
  - `claude-opus-4-8` → `claude-opus-4-8`
  - `claude-haiku-4-5` → `claude-haiku-4-5`
  - `claude-fable-5` → `claude-fable-5`
- **API key**: `resolveAPIKey(keyModel)` busca en el provider store un modelo con ese nombre (`providerStore.GetModelByName`) y usa su `APIKey`. Sin key configurada → `""` (la llamada fallará).
- **Contexto detached**: usa `context.WithTimeout(context.Background(), 20*time.Minute)` — la llamada **sobrevive a la desconexión del cliente** (el Shot Builder puede tardar >5 min).
- **Timeouts**: `option.WithRequestTimeout(15*time.Minute)`.
- **MaxTokens**: `32768` (para EN + ZH; el retry corrige respuestas truncadas).
- **Prompt caching**: `CacheControlEphemeralTTL5m` en el bloque system. Reduce el costo en iteraciones seguidas (el system prompt es grande). Si se toca el system prompt, el caché se invalida.
- **Errores**: extrae `apiErr.RawJSON() → error.message` para mensajes legibles de la API.

---

## 7. Retry y validación de JSON

- **Hasta 3 intentos** (`maxAttempts = 3`).
- Cada intento se guarda como `ShotBuilderAttempt` en memoria (`attempt_number`, prompt enviado, respuesta cruda, `valid`, tokens, duración, error).
- **Extracción**: `extractJSON()` encuentra el objeto JSON más externo balanceando llaves **dentro y fuera de strings** (maneja `{...}` dentro de `prompt.en`).
- **Validación**: `validateShotJSON()` acepta dos formatos:
  - **Nuevo**: `{ "episode": {...}, "scenes": [{ "shots": [...] }] }` (requiere episode + ≥1 escena con shots).
  - **Legacy**: `{ "shots": [...] }` plano.
- **Retry correctivo**: si falla, **NO reenvía la respuesta truncada previa** (quema contexto y produce el mismo resultado). Envía `buildCorrectivePrompt(originalPrompt)` que **reescribe el guion original** + instrucción de brevedad. *Bug documentado: reintentar sin el guion original hace que Claude responda "No script provided".*
- Si se agotan los intentos → `buildExhaustionError()` + 500 + log de fallo.

---

## 8. Logs de fallos (shot_builder_logs)

Regla: **solo se loguean los fallos** ("log-only-failures"). Un éxito NO escribe nada en estas tablas.

**`shot_builder_logs`** — una fila por llamada fallida a generate-shots. Guarda todo lo necesario para reconstruir el request:
- Usuario (denormalizado del JWT: `user_id`, `user_name`, `user_email`), `project_id`, `scene_id`.
- `request_payload` (raw body tal cual llegó, incluye `scene_context` con los recursos asignados), `system_prompt` final, `prompt` final (guion + contexto).
- Modelos (`key_model`, `api_model`), skill (`skill_id`, `skill_name`).
- `status = 'failed'`, `error_message`, `response` (JSON extraído del último intento), `attempts`, tokens totales, `duration_ms`.
- Soft-delete (`deleted_at`).

**`shot_builder_attempts`** — una fila por llamada a la API de Claude dentro de un log fallido (FK `log_id` ON DELETE CASCADE). `attempt_number`, prompt del intento, respuesta cruda, `valid`, `error_message`, tokens (input/output/cache read/cache creation), `duration_ms`.

La persistencia se hace en `persistFailure()` y un error de logging **nunca enmascara** el error real que ya se envió al cliente (solo `log.Printf`).

---

## 9. Scene Context (buildSceneContextBlock)

Antepone al prompt del usuario un bloque con este formato (lo usan Shot Builder **y** Proncer):

```
=== Scene Context ===
Description: ...
Characters (use the id field as assetId):
  - Nombre [slot: @image1] [id: uuid]
Cinematography presets:
  - Label (code): prompt
Reference assets:
  - filename (mime_type) [id: uuid]
```

- Los `characters` llevan `id` para que Claude los use como `assetId` en `assetAssignments` / `references`.
- Los `assets` NO llevan tipo de slot por defecto: Claude infiere `character | location | prop | audio` desde el contexto del guion.
- El frontend arma el `scene_context` **solo con datos a nivel de escena** (descripción cruda, personajes asignados, assets libres). Los presets de cinematografía NO se incluyen a nivel de escena (solo sus IDs), por eso el scene_context del Shot Builder suele ir sin `presets`.

---

## 10. Modelo de datos (migraciones relevantes)

| Migración | Contenido |
|-----------|-----------|
| `00009_scene_assignments.sql` | `scene_presets`, `scene_characters`, `scene_assets` (asignaciones de escena — fuente del scene_context) |
| `00012_chapters_shots.sql` | `chapters`, `shots`, `scenes.chapter_id`, `takes.shot_id` + migración de datos |
| `00016_shot_resources.sql` | Recursos por shot (personajes/assets/presets del shot) |
| `00017_skills.sql` | Tabla `skills` (id TEXT, name, description, system_prompt, soft-delete) |
| `00018_shot_output_format.sql` | `shots.aspect_ratio`, `shots.duration_seconds` |
| `00020_character_slots.sql` | Slots de personajes |
| `00023_shots_number_not_unique.sql` | **Dropea** `UNIQUE(scene_id, number)` en shots — permite duplicar número (clonar shot). Takes mantienen su propio unique. |
| `00024_shot_builder_logs.sql` | `shot_builder_logs` + `shot_builder_attempts` |
| `00025_shot_builder_logs_mode.sql` | Columna `mode` (`generate` \| `refine`) en `shot_builder_logs` + índice |

> **Ojo al revertir** `00023`: el `Down` re-crea la constraint y falla si ya hay duplicados.

---

## 11. Puntos críticos / gotchas para cambios

1. **Dos flujos de texto distintos**: el Shot Builder usa `ClaudeGenerateShots` (síncrono, Anthropic SDK). Los endpoints `POST /studio/text/generate|status|task|preview` son stubs ("not yet implemented") — no confundirlos.
2. **El prompt es un string Go con backticks**: cualquier backtick crudo dentro de `defaultShotBuilderPrompt` rompe la compilación. Correr `handler_test.go` después de tocarlo (`TestPromptNoBackticks`, `TestPromptMentionsEpisode`, `TestPromptHasFormatRule`).
3. **Formato de salida estricto**: Claude debe devolver SOLO JSON. Si cambia el schema del JSON de salida, actualizar **a la vez**: el prompt (ejemplo en `defaultShotBuilderPrompt`), `validateShotJSON()`, los tipos en `claude_types.go`, y el parseo del frontend (Sequence Viewer).
4. **Contrato con el frontend**: el frontend parsea `data.text` (JSON crudo). Los campos `episode/scenes/directorNotes` de la respuesta NO se pueblan — no asumir lo contrario.
5. **Logs solo en fallo**: éxitos no se guardan. Si se necesita trazabilidad de éxitos, hay que cambiarlo a propósito (afecta volumen de datos).
6. **API key de Claude**: se resuelve por nombre de modelo en el provider store (`model` = clave). Agregar un modelo nuevo requiere entrada en `modelNameMap` **y** un registro en Providers con su API key.
7. **generate_zh = false** agrega una instrucción al system prompt pidiendo omitir `prompt.zh` (ahorra tokens).
8. **Salida siempre en inglés** (regla inyectada) aunque el usuario escriba en español.
9. **Timeout largo a propósito**: el sistema prompt es grande y puede tardar >5 min; el contexto detached evita que se cancele con la desconexión del cliente.
10. **Números de shot duplicables** (migración 00023): si algo asume unicidad de `(scene_id, number)` se rompe — revisar consultas de shots.
11. **Cache de 5 min**: tocar el system prompt invalida el caché de prompt caching (costo en la siguiente llamada).
12. **i18n**: si se agregan textos nuevos al frontend del Shot Builder, agregar keys en `en` y `es`.

---

## 12. Tests existentes (backend)

`internal/modules/studio/text/handler_test.go`:
- `TestPromptNoBackticks` / `TestPromptMentionsEpisode` / `TestPromptHasFormatRule` — integridad del system prompt.
- `TestExtractJSON` (+`TestExtractJSONStringBraces`) — extracción de JSON balanceado.
- `TestValidateShotJSON` — formatos nuevo y legacy.
- `TestJSONExampleIsValid` / `TestPreFillWorks` — el ejemplo del prompt parsea y valida.
- `TestBuildCorrectivePrompt` — el retry resend el guion original.
- `TestBuildCorrectivePromptFrom` — el retry del refine resend el breakdown previo.
- `TestRefineModeInstructions` — reglas anti-drift del modo refine.
- `TestBuildExhaustionError` — mensaje de agotamiento.
- `TestParseOptimizeResponse` — parser del Proncer.

Para correr: `go test ./internal/modules/studio/text/...` (desde `dcs-back/`).

---

## 13. Archivos relacionados / docs

- `docs/shot-builder-flow.md` — flujo completo de UI (frontend) y endpoints relevantes.
- `docs/claude-integration.md` — visión/guía de integración con Claude (en parte superada por la implementación real).
- `internal/modules/studio/ARCHITECTURE.md` — arquitectura del pipeline de generación (el Shot Builder es un flujo paralelo).
- `AGENTS.md` — convenciones del repo backend.
