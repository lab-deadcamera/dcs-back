package provider

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ─── Providers ──────────────────────────────────────────────────

func (s *Store) CreateProvider(p *Provider) error {
	query := `INSERT INTO providers (id, name)
		VALUES ($1, $2)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, p.ID, p.Name).Scan(&p.CreatedAt, &p.UpdatedAt)
}

func (s *Store) GetProviderByID(id string) (*Provider, error) {
	p := &Provider{}
	query := `SELECT id, name, active, created_at, updated_at, deleted_at
		FROM providers WHERE id = $1 AND deleted_at IS NULL`
	err := s.db.QueryRow(query, id).Scan(&p.ID, &p.Name, &p.Active, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (s *Store) ListProviders() ([]Provider, error) {
	rows, err := s.db.Query(`SELECT id, name, active, created_at, updated_at, deleted_at
		FROM providers WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []Provider
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.ID, &p.Name, &p.Active, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, nil
}

func (s *Store) UpdateProvider(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	query := "UPDATE providers SET updated_at = NOW()"
	args := []interface{}{}
	argIdx := 1

	for col, val := range updates {
		query += fmt.Sprintf(", %s = $%d", col, argIdx)
		args = append(args, val)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND deleted_at IS NULL", argIdx)
	args = append(args, id)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("provider not found")
	}
	return nil
}

func (s *Store) SoftDeleteProvider(id string) error {
	result, err := s.db.Exec(`UPDATE providers SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("provider not found")
	}
	return nil
}

// ─── Models ─────────────────────────────────────────────────────

func (s *Store) CreateModel(m *Model) error {
	if m.APIKey == "" {
		m.APIKey = "pending"
	}
	query := `INSERT INTO models (id, provider_id, name, api_key, url, endpoint)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, m.ID, m.ProviderID, m.Name, m.APIKey, m.URL, m.Endpoint).
		Scan(&m.CreatedAt, &m.UpdatedAt)
}

func (s *Store) GetModelByID(id string) (*Model, error) {
	m := &Model{}
	query := `SELECT id, provider_id, name, api_key, url, endpoint, active, created_at, updated_at, deleted_at
		FROM models WHERE id = $1 AND deleted_at IS NULL`
	err := s.db.QueryRow(query, id).Scan(&m.ID, &m.ProviderID, &m.Name, &m.APIKey, &m.URL, &m.Endpoint,
		&m.Active, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return m, nil
}

func (s *Store) ListModels() ([]ModelWithProvider, error) {
	rows, err := s.db.Query(`
		SELECT m.id, m.provider_id, m.name, m.api_key, m.url, m.endpoint, m.active,
		       m.created_at, m.updated_at, m.deleted_at, p.name AS provider_name
		FROM models m
		JOIN providers p ON p.id = m.provider_id
		WHERE m.deleted_at IS NULL
		ORDER BY m.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []ModelWithProvider
	for rows.Next() {
		var m ModelWithProvider
		if err := rows.Scan(&m.ID, &m.ProviderID, &m.Name, &m.APIKey, &m.URL, &m.Endpoint,
			&m.Active, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt, &m.ProviderName); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

func (s *Store) ListModelsByProvider(providerID string) ([]Model, error) {
	rows, err := s.db.Query(`
		SELECT id, provider_id, name, api_key, url, endpoint, active, created_at, updated_at, deleted_at
		FROM models WHERE provider_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []Model
	for rows.Next() {
		var m Model
		if err := rows.Scan(&m.ID, &m.ProviderID, &m.Name, &m.APIKey, &m.URL, &m.Endpoint,
			&m.Active, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

func (s *Store) ListModelsForProviders(providerIDs []string) (map[string][]Model, error) {
	result := make(map[string][]Model)
	if len(providerIDs) == 0 {
		return result, nil
	}

	args := make([]interface{}, len(providerIDs))
	placeholders := make([]string, len(providerIDs))
	for i, id := range providerIDs {
		args[i] = id
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(`
		SELECT id, provider_id, name, api_key, url, endpoint, active, created_at, updated_at, deleted_at
		FROM models WHERE provider_id IN (%s) AND deleted_at IS NULL
		ORDER BY created_at DESC`, strings.Join(placeholders, ","))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var m Model
		if err := rows.Scan(&m.ID, &m.ProviderID, &m.Name, &m.APIKey, &m.URL, &m.Endpoint,
			&m.Active, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt); err != nil {
			return nil, err
		}
		result[m.ProviderID] = append(result[m.ProviderID], m)
	}
	return result, nil
}

func (s *Store) UpdateModel(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	query := "UPDATE models SET updated_at = NOW()"
	args := []interface{}{}
	argIdx := 1

	for col, val := range updates {
		query += fmt.Sprintf(", %s = $%d", col, argIdx)
		args = append(args, val)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND deleted_at IS NULL", argIdx)
	args = append(args, id)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("model not found")
	}
	return nil
}

func (s *Store) SoftDeleteModel(id string) error {
	result, err := s.db.Exec(`UPDATE models SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("model not found")
	}
	return nil
}

// ─── Helpers ────────────────────────────────────────────────────

func (s *Store) CreateModelWithID(m *Model) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return s.CreateModel(m)
}
