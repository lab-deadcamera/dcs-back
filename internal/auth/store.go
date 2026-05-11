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

func (s *Store) CreateUser(username, passwordHash, name, surname string) (*User, error) {
	query := `INSERT INTO users (username, password_hash, name, surname, active) VALUES ($1, $2, $3, $4, true) RETURNING id, active, created_at, updated_at`
	
	user := &User{
		Username:     username,
		PasswordHash: passwordHash,
		Name:         name,
		Surname:      surname,
	}

	err := s.db.QueryRow(query, username, passwordHash, name, surname).Scan(&user.ID, &user.Active, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Store) GetUserByUsername(username string) (*User, error) {
	query := `SELECT id, username, password_hash, name, surname, active, created_at, updated_at, deleted_at FROM users WHERE username = $1 AND active = true AND deleted_at IS NULL`
	
	user := &User{}
	err := s.db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Name, &user.Surname, &user.Active, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // User not found
		}
		return nil, err
	}

	return user, nil
}
