package text

import (
	"database/sql"
	"fmt"
	"time"
)

// ─── Types ─────────────────────────────────────────────────────

// ShotBuilderLog is one generate-shots / refine-shots call, whether it
// succeeded or failed. It stores everything needed to reconstruct the request:
// raw payload (incl. scene_context with the assigned resources), the final
// composed prompts, the user (denormalized from the JWT), model, skill, tokens
// and duration.
type ShotBuilderLog struct {
	ID                string     `json:"id"`
	Mode              string     `json:"mode"`
	UserID            int        `json:"user_id"`
	UserName          string     `json:"user_name"`
	UserEmail         string     `json:"user_email"`
	ProjectID         string     `json:"project_id"`
	SceneID           string     `json:"scene_id"`
	KeyModel          string     `json:"key_model"`
	APIModel          string     `json:"api_model"`
	SkillID           string     `json:"skill_id"`
	SkillName         string     `json:"skill_name"`
	RequestPayload    string     `json:"request_payload"` // raw JSON body as sent
	SystemPrompt      string     `json:"system_prompt"`   // final composed system prompt
	Prompt            string     `json:"prompt"`          // final composed user prompt (original script + context)
	Status            string     `json:"status"`
	ErrorMessage      string     `json:"error_message"`
	Response          string     `json:"response"` // extracted JSON from the last attempt
	Attempts          int        `json:"attempts"`
	TotalInputTokens  int        `json:"total_input_tokens"`
	TotalOutputTokens int        `json:"total_output_tokens"`
	DurationMs        int64      `json:"duration_ms"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at,omitempty"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
	// Enriched (LEFT JOIN, not stored)
	ProjectName string `json:"project_name"`
}

// ShotBuilderLogSummary is the light version used in list queries
// (avoids loading the heavy prompt/payload text columns).
type ShotBuilderLogSummary struct {
	ID                string    `json:"id"`
	UserID            int       `json:"user_id"`
	UserName          string    `json:"user_name"`
	UserEmail         string    `json:"user_email"`
	ProjectID         string    `json:"project_id"`
	ProjectName       string    `json:"project_name"`
	SceneID           string    `json:"scene_id"`
	Mode              string    `json:"mode"`
	KeyModel          string    `json:"key_model"`
	APIModel          string    `json:"api_model"`
	SkillID           string    `json:"skill_id"`
	SkillName         string    `json:"skill_name"`
	Status            string    `json:"status"`
	ErrorMessage      string    `json:"error_message"`
	Attempts          int       `json:"attempts"`
	TotalInputTokens  int       `json:"total_input_tokens"`
	TotalOutputTokens int       `json:"total_output_tokens"`
	DurationMs        int64     `json:"duration_ms"`
	CreatedAt         time.Time `json:"created_at"`
}

// ShotBuilderAttempt is one Claude API call within a generate-shots / refine-shots call.
type ShotBuilderAttempt struct {
	ID                  string    `json:"id"`
	LogID               string    `json:"log_id"`
	AttemptNumber       int       `json:"attempt_number"`
	Prompt              string    `json:"prompt"`   // prompt sent for THIS attempt (original or corrective)
	Response            string    `json:"response"` // raw Claude response
	Valid               bool      `json:"valid"`
	ErrorMessage        string    `json:"error_message"`
	InputTokens         int       `json:"input_tokens"`
	OutputTokens        int       `json:"output_tokens"`
	CacheReadTokens     int       `json:"cache_read_tokens"`
	CacheCreationTokens int       `json:"cache_creation_tokens"`
	DurationMs          int64     `json:"duration_ms"`
	CreatedAt           time.Time `json:"created_at"`
}

// ListShotBuilderLogsResponse is the paginated list payload (same shape as
// studio.ListGenerationLogsResponse).
type ListShotBuilderLogsResponse struct {
	Logs       []ShotBuilderLogSummary `json:"logs"`
	Total      int                     `json:"total"`
	Page       int                     `json:"page"`
	Limit      int                     `json:"limit"`
	TotalPages int                     `json:"total_pages"`
}

// ─── Store ─────────────────────────────────────────────────────

type LogStore struct {
	db *sql.DB
}

func NewLogStore(db *sql.DB) *LogStore {
	return &LogStore{db: db}
}

// Light columns for list queries.
const shotBuilderLogListCols = `sbl.id, sbl.user_id,
	COALESCE(sbl.user_name, '') AS user_name,
	COALESCE(sbl.user_email, '') AS user_email,
	COALESCE(sbl.project_id, '') AS project_id,
	COALESCE(p.name, '') AS project_name,
	COALESCE(sbl.scene_id, '') AS scene_id,
	COALESCE(sbl.mode, '') AS mode,
	COALESCE(sbl.key_model, '') AS key_model,
	COALESCE(sbl.api_model, '') AS api_model,
	COALESCE(sbl.skill_id, '') AS skill_id,
	COALESCE(sbl.skill_name, '') AS skill_name,
	sbl.status,
	COALESCE(sbl.error_message, '') AS error_message,
	COALESCE(sbl.attempts, 0) AS attempts,
	COALESCE(sbl.total_input_tokens, 0) AS total_input_tokens,
	COALESCE(sbl.total_output_tokens, 0) AS total_output_tokens,
	COALESCE(sbl.duration_ms, 0) AS duration_ms,
	sbl.created_at`

// Full columns for detail queries (includes the heavy text columns).
const shotBuilderLogFullCols = `sbl.id, sbl.user_id,
	COALESCE(sbl.user_name, '') AS user_name,
	COALESCE(sbl.user_email, '') AS user_email,
	COALESCE(sbl.project_id, '') AS project_id,
	COALESCE(p.name, '') AS project_name,
	COALESCE(sbl.scene_id, '') AS scene_id,
	COALESCE(sbl.mode, '') AS mode,
	COALESCE(sbl.key_model, '') AS key_model,
	COALESCE(sbl.api_model, '') AS api_model,
	COALESCE(sbl.skill_id, '') AS skill_id,
	COALESCE(sbl.skill_name, '') AS skill_name,
	COALESCE(sbl.request_payload, '') AS request_payload,
	COALESCE(sbl.system_prompt, '') AS system_prompt,
	COALESCE(sbl.prompt, '') AS prompt,
	sbl.status,
	COALESCE(sbl.error_message, '') AS error_message,
	COALESCE(sbl.response, '') AS response,
	COALESCE(sbl.attempts, 0) AS attempts,
	COALESCE(sbl.total_input_tokens, 0) AS total_input_tokens,
	COALESCE(sbl.total_output_tokens, 0) AS total_output_tokens,
	COALESCE(sbl.duration_ms, 0) AS duration_ms,
	sbl.created_at`

const shotBuilderLogFrom = `FROM shot_builder_logs sbl
	LEFT JOIN projects p ON p.id::text = sbl.project_id`

// Create inserts a new shot builder log and fills ID/timestamps.
func (s *LogStore) Create(log *ShotBuilderLog) error {
	mode := log.Mode
	if mode == "" {
		mode = "generate"
	}

	query := `INSERT INTO shot_builder_logs
		(mode, user_id, user_name, user_email, project_id, scene_id, key_model, api_model,
		 skill_id, skill_name, request_payload, system_prompt, prompt, status,
		 error_message, response, attempts, total_input_tokens, total_output_tokens, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING id, created_at, updated_at`

	err := s.db.QueryRow(query,
		mode, nullInt(log.UserID), nullIfEmpty(log.UserName), nullIfEmpty(log.UserEmail),
		nullIfEmpty(log.ProjectID), nullIfEmpty(log.SceneID),
		nullIfEmpty(log.KeyModel), nullIfEmpty(log.APIModel),
		nullIfEmpty(log.SkillID), nullIfEmpty(log.SkillName),
		nullIfEmpty(log.RequestPayload), nullIfEmpty(log.SystemPrompt), nullIfEmpty(log.Prompt),
		log.Status, nullIfEmpty(log.ErrorMessage), nullIfEmpty(log.Response),
		log.Attempts, log.TotalInputTokens, log.TotalOutputTokens, log.DurationMs,
	).Scan(&log.ID, &log.CreatedAt, &log.UpdatedAt)
	return err
}

// InsertAttempt inserts one Claude call attempt under a shot builder log.
func (s *LogStore) InsertAttempt(a *ShotBuilderAttempt) error {
	query := `INSERT INTO shot_builder_attempts
		(log_id, attempt_number, prompt, response, valid, error_message,
		 input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at`

	return s.db.QueryRow(query,
		a.LogID, a.AttemptNumber,
		nullIfEmpty(a.Prompt), nullIfEmpty(a.Response), a.Valid, nullIfEmpty(a.ErrorMessage),
		a.InputTokens, a.OutputTokens, a.CacheReadTokens, a.CacheCreationTokens, a.DurationMs,
	).Scan(&a.ID, &a.CreatedAt)
}

// ListLogs returns paginated shot builder logs, newest first.
// Empty filter values are ignored.
func (s *LogStore) ListLogs(page, limit int, projectID, sceneID, mode string, userID int, dateFrom, dateTo string) ([]ShotBuilderLogSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	where := "WHERE sbl.deleted_at IS NULL"
	args := []interface{}{}
	argIdx := 1

	if projectID != "" {
		where += fmt.Sprintf(" AND sbl.project_id = $%d", argIdx)
		args = append(args, projectID)
		argIdx++
	}
	if sceneID != "" {
		where += fmt.Sprintf(" AND sbl.scene_id = $%d", argIdx)
		args = append(args, sceneID)
		argIdx++
	}
	if mode != "" {
		where += fmt.Sprintf(" AND sbl.mode = $%d", argIdx)
		args = append(args, mode)
		argIdx++
	}
	if userID > 0 {
		where += fmt.Sprintf(" AND sbl.user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	if dateFrom != "" {
		where += fmt.Sprintf(" AND sbl.created_at >= $%d", argIdx)
		args = append(args, dateFrom)
		argIdx++
	}
	if dateTo != "" {
		where += fmt.Sprintf(" AND sbl.created_at <= $%d", argIdx)
		args = append(args, dateTo+"T23:59:59Z")
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM shot_builder_logs sbl " + where
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := "SELECT " + shotBuilderLogListCols + " " + shotBuilderLogFrom + " " + where +
		" ORDER BY sbl.created_at DESC LIMIT $" + fmt.Sprintf("%d", argIdx) +
		" OFFSET $" + fmt.Sprintf("%d", argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []ShotBuilderLogSummary
	for rows.Next() {
		var l ShotBuilderLogSummary
		if err := rows.Scan(
			&l.ID, &l.UserID, &l.UserName, &l.UserEmail, &l.ProjectID, &l.ProjectName,
			&l.SceneID, &l.Mode, &l.KeyModel, &l.APIModel, &l.SkillID, &l.SkillName,
			&l.Status, &l.ErrorMessage, &l.Attempts,
			&l.TotalInputTokens, &l.TotalOutputTokens, &l.DurationMs, &l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetLog returns a single shot builder log (full columns) with its attempts,
// or (nil, nil, nil) when the id does not exist.
func (s *LogStore) GetLog(id string) (*ShotBuilderLog, []ShotBuilderAttempt, error) {
	log := &ShotBuilderLog{}
	query := "SELECT " + shotBuilderLogFullCols + " " + shotBuilderLogFrom +
		" WHERE sbl.id = $1 AND sbl.deleted_at IS NULL"

	err := s.db.QueryRow(query, id).Scan(
		&log.ID, &log.UserID, &log.UserName, &log.UserEmail, &log.ProjectID, &log.ProjectName,
		&log.SceneID, &log.Mode, &log.KeyModel, &log.APIModel, &log.SkillID, &log.SkillName,
		&log.RequestPayload, &log.SystemPrompt, &log.Prompt,
		&log.Status, &log.ErrorMessage, &log.Response, &log.Attempts,
		&log.TotalInputTokens, &log.TotalOutputTokens, &log.DurationMs, &log.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	attempts, err := s.listAttempts(id)
	if err != nil {
		return nil, nil, err
	}
	return log, attempts, nil
}

func (s *LogStore) listAttempts(logID string) ([]ShotBuilderAttempt, error) {
	rows, err := s.db.Query(`SELECT id, log_id, attempt_number,
		COALESCE(prompt, '') AS prompt,
		COALESCE(response, '') AS response,
		COALESCE(valid, false) AS valid,
		COALESCE(error_message, '') AS error_message,
		COALESCE(input_tokens, 0) AS input_tokens,
		COALESCE(output_tokens, 0) AS output_tokens,
		COALESCE(cache_read_tokens, 0) AS cache_read_tokens,
		COALESCE(cache_creation_tokens, 0) AS cache_creation_tokens,
		COALESCE(duration_ms, 0) AS duration_ms,
		created_at
		FROM shot_builder_attempts WHERE log_id = $1 ORDER BY attempt_number ASC`, logID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []ShotBuilderAttempt
	for rows.Next() {
		var a ShotBuilderAttempt
		if err := rows.Scan(
			&a.ID, &a.LogID, &a.AttemptNumber, &a.Prompt, &a.Response, &a.Valid,
			&a.ErrorMessage, &a.InputTokens, &a.OutputTokens, &a.CacheReadTokens,
			&a.CacheCreationTokens, &a.DurationMs, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		attempts = append(attempts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return attempts, nil
}

// nullInt returns nil when v is 0, otherwise v (0 = "no user").
func nullInt(v int) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

// nullIfEmpty returns nil if s is empty, otherwise the string.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
