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
	COALESCE(metadata, '') AS metadata, active,
		(SELECT COUNT(*) FROM chapters WHERE project_id = projects.id AND deleted_at IS NULL) AS chapter_count,
	created_at, updated_at, deleted_at`

func (s *ProjectStore) scanProject(p *Project, scanner interface {
	Scan(dest ...interface{}) error
}) error {
	return scanner.Scan(&p.ID, &p.Name, &p.Description, &p.Metadata, &p.Active, &p.ChapterCount, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
}

func (s *ProjectStore) Create(p *Project) error {
	query := `INSERT INTO projects (id, name, description, metadata, active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, p.ID, p.Name, p.Description, nullIfEmpty(p.Metadata), p.Active).
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
	query := `SELECT ` + projectCols + ` FROM projects WHERE deleted_at IS NULL AND active = true ORDER BY created_at DESC`
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

// ListAll returns all non-deleted projects regardless of active status.
// Used by admin endpoints that need to see inactive projects too.
func (s *ProjectStore) ListAll() ([]Project, error) {
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

// ─── Chapter Store ──────────────────────────────────────────────

const chapterCols = `id, project_id, number, COALESCE(name, '') AS name,
	COALESCE(description, '') AS description, active,
		(SELECT COUNT(*) FROM scenes WHERE chapter_id IS NOT NULL AND chapter_id = chapters.id AND deleted_at IS NULL) AS scene_count,
	created_at, updated_at, deleted_at`

func (s *ProjectStore) scanChapter(c *Chapter, scanner interface {
	Scan(dest ...interface{}) error
}) error {
	return scanner.Scan(&c.ID, &c.ProjectID, &c.Number, &c.Name, &c.Description, &c.Active, &c.SceneCount, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
}

func (s *ProjectStore) CreateChapter(c *Chapter) error {
	query := `INSERT INTO chapters (id, project_id, number, name, description, active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, c.ID, c.ProjectID, c.Number, c.Name, c.Description, c.Active).
		Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (s *ProjectStore) GetChapterByID(id string) (*Chapter, error) {
	c := &Chapter{}
	query := `SELECT ` + chapterCols + ` FROM chapters WHERE id = $1 AND deleted_at IS NULL`
	if err := s.scanChapter(c, s.db.QueryRow(query, id)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return c, nil
}

func (s *ProjectStore) ListChapters(projectID string) ([]Chapter, error) {
	query := `SELECT ` + chapterCols + ` FROM chapters WHERE project_id = $1 AND deleted_at IS NULL ORDER BY number ASC`
	rows, err := s.db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chapters []Chapter
	for rows.Next() {
		var c Chapter
		if err := s.scanChapter(&c, rows); err != nil {
			return nil, err
		}
		chapters = append(chapters, c)
	}
	return chapters, rows.Err()
}

func (s *ProjectStore) UpdateChapter(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	query := "UPDATE chapters SET updated_at = NOW()"
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
		return fmt.Errorf("chapter not found")
	}
	return nil
}

func (s *ProjectStore) SoftDeleteChapter(id string) error {
	result, err := s.db.Exec(`UPDATE chapters SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("chapter not found")
	}
	return nil
}

// ─── Scene Store ────────────────────────────────────────────────

const sceneCols = `id, project_id, COALESCE(chapter_id::text, '') AS chapter_id, number,
	COALESCE(name, '') AS name, COALESCE(description, '') AS description, active,
		(SELECT COUNT(*) FROM shots WHERE scene_id IS NOT NULL AND scene_id = scenes.id AND deleted_at IS NULL) AS shot_count,
	created_at, updated_at, deleted_at`

func (s *ProjectStore) scanScene(sc *Scene, scanner interface {
	Scan(dest ...interface{}) error
}) error {
	return scanner.Scan(&sc.ID, &sc.ProjectID, &sc.ChapterID, &sc.Number, &sc.Name, &sc.Description, &sc.Active, &sc.ShotCount, &sc.CreatedAt, &sc.UpdatedAt, &sc.DeletedAt)
}

func (s *ProjectStore) CreateScene(sc *Scene) error {
	query := `INSERT INTO scenes (id, project_id, chapter_id, number, name, description, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, sc.ID, sc.ProjectID, sc.ChapterID, sc.Number, sc.Name, sc.Description, sc.Active).
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

func (s *ProjectStore) ListScenes(chapterID string) ([]Scene, error) {
	query := `SELECT ` + sceneCols + ` FROM scenes WHERE chapter_id = $1 AND deleted_at IS NULL ORDER BY number ASC`
	rows, err := s.db.Query(query, chapterID)
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

// ─── Shot Store ─────────────────────────────────────────────────

const shotCols = `id, scene_id, number, COALESCE(name, '') AS name,
	COALESCE(description, '') AS description, active,
		(SELECT COUNT(*) FROM takes WHERE shot_id IS NOT NULL AND shot_id = shots.id AND deleted_at IS NULL) AS take_count,
	created_at, updated_at, deleted_at`

func (s *ProjectStore) scanShot(sh *Shot, scanner interface {
	Scan(dest ...interface{}) error
}) error {
	return scanner.Scan(&sh.ID, &sh.SceneID, &sh.Number, &sh.Name, &sh.Description, &sh.Active, &sh.TakeCount, &sh.CreatedAt, &sh.UpdatedAt, &sh.DeletedAt)
}

func (s *ProjectStore) CreateShot(sh *Shot) error {
	query := `INSERT INTO shots (id, scene_id, number, name, description, active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, sh.ID, sh.SceneID, sh.Number, sh.Name, sh.Description, sh.Active).
		Scan(&sh.CreatedAt, &sh.UpdatedAt)
}

func (s *ProjectStore) GetShotByID(id string) (*Shot, error) {
	sh := &Shot{}
	query := `SELECT ` + shotCols + ` FROM shots WHERE id = $1 AND deleted_at IS NULL`
	if err := s.scanShot(sh, s.db.QueryRow(query, id)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return sh, nil
}

func (s *ProjectStore) ListShots(sceneID string) ([]Shot, error) {
	query := `SELECT ` + shotCols + ` FROM shots WHERE scene_id = $1 AND deleted_at IS NULL ORDER BY number ASC`
	rows, err := s.db.Query(query, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shots []Shot
	for rows.Next() {
		var sh Shot
		if err := s.scanShot(&sh, rows); err != nil {
			return nil, err
		}
		shots = append(shots, sh)
	}
	return shots, rows.Err()
}

func (s *ProjectStore) UpdateShot(id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	query := "UPDATE shots SET updated_at = NOW()"
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
		return fmt.Errorf("shot not found")
	}
	return nil
}

func (s *ProjectStore) SoftDeleteShot(id string) error {
	result, err := s.db.Exec(`UPDATE shots SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("shot not found")
	}
	return nil
}

// ─── Take Store ─────────────────────────────────────────────────

const takeCols = `t.id, t.scene_id, COALESCE(t.shot_id::text, '') AS shot_id, t.number, COALESCE(t.video_url, '') AS video_url,
	COALESCE(t.video_local_url, '') AS video_local_url,
	COALESCE(t.status, 'pending') AS status, t.active, t.final,
	COALESCE(t.task_id, '') AS task_id,
	t.created_at, t.updated_at, t.deleted_at`

const takeListCols = takeCols + `,
	COALESCE(gl.request_payload, '') AS request_payload`

const takeListFrom = `FROM takes t
	LEFT JOIN generation_logs gl ON (gl.task_id = t.task_id OR (t.task_id = '' AND gl.scene_id::text = t.scene_id::text AND gl.take_number = t.number)) AND gl.deleted_at IS NULL`

func (s *ProjectStore) scanTake(t *Take, scanner interface {
	Scan(dest ...interface{}) error
}) error {
	return scanner.Scan(&t.ID, &t.SceneID, &t.ShotID, &t.Number, &t.VideoURL, &t.VideoLocalURL, &t.Status, &t.Active, &t.Final, &t.TaskID, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt)
}

func (s *ProjectStore) scanTakeWithPayload(t *Take, scanner interface {
	Scan(dest ...interface{}) error
}) error {
	return scanner.Scan(&t.ID, &t.SceneID, &t.ShotID, &t.Number, &t.VideoURL, &t.VideoLocalURL, &t.Status, &t.Active, &t.Final, &t.TaskID, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt, &t.RequestPayload)
}

func (s *ProjectStore) CreateTake(t *Take) error {
	query := `INSERT INTO takes (id, scene_id, shot_id, number, video_url, video_local_url, status, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`
	return s.db.QueryRow(query, t.ID, t.SceneID, t.ShotID, t.Number, t.VideoURL, t.VideoLocalURL, t.Status, t.Active).
		Scan(&t.CreatedAt, &t.UpdatedAt)
}

func (s *ProjectStore) GetTakeByID(id string) (*Take, error) {
	t := &Take{}
	query := `SELECT ` + takeCols + ` ` + takeListFrom + ` AND t.id = $1`
	if err := s.scanTakeWithPayload(t, s.db.QueryRow(query, id)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

func (s *ProjectStore) ListTakes(shotID string) ([]Take, error) {
	query := `SELECT ` + takeListCols + ` ` + takeListFrom + ` WHERE t.shot_id = $1 AND t.deleted_at IS NULL ORDER BY t.number ASC, t.created_at DESC`
	rows, err := s.db.Query(query, shotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var takes []Take
	for rows.Next() {
		var t Take
		if err := s.scanTakeWithPayload(&t, rows); err != nil {
			return nil, err
		}
		takes = append(takes, t)
	}
	return takes, rows.Err()
}

// ListActiveTakes returns only active (non-discarded) takes for a shot,
// at most one per number due to the partial unique index.
func (s *ProjectStore) ListActiveTakes(shotID string) ([]Take, error) {
	query := `SELECT ` + takeListCols + ` ` + takeListFrom + ` WHERE t.shot_id = $1 AND t.deleted_at IS NULL AND t.active = true ORDER BY t.number ASC`
	rows, err := s.db.Query(query, shotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var takes []Take
	for rows.Next() {
		var t Take
		if err := s.scanTakeWithPayload(&t, rows); err != nil {
			return nil, err
		}
		takes = append(takes, t)
	}
	return takes, rows.Err()
}

// GetActiveTakeByNumber returns the active take for a shot+number pair,
// or nil if none exists.
func (s *ProjectStore) GetActiveTakeByNumber(shotID string, number int) (*Take, error) {
	t := &Take{}
	query := `SELECT ` + takeListCols + ` ` + takeListFrom + ` WHERE t.shot_id = $1 AND t.number = $2 AND t.deleted_at IS NULL AND t.active = true`
	if err := s.scanTakeWithPayload(t, s.db.QueryRow(query, shotID, number)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

// DeactivateTakesByNumber sets active=false on all active takes matching
// shot+number. Used before inserting a new generation for the same take slot.
func (s *ProjectStore) DeactivateTakesByNumber(shotID string, number int) error {
	_, err := s.db.Exec(
		`UPDATE takes SET active = false, updated_at = NOW() WHERE shot_id = $1 AND number = $2 AND deleted_at IS NULL AND active = true`,
		shotID, number,
	)
	return err
}

func (s *ProjectStore) DeactivateFinalsByNumber(shotID string, number int) error {
	_, err := s.db.Exec(
		`UPDATE takes SET final = false, finalized_at = NULL, updated_at = NOW() WHERE shot_id = $1 AND number = $2 AND deleted_at IS NULL AND final = true`,
		shotID, number,
	)
	return err
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

// GetTakeByVideoURL returns a take by shot_id and video_url, or nil if not found.
func (s *ProjectStore) GetTakeByVideoURL(shotID string, videoURL string) (*Take, error) {
	t := &Take{}
	query := `SELECT ` + takeListCols + ` ` + takeListFrom + ` AND t.shot_id = $1 AND t.video_url = $2 AND t.deleted_at IS NULL`
	if err := s.scanTakeWithPayload(t, s.db.QueryRow(query, shotID, videoURL)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

// GetPendingTakeByNumber returns the most recent pending take for a
// shot+number pair, or nil if none exists. Unlike GetActiveTakeByNumber
// it does NOT filter by active — it finds the pending take regardless of
// whether something else already deactivated it.
func (s *ProjectStore) GetPendingTakeByNumber(shotID string, number int) (*Take, error) {
	t := &Take{}
	query := `SELECT ` + takeListCols + ` ` + takeListFrom + ` AND t.shot_id = $1 AND t.number = $2 AND t.deleted_at IS NULL AND t.status = 'pending' ORDER BY t.created_at DESC LIMIT 1`
	if err := s.scanTakeWithPayload(t, s.db.QueryRow(query, shotID, number)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
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

// ─── Scene Assignment Store ─────────────────────────────────────

func (s *ProjectStore) GetScenePresets(sceneID string) ([]ScenePresetAssignment, error) {
	query := `SELECT sp.id, sp.scene_id, sp.preset_id, p.code, p.label, pg.slug AS group_slug, sp.created_at
			FROM scene_presets sp
			JOIN presets p ON p.id = sp.preset_id
			JOIN preset_groups pg ON pg.id = p.group_id
			WHERE sp.scene_id = $1
			ORDER BY pg.slug, p.code`
	rows, err := s.db.Query(query, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ScenePresetAssignment
	for rows.Next() {
		var a ScenePresetAssignment
		if err := rows.Scan(&a.ID, &a.SceneID, &a.PresetID, &a.Code, &a.Label, &a.GroupSlug, &a.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (s *ProjectStore) GetSceneCharacters(sceneID string) ([]SceneCharacterAssignment, error) {
	query := `SELECT sc.id, sc.scene_id, sc.character_id, c.name, sc.created_at
			FROM scene_characters sc
			JOIN characters c ON c.id = sc.character_id
			WHERE sc.scene_id = $1
			ORDER BY c.name`
	rows, err := s.db.Query(query, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SceneCharacterAssignment
	for rows.Next() {
		var a SceneCharacterAssignment
		if err := rows.Scan(&a.ID, &a.SceneID, &a.CharacterID, &a.Name, &a.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (s *ProjectStore) GetSceneAssets(sceneID string) ([]SceneAssetAssignment, error) {
	query := `SELECT sa.id, sa.scene_id, sa.file_id, f.filename, f.mime_type, sa.created_at
			FROM scene_assets sa
			JOIN files f ON f.id = sa.file_id
			WHERE sa.scene_id = $1
			ORDER BY f.filename`
	rows, err := s.db.Query(query, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []SceneAssetAssignment
	for rows.Next() {
		var a SceneAssetAssignment
		if err := rows.Scan(&a.ID, &a.SceneID, &a.FileID, &a.Filename, &a.MimeType, &a.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (s *ProjectStore) AssignPresetToScene(sceneID, presetID string) (string, error) {
	var id string
	query := `INSERT INTO scene_presets (id, scene_id, preset_id)
		VALUES (gen_random_uuid(), $1, $2)
		RETURNING id`
	err := s.db.QueryRow(query, sceneID, presetID).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *ProjectStore) AssignCharacterToScene(sceneID, characterID string) (string, error) {
	var id string
	query := `INSERT INTO scene_characters (id, scene_id, character_id)
		VALUES (gen_random_uuid(), $1, $2)
		RETURNING id`
	err := s.db.QueryRow(query, sceneID, characterID).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *ProjectStore) AssignAssetToScene(sceneID, fileID string) (string, error) {
	var id string
	query := `INSERT INTO scene_assets (id, scene_id, file_id)
		VALUES (gen_random_uuid(), $1, $2)
		RETURNING id`
	err := s.db.QueryRow(query, sceneID, fileID).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *ProjectStore) RemoveScenePreset(assignmentID string) error {
	result, err := s.db.Exec(`DELETE FROM scene_presets WHERE id = $1`, assignmentID)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("assignment not found")
	}
	return nil
}

func (s *ProjectStore) RemoveSceneCharacter(assignmentID string) error {
	result, err := s.db.Exec(`DELETE FROM scene_characters WHERE id = $1`, assignmentID)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("assignment not found")
	}
	return nil
}

func (s *ProjectStore) RemoveSceneAsset(assignmentID string) error {
	result, err := s.db.Exec(`DELETE FROM scene_assets WHERE id = $1`, assignmentID)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return fmt.Errorf("assignment not found")
	}
	return nil
}
