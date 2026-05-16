package project

import (
	"database/sql"
	"fmt"
)

// ─── Project Store ──────────────────────────────────────────────

type ProjectStore struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *ProjectStore {
	return &ProjectStore{db: db}
}

const projectCols = `id, name, COALESCE(description, '') AS description,
	COALESCE(metadata, '') AS metadata,
	created_at, updated_at, deleted_at`

func (s *ProjectStore) scanProject(p *Project, scanner interface{ Scan(dest ...interface{}) error }) error {
	return scanner.Scan(&p.ID, &p.Name, &p.Description, &p.Metadata, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
}

func (s *ProjectStore) Create(p *Project) error {
	query := `INSERT INTO projects (id, name, description, metadata)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, p.ID, p.Name, p.Description, nullIfEmpty(p.Metadata)).
		Scan(&p.CreatedAt, &p.UpdatedAt)
}

func (s *ProjectStore) GetByID(id string) (*Project, error) {
	p := &Project{}
	query := `SELECT ` + projectCols + ` FROM projects WHERE id = $1 AND deleted_at IS NULL`
	if err := s.scanProject(p, s.db.QueryRow(query, id)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (s *ProjectStore) List() ([]Project, error) {
	query := `SELECT ` + projectCols + ` FROM projects WHERE deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := s.scanProject(&p, rows); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (s *ProjectStore) Update(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	query := "UPDATE projects SET updated_at = NOW()"
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
		return fmt.Errorf("project not found")
	}
	return nil
}

func (s *ProjectStore) SoftDelete(id string) error {
	result, err := s.db.Exec(`UPDATE projects SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("project not found")
	}
	return nil
}

// ─── Scene Store ────────────────────────────────────────────────

const sceneCols = `id, project_id, number, COALESCE(name, '') AS name,
	COALESCE(description, '') AS description,
	created_at, updated_at, deleted_at`

func (s *ProjectStore) scanScene(sc *Scene, scanner interface{ Scan(dest ...interface{}) error }) error {
	return scanner.Scan(&sc.ID, &sc.ProjectID, &sc.Number, &sc.Name, &sc.Description, &sc.CreatedAt, &sc.UpdatedAt, &sc.DeletedAt)
}

func (s *ProjectStore) CreateScene(sc *Scene) error {
	query := `INSERT INTO scenes (id, project_id, number, name, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, sc.ID, sc.ProjectID, sc.Number, sc.Name, sc.Description).
		Scan(&sc.CreatedAt, &sc.UpdatedAt)
}

func (s *ProjectStore) GetSceneByID(id string) (*Scene, error) {
	sc := &Scene{}
	query := `SELECT ` + sceneCols + ` FROM scenes WHERE id = $1 AND deleted_at IS NULL`
	if err := s.scanScene(sc, s.db.QueryRow(query, id)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return sc, nil
}

func (s *ProjectStore) ListScenes(projectID string) ([]Scene, error) {
	query := `SELECT ` + sceneCols + ` FROM scenes WHERE project_id = $1 AND deleted_at IS NULL ORDER BY number ASC`
	rows, err := s.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scenes []Scene
	for rows.Next() {
		var sc Scene
		if err := s.scanScene(&sc, rows); err != nil {
			return nil, err
		}
		scenes = append(scenes, sc)
	}
	return scenes, rows.Err()
}

func (s *ProjectStore) UpdateScene(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	query := "UPDATE scenes SET updated_at = NOW()"
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
		return fmt.Errorf("scene not found")
	}
	return nil
}

func (s *ProjectStore) SoftDeleteScene(id string) error {
	result, err := s.db.Exec(`UPDATE scenes SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("scene not found")
	}
	return nil
}

// ─── Take Store ─────────────────────────────────────────────────

const takeCols = `id, scene_id, number, COALESCE(video_url, '') AS video_url,
	COALESCE(video_local_url, '') AS video_local_url,
	COALESCE(status, 'pending') AS status,
	created_at, updated_at, deleted_at`

func (s *ProjectStore) scanTake(t *Take, scanner interface{ Scan(dest ...interface{}) error }) error {
	return scanner.Scan(&t.ID, &t.SceneID, &t.Number, &t.VideoURL, &t.VideoLocalURL, &t.Status, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt)
}

func (s *ProjectStore) CreateTake(t *Take) error {
	query := `INSERT INTO takes (id, scene_id, number, video_url, video_local_url, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, t.ID, t.SceneID, t.Number, t.VideoURL, t.VideoLocalURL, t.Status).
		Scan(&t.CreatedAt, &t.UpdatedAt)
}

func (s *ProjectStore) GetTakeByID(id string) (*Take, error) {
	t := &Take{}
	query := `SELECT ` + takeCols + ` FROM takes WHERE id = $1 AND deleted_at IS NULL`
	if err := s.scanTake(t, s.db.QueryRow(query, id)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

func (s *ProjectStore) ListTakes(sceneID string) ([]Take, error) {
	query := `SELECT ` + takeCols + ` FROM takes WHERE scene_id = $1 AND deleted_at IS NULL ORDER BY number ASC`
	rows, err := s.db.Query(query, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var takes []Take
	for rows.Next() {
		var t Take
		if err := s.scanTake(&t, rows); err != nil {
			return nil, err
		}
		takes = append(takes, t)
	}
	return takes, rows.Err()
}

func (s *ProjectStore) UpdateTake(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	query := "UPDATE takes SET updated_at = NOW()"
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
		return fmt.Errorf("take not found")
	}
	return nil
}

func (s *ProjectStore) SoftDeleteTake(id string) error {
	result, err := s.db.Exec(`UPDATE takes SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("take not found")
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
