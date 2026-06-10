package studio

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type GenerationLogStore struct {
	db *sql.DB
}

func NewGenerationLogStore(db *sql.DB) *GenerationLogStore {
	return &GenerationLogStore{db: db}
}

// Light columns for list queries (avoids loading heavy text/blob columns).
const genLogListCols = `gl.id, gl.task_id, gl.model_name,
		gl.status,
		COALESCE(gl.error_message, '') AS error_message,
		gl.user_id,
		COALESCE(gl.project_id, '') AS project_id,
		COALESCE(gl.scene_id, '') AS scene_id,
		COALESCE(gl.shot_id, '') AS shot_id,
		COALESCE(gl.scene_code, '') AS scene_code,
		COALESCE(gl.take_number, 0) AS take_number,
		COALESCE(gl.outputs, '') AS outputs,
		gl.resource_type, gl.content_types,
		gl.estimated_cost, gl.cost_source,
		gl.created_at, gl.updated_at`

// Full columns for detail queries (includes request payload and outputs).
const genLogFullCols = `gl.id, gl.task_id, gl.model_name,
		COALESCE(gl.request_payload, '') AS request_payload,
		COALESCE(gl.outputs, '') AS outputs,
		gl.status,
		COALESCE(gl.error_message, '') AS error_message,
		gl.user_id,
		COALESCE(gl.project_id, '') AS project_id,
		COALESCE(gl.scene_id, '') AS scene_id,
		COALESCE(gl.shot_id, '') AS shot_id,
		COALESCE(gl.scene_code, '') AS scene_code,
		COALESCE(gl.take_number, 0) AS take_number,
		gl.resource_type, gl.content_types,
		gl.estimated_cost, gl.cost_source,
		gl.created_at, gl.updated_at, gl.deleted_at`

const genLogJoinCols = `COALESCE(u.username, '') AS user_name,
		COALESCE(u.name || ' ' || u.surname, '') AS user_display_name,
		COALESCE(p.name, '') AS project_name,
		COALESCE(s.name, '') AS scene_name,
		COALESCE(s.number, 0) AS scene_number`

const genLogFromJoins = `FROM generation_logs gl
		LEFT JOIN users u ON u.id = gl.user_id
		LEFT JOIN projects p ON p.id::text = gl.project_id
		LEFT JOIN scenes s ON s.id::text = gl.scene_id`

// scanListRow scans a list query row (without request payload).
func (s *GenerationLogStore) scanListRow(row *GenerationLog, scanner interface {
	Scan(dest ...interface{}) error
}) error {
	var outputsStr string
	err := scanner.Scan(
		&row.ID, &row.TaskID, &row.ModelName,
		&row.Status, &row.ErrorMessage,
		&row.UserID, &row.ProjectID, &row.SceneID, &row.ShotID, &row.SceneCode,
		&row.TakeNumber,
		&outputsStr,
		&row.ResourceType, &row.ContentTypes,
		&row.EstimatedCost, &row.CostSource,
		&row.CreatedAt, &row.UpdatedAt,
		&row.UserName, &row.UserDisplayName, &row.ProjectName, &row.SceneName, &row.SceneNumber,
	)
	if err != nil {
		return err
	}
	if outputsStr != "" {
		json.Unmarshal([]byte(outputsStr), &row.Outputs)
	}
	return nil
}

// scanDetailRow scans a detail query row (includes request payload and outputs).
func (s *GenerationLogStore) scanDetailRow(row *GenerationLog, scanner interface {
	Scan(dest ...interface{}) error
}) error {
	var outputsStr string
	err := scanner.Scan(
		&row.ID, &row.TaskID, &row.ModelName,
		&row.Request, &outputsStr,
		&row.Status, &row.ErrorMessage,
		&row.UserID, &row.ProjectID, &row.SceneID, &row.ShotID, &row.SceneCode,
		&row.TakeNumber,
		&row.ResourceType, &row.ContentTypes,
		&row.EstimatedCost, &row.CostSource,
		&row.CreatedAt, &row.UpdatedAt, &row.DeletedAt,
		&row.UserName, &row.UserDisplayName, &row.ProjectName, &row.SceneName, &row.SceneNumber,
	)
	if err != nil {
		return err
	}
	if outputsStr != "" {
		json.Unmarshal([]byte(outputsStr), &row.Outputs)
	}
	return nil
}

