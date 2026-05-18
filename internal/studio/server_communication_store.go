package studio

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ServerCommunication stores a trace of every request sent to an external AI API.
type ServerCommunication struct {
	ID           string     `json:"id"`
	TaskID       string     `json:"task_id"`
	ModelName    string     `json:"model_name"`
	Endpoint     string     `json:"endpoint"`
	Method       string     `json:"method"`
	RequestBody  string     `json:"request_body,omitempty"`
	ResponseBody string     `json:"response_body,omitempty"`
	StatusCode   int        `json:"status_code"`
	DurationMs   int64      `json:"duration_ms"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type ServerCommunicationStore struct {
	db *sql.DB
}

func NewServerCommunicationStore(db *sql.DB) *ServerCommunicationStore {
	return &ServerCommunicationStore{db: db}
}

const serverCommCols = `id, task_id, model_name, endpoint, method,
	COALESCE(request_body, '') AS request_body,
	COALESCE(response_body, '') AS response_body,
	status_code, duration_ms,
	COALESCE(error_message, '') AS error_message,
	created_at`

func (s *ServerCommunicationStore) Create(log *ServerCommunication) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	query := `INSERT INTO server_communications
		(id, task_id, model_name, endpoint, method, request_body, response_body, status_code, duration_ms, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		RETURNING created_at`
	return s.db.QueryRow(query, log.ID, log.TaskID, log.ModelName, log.Endpoint, log.Method,
		nullIfEmpty(log.RequestBody), nullIfEmpty(log.ResponseBody),
		log.StatusCode, log.DurationMs, nullIfEmpty(log.ErrorMessage)).
		Scan(&log.CreatedAt)
}

func (s *ServerCommunicationStore) GetByID(id string) (*ServerCommunication, error) {
	log := &ServerCommunication{}
	query := `SELECT ` + serverCommCols + ` FROM server_communications WHERE id = $1`
	err := s.db.QueryRow(query, id).Scan(
		&log.ID, &log.TaskID, &log.ModelName, &log.Endpoint, &log.Method,
		&log.RequestBody, &log.ResponseBody,
		&log.StatusCode, &log.DurationMs, &log.ErrorMessage, &log.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return log, nil
}

// ServerCommFilter holds optional filters for listing server communications.
type ServerCommFilter struct {
	TaskID    string
	ModelName string
	Page      int
	Limit     int
}

// ServerCommListResponse holds paginated results.
type ServerCommListResponse struct {
	Logs       []ServerCommunication `json:"logs"`
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
	TotalPages int                   `json:"total_pages"`
}

func (s *ServerCommunicationStore) List(filter ServerCommFilter) (*ServerCommListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}
	offset := (filter.Page - 1) * filter.Limit

	where := ""
	args := []interface{}{}
	argIdx := 1

	if filter.TaskID != "" {
		where += fmt.Sprintf(" WHERE task_id = $%d", argIdx)
		args = append(args, filter.TaskID)
		argIdx++
	}
	if filter.ModelName != "" {
		prefix := " WHERE"
		if where != "" {
			prefix = " AND"
		}
		where += fmt.Sprintf("%s model_name = $%d", prefix, argIdx)
		args = append(args, filter.ModelName)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM server_communications" + where
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	query := "SELECT " + serverCommCols + " FROM server_communications" + where +
		" ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", argIdx) +
		" OFFSET $" + fmt.Sprintf("%d", argIdx+1)
	args = append(args, filter.Limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []ServerCommunication
	for rows.Next() {
		var l ServerCommunication
		if err := rows.Scan(
			&l.ID, &l.TaskID, &l.ModelName, &l.Endpoint, &l.Method,
			&l.RequestBody, &l.ResponseBody,
			&l.StatusCode, &l.DurationMs, &l.ErrorMessage, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalPages := (total + filter.Limit - 1) / filter.Limit
	if totalPages < 1 {
		totalPages = 1
	}

	return &ServerCommListResponse{
		Logs:       logs,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}
