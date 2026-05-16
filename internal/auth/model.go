package auth

import "time"

// ─── Role ─────────────────────────────────────────────────────────

// Role represents a permission level.
// Lower Level = more privileges.
type Role struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// ─── User ──────────────────────────────────────────────────────────

type User struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	Name         string     `json:"name"`
	Surname      string     `json:"surname"`
	UserName     string     `json:"user_name"`
	Email        string     `json:"email"`
	RoleID       int        `json:"role_id"`
	Active       bool       `json:"active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// UserResponse is the public-safe user representation (no password).
type UserResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	Role     Role   `json:"role"`
	Active   bool   `json:"active"`
}

// ─── Requests / Responses ──────────────────────────────────────────

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
	Surname  string `json:"surname" binding:"required"`
	UserName string `json:"user_name"`
	Email    string `json:"email"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type TokenResponse struct {
	Token string       `json:"token"`
	User  *UserResponse `json:"user"`
}

// CreateUserRequest is used by admins to create users with a specific role.
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
	Surname  string `json:"surname" binding:"required"`
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	RoleID   int    `json:"role_id" binding:"required"`
}
