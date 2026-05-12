package character

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

func (s *Store) Create(ch *Character) error {
	query := `INSERT INTO characters (id, name, description, metadata)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, ch.ID, ch.Name, ch.Description, ch.Metadata).
		Scan(&ch.CreatedAt, &ch.UpdatedAt)
}

func (s *Store) GetByID(id string) (*Character, error) {
	ch := &Character{}
	query := `SELECT id, name, description, metadata, created_at, updated_at, deleted_at
		FROM characters WHERE id = $1 AND deleted_at IS NULL`
	err := s.db.QueryRow(query, id).Scan(&ch.ID, &ch.Name, &ch.Description, &ch.Metadata,
		&ch.CreatedAt, &ch.UpdatedAt, &ch.DeletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return ch, nil
}

func (s *Store) List() ([]Character, error) {
	rows, err := s.db.Query(`SELECT id, name, description, metadata, created_at, updated_at, deleted_at
		FROM characters WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chars []Character
	for rows.Next() {
		var ch Character
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Description, &ch.Metadata,
			&ch.CreatedAt, &ch.UpdatedAt, &ch.DeletedAt); err != nil {
			return nil, err
		}
		chars = append(chars, ch)
	}
	return chars, nil
}

func (s *Store) Update(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	query := "UPDATE characters SET updated_at = NOW()"
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
		return fmt.Errorf("character not found")
	}
	return nil
}

func (s *Store) SoftDelete(id string) error {
	result, err := s.db.Exec(`UPDATE characters SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("character not found")
	}
	return nil
}

// ─── Character-File relations ─────────────────────────────────

func (s *Store) AddFile(characterID, fileID, role string) error {
	if role == "" {
		role = "reference"
	}
	_, err := s.db.Exec(`INSERT INTO character_files (character_id, file_id, role)
		VALUES ($1, $2, $3) ON CONFLICT (character_id, file_id, role) DO NOTHING`,
		characterID, fileID, role)
	return err
}

func (s *Store) RemoveFile(characterID, fileID string) error {
	_, err := s.db.Exec(`DELETE FROM character_files WHERE character_id = $1 AND file_id = $2`,
		characterID, fileID)
	return err
}

func (s *Store) ListFiles(characterID string) ([]FileRef, error) {
	rows, err := s.db.Query(`SELECT file_id, role, created_at FROM character_files
		WHERE character_id = $1 ORDER BY created_at ASC`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []FileRef
	for rows.Next() {
		var r FileRef
		if err := rows.Scan(&r.FileID, &r.Role, &r.CreatedAt); err != nil {
			return nil, err
		}
		refs = append(refs, r)
	}
	return refs, nil
}