// Create inserts a new generation log entry.
func (s *GenerationLogStore) Create(log *GenerationLog) error {
	query := `INSERT INTO generation_logs (task_id, model_name, request_payload, outputs, status, error_message, user_id, project_id, scene_id, shot_id, scene_code, take_number, resource_type, content_types, estimated_cost, cost_source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at, updated_at`

	outputsStr := marshalOutputs(log.Outputs)

	return s.db.QueryRow(query,
		log.TaskID,
		log.ModelName,
		nullIfEmpty(log.Request),
		nullIfEmpty(outputsStr),
		log.Status,
		nullIfEmpty(log.ErrorMessage),
		log.UserID,
		nullIfEmpty(log.ProjectID),
		nullIfEmpty(log.SceneID),
		nullIfEmpty(log.ShotID),
		nullIfEmpty(log.SceneCode),
		log.TakeNumber,
		log.ResourceType,
		log.ContentTypes,
		log.EstimatedCost,
		log.CostSource,
	).Scan(&log.ID, &log.CreatedAt, &log.UpdatedAt)
}

// UpdateCost updates the estimated cost for a completed generation log.
func (s *GenerationLogStore) UpdateCost(taskID string, cost float64, source string) error {
	query := `UPDATE generation_logs
		SET estimated_cost = $1, cost_source = $2, updated_at = NOW()
		WHERE task_id = $3 AND deleted_at IS NULL`

	result, err := s.db.Exec(query, cost, source, taskID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("generation log not found for task: %s", taskID)
	}
	return nil
}

