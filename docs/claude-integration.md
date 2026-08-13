# DCS Studio — Integración de Claude API en Go

> Director Assistant para Dead Camera Studios
> Stack: Go 1.24+ · Anthropic SDK oficial · SSE para streaming

---

## 1. Resumen

Este documento describe cómo integrar Claude API como asistente conversacional de prompts dentro de DCS Studio. El asistente:

- Mantiene contexto multi-turno (conversación real, no Q&A aislado).
- Carga dinámicamente skills (`SKILL.md`) según el modo de trabajo activo.
- Recibe el estado vivo de la UI (escena actual, lente, grade, assets cargados) en cada turno.
- Aprovecha **prompt caching** para reducir ~90% el costo de iteración intensiva.
- Soporta streaming token-por-token para UX de chat fluido.

**Arquitectura:**

```
┌──────────────┐    POST /api/director-chat    ┌──────────────┐
│  Angular UI  │ ─────────────────────────────> │   Go API     │
│ (dcs-front)  │                                │   (this)     │
│              │ <───────────────────────────── │              │
└──────────────┘     SSE stream / JSON           └──────┬───────┘
                                                       │
                                                       │ Messages API
                                                       ▼
                                                ┌──────────────┐
                                                │ Anthropic    │
                                                │ Claude API   │
                                                └──────────────┘
```

---

## 2. Setup

### Dependencias

```bash
go get github.com/anthropics/anthropic-sdk-go
```

Requiere **Go 1.24+**.

### Variables de entorno

```bash
# .env
ANTHROPIC_API_KEY=sk-ant-...
PORT=8080
SKILLS_DIR=./skills
```

### Estructura de directorios esperada

```
dcs-director-assistant/
├── main.go
├── go.mod
├── .env
└── skills/
    ├── seedance-dramatic-prompt/
    │   └── SKILL.md
    ├── cinema-worldbuilder-pro-20/
    │   └── SKILL.md
    └── script-to-seedance/
        └── SKILL.md
```

Cada skill es un folder con su `SKILL.md` — mismo formato que ya usas en tu pipeline actual.

---

## 3. Contrato API

### Request

`POST /api/director-chat`

```json
{
  "active_skill": "seedance-dramatic-prompt",
  "scene_context": {
    "episode": "EP08",
    "shot": "S03",
    "characters": ["Wyatt"],
    "lens": "50mm",
    "grade": "OHIO",
    "aspect": "9:16",
    "mode": "M1_NARRATIVE",
    "last_validated_prompt": "..."
  },
  "messages": [
    { "role": "user", "content": "..." },
    { "role": "assistant", "content": "..." },
    { "role": "user", "content": "..." }
  ]
}
```

**Reglas:**

- `messages` es el historial completo de la sesión. El frontend lo posee y lo crece en cada turno.
- `scene_context` puede cambiar entre turnos — refleja el estado vivo de la UI.
- `active_skill` es el folder dentro de `skills/`.

### Response (no-streaming)

```json
{
  "reply": "...",
  "usage": {
    "input_tokens": 1234,
    "output_tokens": 567,
    "cache_read_tokens": 8900,
    "cache_created_tokens": 0
  }
}
```

### Response (streaming)

Server-Sent Events:

```
data: First chunk of text
data: Second chunk
data: ...
data: [DONE]
```

---

## 4. Implementación — Servidor HTTP base

```go
// main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// --- Modelos de datos compartidos con el frontend ---

type ChatMessage struct {
	Role    string `json:"role"`    // "user" | "assistant"
	Content string `json:"content"`
}

type ChatRequest struct {
	ActiveSkill  string         `json:"active_skill"`
	SceneContext map[string]any `json:"scene_context"`
	Messages     []ChatMessage  `json:"messages"`
}

type ChatResponse struct {
	Reply string         `json:"reply"`
	Usage map[string]int `json:"usage"`
}

// --- Construcción del system prompt con skill inyectado ---

func buildSystemPrompt(skillName string, sceneContext map[string]any) (string, error) {
	skillPath := fmt.Sprintf("skills/%s/SKILL.md", skillName)
	skillBytes, err := os.ReadFile(skillPath)
	if err != nil {
		return "", fmt.Errorf("reading skill %s: %w", skillName, err)
	}

	ctxJSON, _ := json.MarshalIndent(sceneContext, "", "  ")

	return fmt.Sprintf(`You are the prompt director for Swipe Right (Dead Camera Studios).

