package studio

import (
	"database/sql"
	"fmt"
)

type GenerationLogStore struct {
	db *sql.DB
}

func NewGenerationLogStore(db *sql.DB) *GenerationLogStore {
	return &GenerationLogStore{db: db}
}

// colList is the columns used in SELECT queries, with COALESCE for nullable TEXT fields.
const genLogCols = `id, task_id, model_name,
	COALESCE(request_payload, '') AS request_payload,
	COALESCE(ai_response, '') AS ai_response,
	COALESCE(ai_call_payload, '') AS ai_call_payload,
	COALESCE(outputs, '') AS outputs,
	status,
	COALESCE(error_message, '') AS error_message,
	created_at, updated_at, deleted_at`

func (s *GenerationLogStore) scanRow(row *GenerationLog, scanner interface{ Scan(dest ...interface{}) error }) error {
	return scanner.Scan(
		&row.ID, &row.TaskID, &row.ModelName,
		&row.Request, &row.AIResponse, &row.AICallPayload,
		&row.Outputs, &row.Status, &row.ErrorMessage,
		&row.CreatedAt, &row.UpdatedAt, &row.DeletedAt,
	)
}

// Create inserts a new generation log entry.
func (s *GenerationLogStore) Create(log *GenerationLog) error {
	query := `INSERT INTO generation_logs (task_id, model_name, request_payload, ai_response, ai_call_payload, outputs, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	return s.db.QueryRow(query,
		log.TaskID,
		log.ModelName,
		nullIfEmpty(log.Request),
		nullIfEmpty(log.AIResponse),
		nullIfEmpty(log.AICallPayload),
		nullIfEmpty(log.Outputs),
		log.Status,
		nullIfEmpty(log.ErrorMessage),
	).Scan(&log.ID, &log.CreatedAt, &log.UpdatedAt)
}

// GetByID returns a single log entry by its ID.
func (s *GenerationLogStore) GetByID(id string) (*GenerationLog, error) {
	log := &GenerationLog{}
	query := `SELECT ` + genLogCols + ` FROM generation_logs WHERE id = $1 AND deleted_at IS NULL`

	if err := s.scanRow(log, s.db.QueryRow(query, id)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return log, nil
}

// GetByTaskID returns a log entry by its task ID.
func (s *GenerationLogStore) GetByTaskID(taskID string) (*GenerationLog, error) {
	log := &GenerationLog{}
	query := `SELECT ` + genLogCols + ` FROM generation_logs WHERE task_id = $1 AND deleted_at IS NULL`

	if err := s.scanRow(log, s.db.QueryRow(query, taskID)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return log, nil
}

// UpdateByTaskID updates a log entry by its task ID (used when async tasks complete).
func (s *GenerationLogStore) UpdateByTaskID(taskID, aiResponse, outputs, status, errorMessage string) error {
	query := `UPDATE generation_logs
		SET ai_response = $1, outputs = $2, status = $3, error_message = $4, updated_at = NOW()
		WHERE task_id = $5 AND deleted_at IS NULL`

	result, err := s.db.Exec(query,
		nullIfEmpty(aiResponse), nullIfEmpty(outputs), status, nullIfEmpty(errorMessage), taskID,
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

// List returns paginated generation logs, newest first.
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

	query := `SELECT ` + genLogCols + ` FROM generation_logs WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []GenerationLog
	for rows.Next() {
		var l GenerationLog
		if err := s.scanRow(&l, rows); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// nullIfEmpty returns nil if s is empty, otherwise the string.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
