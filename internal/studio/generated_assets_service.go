package studio

import (
	"fmt"

	
)

// saveGeneratedAssets guarda las URLs de salida como GeneratedAssets (status: pending)
// cuando una tarea de generación se completa exitosamente.
func (s *Service) saveGeneratedAssets(taskID string, result *GeneratorResult) {
	if s.assetStore == nil || s.logStore == nil {
		return
	}

	// Obtener el log para tener datos de sesión (user_id, project, scene, take)
	log, logErr := s.logStore.GetByTaskID(taskID)
	if logErr != nil || log == nil {
		return
	}

	for _, out := range result.Outputs {
		asset := &GeneratedAsset{
			TaskID:      taskID,
			ModelName:   log.ModelName,
			UserID:      log.UserID,
			ProjectID:   log.ProjectID,
			SceneID:     log.SceneID,
			SceneCode:   log.SceneCode,
			TakeNumber:  log.TakeNumber,
			OriginalURL: out.URL,
			Status:      "pending",
		}
		if asset.OriginalURL == "" {
			asset.OriginalURL = out.LocalURL
		}
		if asset.OriginalURL == "" {
			continue
		}
		if err := s.assetStore.Create(asset); err != nil {
			fmt.Printf("failed to save generated asset for task %s: %v\n", taskID, err)
		}
	}
}