ACTIVE SKILL LOADED:
%s

CURRENT PROJECT STATE:
%s

INSTRUCCIONES:
- Responde conversacionalmente en español.
- Los prompts de Seedance van siempre en inglés + 中文.
- Respeta el límite de 3,500 caracteres por idioma.
- Itera basado en el feedback del director sin reescribir todo cada vez.
- Si el director dice "bájale a X" o "cambia Y", modifica solo eso y mantén el resto.
`, string(skillBytes), string(ctxJSON)), nil
}

// --- Handler no-streaming ---

func handleChat(client *anthropic.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		systemText, err := buildSystemPrompt(req.ActiveSkill, req.SceneContext)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convertir historial al formato del SDK
		messages := make([]anthropic.MessageParam, 0, len(req.Messages))
		for _, m := range req.Messages {
			block := anthropic.NewTextBlock(m.Content)
			switch m.Role {
			case "user":
				messages = append(messages, anthropic.NewUserMessage(block))
			case "assistant":
				messages = append(messages, anthropic.NewAssistantMessage(block))
			}
		}

		// System prompt con CACHE habilitado — clave para abaratar iteración
		systemBlocks := []anthropic.TextBlockParam{
			{
				Text: systemText,
				CacheControl: anthropic.CacheControlEphemeralParam{
					TTL: anthropic.CacheControlEphemeralTTLTTL5m,
				},
			},
		}

		resp, err := client.Messages.New(r.Context(), anthropic.MessageNewParams{
			Model:     anthropic.ModelClaudeOpus4_7,
			MaxTokens: 4096,
			System:    systemBlocks,
			Messages:  messages,
		})
		if err != nil {
			log.Printf("anthropic error: %v", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		// Concatenar bloques de texto de la respuesta
		var reply string
		for _, block := range resp.Content {
			if textBlock, ok := block.AsAny().(anthropic.TextBlock); ok {
				reply += textBlock.Text
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatResponse{
			Reply: reply,
			Usage: map[string]int{
				"input_tokens":         int(resp.Usage.InputTokens),
				"output_tokens":        int(resp.Usage.OutputTokens),
				"cache_read_tokens":    int(resp.Usage.CacheReadInputTokens),
				"cache_created_tokens": int(resp.Usage.CacheCreationInputTokens),
			},
		})
	}
}

func main() {
	client := anthropic.NewClient(
		option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/director-chat", handleChat(&client))
	mux.HandleFunc("/api/director-chat-stream", handleChatStream(&client))

	addr := ":" + getenv("PORT", "8080")
	log.Println("Director Assistant ready on", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

---

## 5. Implementación — Streaming (recomendado para producción)

Para UX de chat fluido (texto aparece palabra por palabra):

```go
func handleChatStream(client *anthropic.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		systemText, err := buildSystemPrompt(req.ActiveSkill, req.SceneContext)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		messages := make([]anthropic.MessageParam, 0, len(req.Messages))
		for _, m := range req.Messages {
			block := anthropic.NewTextBlock(m.Content)
			if m.Role == "user" {
				messages = append(messages, anthropic.NewUserMessage(block))
			} else {
				messages = append(messages, anthropic.NewAssistantMessage(block))
			}
		}

		// Headers SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		stream := client.Messages.NewStreaming(r.Context(), anthropic.MessageNewParams{
			Model:     anthropic.ModelClaudeOpus4_7,
			MaxTokens: 4096,
			System: []anthropic.TextBlockParam{{
				Text: systemText,
				CacheControl: anthropic.CacheControlEphemeralParam{
					TTL: anthropic.CacheControlEphemeralTTLTTL5m,
				},
			}},
			Messages: messages,
		})

		for stream.Next() {
			event := stream.Current()
			if delta, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
				if textDelta, ok := delta.Delta.AsAny().(anthropic.TextDelta); ok {
					// Escapar saltos de línea para SSE
					fmt.Fprintf(w, "data: %s\n\n", textDelta.Text)
					flusher.Flush()
				}
			}
		}

		if err := stream.Err(); err != nil {
			log.Printf("stream error: %v", err)
		}

		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}
}
```

---

## 6. Consumo desde el frontend (Angular / dcs-front)

### Servicio Angular

```typescript
// director-chat.service.ts
import { Injectable } from "@angular/core";
import { HttpClient } from "@angular/common/http";
import { Observable } from "rxjs";

