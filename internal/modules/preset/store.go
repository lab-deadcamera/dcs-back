package preset

import (
	"database/sql"
	"fmt"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ─── Groups ─────────────────────────────────────────────────────

const groupCols = `id, name, slug, COALESCE(description, '') AS description, active, created_at, updated_at, deleted_at`

func (s *Store) ListGroups(includeInactive bool) ([]Group, error) {
	where := "WHERE deleted_at IS NULL"
	if !includeInactive {
		where += " AND active = true"
	}
	query := `SELECT ` + groupCols + ` FROM preset_groups ` + where + ` ORDER BY name ASC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Slug, &g.Description, &g.Active, &g.CreatedAt, &g.UpdatedAt, &g.DeletedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (s *Store) CreateGroup(g *Group) error {
	query := `INSERT INTO preset_groups (id, name, slug, description, active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, g.ID, g.Name, g.Slug, nullIfEmpty(g.Description), g.Active).
		Scan(&g.CreatedAt, &g.UpdatedAt)
}

func (s *Store) UpdateGroup(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	query := "UPDATE preset_groups SET updated_at = NOW()"
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
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("group not found")
	}
	return nil
}

// ─── Presets ────────────────────────────────────────────────────

const presetCols = `id, group_id, code, label, COALESCE(label_key, '') AS label_key, prompt, active, created_at, updated_at, deleted_at`

func (s *Store) ListPresets(groupID string, includeInactive bool) ([]Preset, error) {
	where := "WHERE deleted_at IS NULL"
	args := []interface{}{}
	argIdx := 1

	if groupID != "" {
		where += fmt.Sprintf(" AND group_id = $%d", argIdx)
		args = append(args, groupID)
		argIdx++
	}
	if !includeInactive {
		where += " AND active = true"
	}
	query := `SELECT ` + presetCols + ` FROM presets ` + where + ` ORDER BY created_at ASC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var presets []Preset
	for rows.Next() {
		var p Preset
		if err := rows.Scan(&p.ID, &p.GroupID, &p.Code, &p.Label, &p.LabelKey, &p.Prompt, &p.Active, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt); err != nil {
			return nil, err
		}
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

func (s *Store) GetPresetByID(id string) (*Preset, error) {
	p := &Preset{}
	query := `SELECT ` + presetCols + ` FROM presets WHERE id = $1 AND deleted_at IS NULL`
	err := s.db.QueryRow(query, id).Scan(&p.ID, &p.GroupID, &p.Code, &p.Label, &p.LabelKey, &p.Prompt, &p.Active, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (s *Store) CreatePreset(p *Preset) error {
	query := `INSERT INTO presets (id, group_id, code, label, label_key, prompt, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, p.ID, p.GroupID, p.Code, p.Label, p.LabelKey, p.Prompt, p.Active).
		Scan(&p.CreatedAt, &p.UpdatedAt)
}

func (s *Store) UpdatePreset(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	query := "UPDATE presets SET updated_at = NOW()"
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
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("preset not found")
	}
	return nil
}

func (s *Store) SoftDeletePreset(id string) error {
	result, err := s.db.Exec(`UPDATE presets SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("preset not found")
	}
	return nil
}

// ─── Helpers ────────────────────────────────────────────────────

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
