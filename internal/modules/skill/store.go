package skill

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

func (s *Store) Create(skill *Skill) error {
	query := `INSERT INTO skills (id, name, description, system_prompt)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, skill.ID, skill.Name, skill.Description, skill.SystemPrompt).
		Scan(&skill.CreatedAt, &skill.UpdatedAt)
}

func (s *Store) GetByID(id string) (*Skill, error) {
	skill := &Skill{}
	query := `SELECT id, name, description, system_prompt, created_at, updated_at, deleted_at
		FROM skills WHERE id = $1 AND deleted_at IS NULL`
	err := s.db.QueryRow(query, id).Scan(
		&skill.ID, &skill.Name, &skill.Description, &skill.SystemPrompt,
		&skill.CreatedAt, &skill.UpdatedAt, &skill.DeletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return skill, nil
}

func (s *Store) List() ([]Skill, error) {
	rows, err := s.db.Query(`SELECT id, name, description, system_prompt, created_at, updated_at, deleted_at
		FROM skills WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var sk Skill
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.Description, &sk.SystemPrompt,
			&sk.CreatedAt, &sk.UpdatedAt, &sk.DeletedAt); err != nil {
			return nil, err
		}
		skills = append(skills, sk)
	}
	return skills, nil
}

func (s *Store) Update(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	query := "UPDATE skills SET updated_at = NOW()"
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
		return fmt.Errorf("skill not found")
	}
	return nil
}

func (s *Store) SoftDelete(id string) error {
	result, err := s.db.Exec(`UPDATE skills SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("skill not found")
	}
	return nil
}