export interface ChatMessage {
  role: "user" | "assistant";
  content: string;
}

export interface SceneContext {
  episode: string;
  shot: string;
  characters: string[];
  lens: string;
  grade: string;
  aspect: string;
  mode: string;
}

export interface ChatRequest {
  active_skill: string;
  scene_context: SceneContext;
  messages: ChatMessage[];
}

export interface ChatResponse {
  reply: string;
  usage: {
    input_tokens: number;
    output_tokens: number;
    cache_read_tokens: number;
    cache_created_tokens: number;
  };
}

@Injectable({ providedIn: "root" })
export class DirectorChatService {
  private endpoint = "/api/director-chat";
  private streamEndpoint = "/api/director-chat-stream";

  constructor(private http: HttpClient) {}

  sendMessage(req: ChatRequest): Observable<ChatResponse> {
    return this.http.post<ChatResponse>(this.endpoint, req);
  }

  streamMessage(
    req: ChatRequest,
    onChunk: (text: string) => void,
  ): Promise<void> {
    return new Promise(async (resolve, reject) => {
      try {
        const response = await fetch(this.streamEndpoint, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(req),
        });

        const reader = response.body!.getReader();
        const decoder = new TextDecoder();

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          const chunk = decoder.decode(value);
          const lines = chunk.split("\n\n");

          for (const line of lines) {
            if (line.startsWith("data: ")) {
              const data = line.slice(6);
              if (data === "[DONE]") {
                resolve();
                return;
              }
              onChunk(data);
            }
          }
        }
        resolve();
      } catch (err) {
        reject(err);
      }
    });
  }
}
```

### Componente de chat (esqueleto)

```typescript
// director-chat.component.ts
import { Component, signal } from "@angular/core";
import { DirectorChatService, ChatMessage } from "./director-chat.service";

@Component({
  selector: "app-director-chat",
  template: `
    <div class="chat-section">
      <h2>07 DIRECTOR ASSISTANT</h2>
      <div class="messages">
        <div *ngFor="let msg of messages()" [class]="msg.role">
          {{ msg.content }}
        </div>
        <div *ngIf="streaming()" class="assistant streaming">
          {{ currentStream() }}
        </div>
      </div>
      <textarea [(ngModel)]="input" (keydown.enter)="send()"></textarea>
      <button (click)="send()" [disabled]="streaming()">Send</button>
      <button (click)="regenerate()" [disabled]="streaming()">
        ↻ Regenerate
      </button>
      <button (click)="reset()">Reset</button>
    </div>
  `,
})
export class DirectorChatComponent {
  messages = signal<ChatMessage[]>([]);
  currentStream = signal("");
  streaming = signal(false);
  input = "";

  constructor(
    private chat: DirectorChatService,
    private studioState: StudioStateService, // tu servicio existente con la UI state
  ) {}

  async send() {
    const userMsg: ChatMessage = { role: "user", content: this.input };
    this.messages.update((m) => [...m, userMsg]);
    this.input = "";
    this.streaming.set(true);
    this.currentStream.set("");

    await this.chat.streamMessage(
      {
        active_skill: this.studioState.activeSkill(),
        scene_context: this.studioState.sceneContext(),
        messages: this.messages(),
      },
      (chunk) => this.currentStream.update((s) => s + chunk),
    );

    this.messages.update((m) => [
      ...m,
      { role: "assistant", content: this.currentStream() },
    ]);
    this.currentStream.set("");
    this.streaming.set(false);
  }

  regenerate() {
    // Quita la última respuesta del assistant y reenvía
    this.messages.update((m) => m.slice(0, -1));
    this.send();
  }

