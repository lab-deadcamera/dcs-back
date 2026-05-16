package auth

import (
	"database/sql"
	"errors"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ─── Roles ─────────────────────────────────────────────────────────

func (s *Store) GetRoleByID(id int) (*Role, error) {
	r := &Role{}
	err := s.db.QueryRow(`SELECT id, name, level FROM roles WHERE id = $1`, id).
		Scan(&r.ID, &r.Name, &r.Level)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r, nil
}

func (s *Store) GetRoleByLevel(level int) (*Role, error) {
	r := &Role{}
	err := s.db.QueryRow(`SELECT id, name, level FROM roles WHERE level = $1`, level).
		Scan(&r.ID, &r.Name, &r.Level)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r, nil
}

func (s *Store) ListRoles() ([]Role, error) {
	rows, err := s.db.Query(`SELECT id, name, level FROM roles ORDER BY level ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Level); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, nil
}

// ─── Users ─────────────────────────────────────────────────────────

const userCols = `id, username, name, surname, COALESCE(user_name, '') AS user_name,
	COALESCE(email, '') AS email, role_id, active, created_at, updated_at, deleted_at`

func (s *Store) scanUser(u *User, scanner interface{ Scan(dest ...interface{}) error }) error {
	return scanner.Scan(&u.ID, &u.Username, &u.Name, &u.Surname, &u.UserName, &u.Email, &u.RoleID, &u.Active, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
}

func (s *Store) CreateUser(username, passwordHash, name, surname, userName, email string, roleID int) (*User, error) {
	query := `INSERT INTO users (username, password_hash, name, surname, user_name, email, role_id, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true)
		RETURNING ` + userCols

	u := &User{Username: username, PasswordHash: passwordHash, Name: name, Surname: surname, UserName: userName, Email: email, RoleID: roleID}
	err := s.scanUser(u, s.db.QueryRow(query, username, passwordHash, name, surname, userName, email, roleID))
	if err != nil {
		return nil, err
	}
	return u, nil
}

// GetUserByUsernameAll busca un usuario por username IGNORANDO los filtros
// de soft-delete (active / deleted_at). Se usa exclusivamente para el seed
// del super admin, donde necesitamos detectar usuarios incluso si fueron
// desactivados para poder reactivarlos.
func (s *Store) GetUserByUsernameAll(username string) (*User, error) {
	query := `SELECT ` + userCols + `, password_hash FROM users WHERE username = $1`

	u := &User{}
	err := s.db.QueryRow(query, username).
		Scan(&u.ID, &u.Username, &u.Name, &u.Surname, &u.UserName, &u.Email, &u.RoleID, &u.Active, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

// ReactivateUser reactiva un usuario (active=true, deleted_at=NULL)
// y actualiza sus credenciales con los valores proporcionados.
func (s *Store) ReactivateUser(id int64, passwordHash, name, surname, userName, email string, roleID int) error {
	query := `UPDATE users SET password_hash = $1, name = $2, surname = $3, user_name = $4, email = $5, role_id = $6, active = true, deleted_at = NULL, updated_at = NOW() WHERE id = $7`
	_, err := s.db.Exec(query, passwordHash, name, surname, userName, email, roleID, id)
	return err
}

func (s *Store) GetUserByUsername(username string) (*User, error) {
	query := `SELECT ` + userCols + `, password_hash FROM users WHERE username = $1 AND active = true AND deleted_at IS NULL`

	u := &User{}
	err := s.db.QueryRow(query, username).
		Scan(&u.ID, &u.Username, &u.Name, &u.Surname, &u.UserName, &u.Email, &u.RoleID, &u.Active, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func (s *Store) GetUserByID(id int64) (*User, error) {
	query := `SELECT ` + userCols + ` FROM users WHERE id = $1 AND deleted_at IS NULL`

	u := &User{}
	err := s.scanUser(u, s.db.QueryRow(query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT ` + userCols + ` FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := s.scanUser(&u, rows); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *Store) UpdateUserRole(id int64, roleID int) error {
	result, err := s.db.Exec(`UPDATE users SET role_id = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`, roleID, id)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return errors.New("user not found")
	}
	return nil
}