// GetByID returns a single log entry by its ID (includes full payload).
func (s *GenerationLogStore) GetByID(id string) (*GenerationLog, error) {
	log := &GenerationLog{}
	query := `SELECT ` + genLogFullCols + `, ` + genLogJoinCols + ` ` + genLogFromJoins + ` WHERE gl.id = $1 AND gl.deleted_at IS NULL`

	if err := s.scanDetailRow(log, s.db.QueryRow(query, id)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return log, nil
}

// GetByTaskID returns a log entry by its task ID (includes full payload).
func (s *GenerationLogStore) GetByTaskID(taskID string) (*GenerationLog, error) {
	log := &GenerationLog{}
	query := `SELECT ` + genLogFullCols + `, ` + genLogJoinCols + ` ` + genLogFromJoins + ` WHERE gl.task_id = $1 AND gl.deleted_at IS NULL`

	if err := s.scanDetailRow(log, s.db.QueryRow(query, taskID)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return log, nil
}

// UpdateByTaskID updates a log entry by its task ID (used when async tasks complete).
func (s *GenerationLogStore) UpdateByTaskID(taskID string, outputs []OutputResource, status, errorMessage string) error {
	query := `UPDATE generation_logs
		SET outputs = $1, status = $2, error_message = $3, updated_at = NOW()
		WHERE task_id = $4 AND deleted_at IS NULL`

	outputsStr := marshalOutputs(outputs)

	result, err := s.db.Exec(query,
		nullIfEmpty(outputsStr), status, nullIfEmpty(errorMessage), taskID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("generation log not found for task: %s", taskID)
	}
	return nil
}

// List returns paginated generation logs, newest first (light columns).
func (s *GenerationLogStore) List(page, limit int) ([]GenerationLog, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int
	countQuery := `SELECT COUNT(*) FROM generation_logs WHERE deleted_at IS NULL`
	if err := s.db.QueryRow(countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + genLogListCols + `, ` + genLogJoinCols + ` ` + genLogFromJoins + ` WHERE gl.deleted_at IS NULL
		ORDER BY gl.created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []GenerationLog
	for rows.Next() {
		var l GenerationLog
		if err := s.scanListRow(&l, rows); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ListByFilter returns paginated generation logs filtered by the given criteria, newest first.
// Empty filter values are ignored (no filter applied for that field).
func (s *GenerationLogStore) ListByFilter(page, limit int, projectID, sceneID, status, modelName string, userID int, dateFrom, dateTo, resourceType string) ([]GenerationLog, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	where := "WHERE gl.deleted_at IS NULL"
	args := []interface{}{}
	argIdx := 1

	if projectID != "" {
		where += fmt.Sprintf(" AND gl.project_id = $%d", argIdx)
		args = append(args, projectID)
		argIdx++
	}
	if sceneID != "" {
		where += fmt.Sprintf(" AND gl.scene_id = $%d", argIdx)
		args = append(args, sceneID)
		argIdx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND gl.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if modelName != "" {
		where += fmt.Sprintf(" AND gl.model_name ILIKE $%d", argIdx)
		args = append(args, "%"+modelName+"%")
		argIdx++
	}
	if userID > 0 {
		where += fmt.Sprintf(" AND gl.user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	if dateFrom != "" {
		where += fmt.Sprintf(" AND gl.created_at >= $%d", argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		where += fmt.Sprintf(" AND gl.created_at <= $%d", argIdx)
		args = append(args, dateTo+"T23:59:59Z")
		argIdx++
	}
	if resourceType != "" {
		where += fmt.Sprintf(" AND gl.resource_type = $%d", argIdx)
		args = append(args, resourceType)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM generation_logs gl " + where
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := "SELECT " + genLogListCols + ", " + genLogJoinCols + " " + genLogFromJoins + " " + where + " ORDER BY gl.created_at DESC LIMIT $" + fmt.Sprintf("%d", argIdx) + " OFFSET $" + fmt.Sprintf("%d", argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []GenerationLog
	for rows.Next() {
		var l GenerationLog
		if err := s.scanListRow(&l, rows); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// SumCostByFilter returns the total estimated_cost matching the given filters (no pagination).
func (s *GenerationLogStore) SumCostByFilter(projectID, sceneID, status, modelName string, userID int, dateFrom, dateTo, resourceType string) (float64, error) {
	where := "WHERE gl.deleted_at IS NULL"
	args := []interface{}{}
	argIdx := 1

	if projectID != "" {
		where += fmt.Sprintf(" AND gl.project_id = $%d", argIdx)
		args = append(args, projectID)
		argIdx++
	}
	if sceneID != "" {
		where += fmt.Sprintf(" AND gl.scene_id = $%d", argIdx)
		args = append(args, sceneID)
		argIdx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND gl.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if modelName != "" {
		where += fmt.Sprintf(" AND gl.model_name ILIKE $%d", argIdx)
		args = append(args, "%"+modelName+"%")
		argIdx++
	}
	if userID > 0 {
		where += fmt.Sprintf(" AND gl.user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	if dateFrom != "" {
		where += fmt.Sprintf(" AND gl.created_at >= $%d", argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		where += fmt.Sprintf(" AND gl.created_at <= $%d", argIdx)
		args = append(args, dateTo+"T23:59:59Z")
		argIdx++
	}
	if resourceType != "" {
		where += fmt.Sprintf(" AND gl.resource_type = $%d", argIdx)
		args = append(args, resourceType)
		argIdx++
	}

	query := "SELECT COALESCE(SUM(gl.estimated_cost), 0) FROM generation_logs gl " + where
	var total float64
	if err := s.db.QueryRow(query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// marshalOutputs marshals a []OutputResource slice to a JSON string.
// Returns an empty string if the slice is nil or empty.
func marshalOutputs(outputs []OutputResource) string {
	if len(outputs) == 0 {
		return ""
	}
	b, err := json.Marshal(outputs)
	if err != nil {
		return ""
	}
	return string(b)
}

// nullIfEmpty returns nil if s is empty, otherwise the string.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
