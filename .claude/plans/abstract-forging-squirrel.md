# Plan: Simplificar GetStatus / statusFromLog y actualizar logs

## Context

Actualmente `GetStatus` tiene dos caminos cuando un task se completa:

1. **Path en memoria** (task en `s.tasks`): llama `gen.GetStatus()` → descarga video → `updateLogWithFinalStatus` → `saveToTakes` → `saveGeneratedAssets`
2. **Path statusFromLog** (task no en memoria, ej. después de reinicio): llama `gen.GetStatus()` → descarga video OTRA VEZ → `updateLogWithFinalStatus` → `saveGeneratedAssets` → luego `GetStatus` llama `saveToTakes`

Problemas:
- `statusFromLog` llama a `gen.GetStatus()` incluso cuando el log ya tiene estado final (succeeded/failed), forzando una descarga redundante del video
- El generation_log se actualiza con `outputs` (JSON con URL/LocalURL), pero no hay un campo dedicado para `video_url` que relacione el log con el take creado
- `saveToTakes` se llama en ambos paths, lo que es redundante (aunque `SaveGeneration` ya es idempotente)

## Objetivo

1. **Simplificar statusFromLog**: si el log ya tiene estado final (succeeded/failed), no llamar a `gen.GetStatus()`. Solo devolver la info del log con los outputs ya guardados.
2. **Actualizar log con info del take**: después de `saveToTakes`, actualizar el generation_log con el `video_url` y `video_local_url` del resultado. Esto asegura que aunque se consulte el log después, tenga los datos actualizados.

## Cambios

### 1. service.go — `statusFromLog()` (línea 907)

- **Al inicio**: si `log.Status` es `succeeded` o `failed`, devolver inmediatamente un `StatusResult` con los outputs del log (sin llamar al generador).
- Esto evita la descarga redundante del video y la llamada HTTP a la API externa.

```go
func (s *Service) statusFromLog(log *GenerationLog) (*StatusResult, error) {
    // Si el log ya tiene estado final, devolver lo que tiene
    if log.Status == config.STATUS_SUCCESS || log.Status == config.STATUS_FAILED {
        sr := &StatusResult{
            Status: log.Status,
            Error:  log.ErrorMessage,
        }
        if len(log.Outputs) > 0 {
            sr.VideoURL = log.Outputs[0].URL
            sr.VideoLocalURL = log.Outputs[0].LocalURL
        }
        return sr, nil
    }
    // ... resto del método igual (llamar al generador si está running/pending)
}
```

### 2. service.go — `saveToTakes()` (línea 981)

- **Después de llamar a `takeSaver` exitosamente**: actualizar el generation_log con los outputs actualizados (video_url, video_local_url).
- Esto asegura que el log refleje la info del take que se acaba de crear/actualizar.

```go
func (s *Service) saveToTakes(taskID string, videoURL, localURL string) {
    if s.takeSaver == nil || s.logStore == nil {
        return
    }
    log, logErr := s.logStore.GetByTaskID(taskID)
    if logErr != nil || log == nil {
        return
    }
    if log.SceneID == "" || log.TakeNumber <= 0 {
        return
    }

    if err := s.takeSaver(log.SceneID, log.TakeNumber, videoURL, localURL); err != nil {
        fmt.Printf("failed to save take for task %s: %v\n", taskID, err)
        return
    }

    // Actualizar el generation_log con los outputs (video URL)
    outputs := []OutputResource{{URL: videoURL, LocalURL: localURL}}
    if err := s.logStore.UpdateByTaskID(taskID, outputs, log.Status, log.ErrorMessage); err != nil {
        fmt.Printf("failed to update log outputs for task %s: %v\n", taskID, err)
    }
}
```

### 3. service.go — `GetStatus()` path en memoria (línea 1084-1086)

- No requiere cambios. Ya llama `updateLogWithFinalStatus` + `saveToTakes`. Con el cambio #2, `saveToTakes` actualizará el log, pero como `updateLogWithFinalStatus` se llama primero, los outputs ya están guardados. `saveToTakes` sobrescribirá con los mismos datos. Es idempotente.

### 4. service.go — `GetStatus()` path statusFromLog (línea 1015-1018)

- No requiere cambios. `saveToTakes` con el cambio #2 actualizará el log con los outputs.

## Archivos a modificar

- `internal/modules/studio/service.go` — `statusFromLog()` y `saveToTakes()`

## Archivos a NO modificar

- `internal/modules/studio/generation_log_store.go` — ya tiene `UpdateByTaskID`
- `internal/modules/project/service.go` — `SaveGeneration` ya funciona correctamente
- `internal/modules/studio/types.go` — tipos ya son suficientes

## Verificación

1. Build: `go build ./...`
2. Tests: `go test ./...`
3. Validación manual del flujo:
   - Task en memoria: GetStatus → gen.GetStatus() returns succeeded → saveToTakes crea take → log se actualiza con outputs
   - Task en log (status succeeded): statusFromLog retorna inmediatamente sin llamar al generador → saveToTakes se llama (idempotente)
   - Task en log (status running): statusFromLog llama al generador normalmente