  reset() {
    this.messages.set([]);
  }
}
```

---

## 7. Prompt Caching — Por qué es crítico para DCS

El system prompt completo (SKILL.md + scene context) puede pesar fácil **10,000+ tokens**. Sin caché, pagás eso en cada turno de chat. Con `CacheControl: ephemeral`:

| Escenario             | Sin caché       | Con caché               |
| --------------------- | --------------- | ----------------------- |
| Turno 1 (cold)        | 100% input cost | 100% (escribe el caché) |
| Turno 2 (warm, <5min) | 100%            | ~10%                    |
| Turno 3 (warm)        | 100%            | ~10%                    |
| Turno 4 (warm)        | 100%            | ~10%                    |

**Para una sesión típica de 15 iteraciones sobre una toma**, el ahorro es ~85% en costo de input. Esto convierte el asistente de "carísimo" a "barato".

**Caveat:** el caché expira a los 5 minutos sin uso. Si el director vuelve después de un break largo, la primera llamada paga de nuevo. Si necesitás caché más largo, usá `CacheControlEphemeralTTLTTL1h` (1 hora, premium).

---

## 8. Endpoints de tooling extendido (v2, opcional)

Cuando el chat básico esté funcionando, agregale **tool use** para que Claude pueda actuar sobre la app:

```go
// Tools que Claude puede invocar
tools := []anthropic.ToolUnionParam{
    {
        OfTool: &anthropic.ToolParam{
            Name:        "save_prompt_to_library",
            Description: "Guarda un prompt validado al Takes Reel del proyecto.",
            InputSchema: anthropic.ToolInputSchemaParam{
                Properties: map[string]any{
                    "scene_id": map[string]any{"type": "string"},
                    "prompt_en": map[string]any{"type": "string"},
                    "prompt_zh": map[string]any{"type": "string"},
                    "tags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
                },
                Required: []string{"scene_id", "prompt_en", "prompt_zh"},
            },
        },
    },
    {
        OfTool: &anthropic.ToolParam{
            Name:        "fetch_recent_takes",
            Description: "Lee las últimas N takes generadas para una escena.",
            InputSchema: anthropic.ToolInputSchemaParam{
                Properties: map[string]any{
                    "scene_id": map[string]any{"type": "string"},
                    "limit": map[string]any{"type": "integer"},
                },
                Required: []string{"scene_id"},
            },
        },
    },
}
```

Esto deja al asistente ejecutar acciones reales en tu pipeline, no solo dar consejos.

---

## 9. Consideraciones de producción

### Modelos disponibles

- `anthropic.ModelClaudeOpus4_7` — máxima calidad de razonamiento, recomendado para prompt engineering.
- `anthropic.ModelClaudeSonnet4_6` — balance velocidad/calidad para iteración rápida.
- `anthropic.ModelClaudeHaiku4_5` — ultra rápido, para tareas simples (clasificación, parsing).

### Rate limits

Anthropic enforza rate limits por workspace. Para producción, implementar:

- Retry con exponential backoff (el SDK lo trae por default).
- Queue interno si múltiples directores usan el mismo backend.

### Seguridad

- **NUNCA** exponer `ANTHROPIC_API_KEY` al frontend. Siempre proxy a través del backend Go.
- Validar `active_skill` contra una whitelist antes de leer archivos (evitar path traversal).

### Observabilidad

Loguear `usage.cache_read_tokens` vs `cache_created_tokens` para validar que el caché está funcionando. Si siempre ves `created` y nunca `read`, hay un bug en la construcción del system prompt (estás cambiando algo que debería ser estable).

### Timeout

El SDK aborta non-streaming requests >10min por default. Para tareas largas, usá `NewStreaming` o configurá timeout custom:

```go
client := anthropic.NewClient(
    option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    option.WithRequestTimeout(15 * time.Minute),
)
```

---

## 10. Roadmap sugerido

| Fase     | Entregable                                         | Estimado |
| -------- | -------------------------------------------------- | -------- |
| **v1.0** | Endpoint no-streaming + chat básico en Angular     | 2-3 días |
| **v1.1** | Streaming SSE + UX (regenerate, edit, reset)       | 1-2 días |
| **v1.2** | Inyección de scene context en vivo desde la UI     | 1 día    |
| **v2.0** | Tool use: save_to_library, fetch_takes, edit_asset | 3-5 días |
| **v2.1** | Branches de conversación (takes alternos)          | 2 días   |
| **v3.0** | Migración a Agent Skills oficial cuando haga falta | TBD      |

---

## 11. Recursos

- **SDK oficial:** https://github.com/anthropics/anthropic-sdk-go
- **Docs Messages API:** https://platform.claude.com/docs/en/api/messages
- **Prompt Caching:** https://platform.claude.com/docs/en/build-with-claude/prompt-caching
- **Agent Skills:** https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview
- **Tool Use:** https://platform.claude.com/docs/en/build-with-claude/tool-use

---

_Documento técnico — Dead Camera Studios · The Electric Mind_
