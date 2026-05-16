package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists      = errors.New("username already exists")
	ErrInvalidCreds    = errors.New("invalid username or password")
	ErrInsufficientRole = errors.New("insufficient role level")
	ErrCannotCreateAdmin = errors.New("only super admin can create admins")
)

type Service struct {
	store     *Store
	jwtSecret string

	// Super admin seed credentials
	superAdminUsername string
	superAdminPassword string
	superAdminName     string
	superAdminSurname  string
	superAdminUserName string
	superAdminEmail    string
}

func NewService(store *Store, jwtSecret string) *Service {
	return &Service{store: store, jwtSecret: jwtSecret}
}

// SetSuperAdminConfig configures the super admin seed credentials.
func (s *Service) SetSuperAdminConfig(username, password, name, surname, userName, email string) {
	s.superAdminUsername = username
	s.superAdminPassword = password
	s.superAdminName = name
	s.superAdminSurname = surname
	s.superAdminUserName = userName
	s.superAdminEmail = email
}

// SeedSuperAdmin garantiza que el usuario super admin exista y esté activo
// al arrancar la aplicación. Las credenciales se cargan desde .env o
// variables de entorno. Maneja tres escenarios:
//  1. El usuario no existe → lo crea.
//  2. El usuario existe pero está inactivo/borrado → lo reactiva y actualiza credenciales.
//  3. El usuario existe y está activo → no-op (no sobreescribe credenciales existentes).
func (s *Service) SeedSuperAdmin() error {
	if s.superAdminUsername == "" || s.superAdminPassword == "" {
		return errors.New("SUPER_ADMIN_USERNAME and SUPER_ADMIN_PASSWORD must be set")
	}

	superRole, err := s.store.GetRoleByLevel(0)
	if err != nil {
		return fmt.Errorf("getting super admin role: %w", err)
	}
	if superRole == nil {
		return errors.New("SUPER_ADMIN role not found — run migrations first")
	}

	// Buscar el usuario sin filtros de soft-delete
	existing, err := s.store.GetUserByUsernameAll(s.superAdminUsername)
	if err != nil {
		return fmt.Errorf("checking super admin: %w", err)
	}
	if existing != nil && existing.Active && existing.DeletedAt == nil {
		return nil // ya existe y está activo
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(s.superAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing super admin password: %w", err)
	}

	name := s.superAdminName
	if name == "" {
		name = "Super"
	}
	surname := s.superAdminSurname
	if surname == "" {
		surname = "Admin"
	}
	userName := s.superAdminUserName
	if userName == "" {
		userName = "superadmin"
	}
	email := s.superAdminEmail
	if email == "" {
		email = "superadmin@deadcamera.studio"
	}

	if existing != nil {
		// Existe pero está inactivo/borrado → reactivar y actualizar credenciales
		return s.store.ReactivateUser(existing.ID, string(hash), name, surname, userName, email, superRole.ID)
	}

	// No existe → crearlo
	_, err = s.store.CreateUser(s.superAdminUsername, string(hash), name, surname, userName, email, superRole.ID)
	return err
}

// ─── Auth ──────────────────────────────────────────────────────────

func (s *Service) Register(req *RegisterRequest) (*UserResponse, error) {
	existing, err := s.store.GetUserByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// New registrations always get USER role (level=3)
	userRole, err := s.store.GetRoleByLevel(3)
	if err != nil {
		return nil, err
	}

	userName := req.UserName
	if userName == "" {
		userName = req.Username
	}

	user, err := s.store.CreateUser(req.Username, string(hash), req.Name, req.Surname, userName, req.Email, userRole.ID)
	if err != nil {
		return nil, err
	}

	return s.userToResponse(user)
}

func (s *Service) Login(username, password string) (*TokenResponse, error) {
	user, err := s.store.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCreds
	}

	role, err := s.store.GetRoleByID(user.RoleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("user role not found")
	}

	// Generate JWT with role info
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":       user.ID,
		"username":  user.Username,
		"name":      user.Name,
		"surname":   user.Surname,
		"user_name": user.UserName,
		"email":     user.Email,
		"role_id":   user.RoleID,
		"role_name": role.Name,
		"role_level": role.Level,
		"iat":       now.Unix(),
		"exp":       now.Add(24 * time.Hour).Unix(),
	}

	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	userResp, err := s.userToResponse(user)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{Token: tokenStr, User: userResp}, nil
}

// ─── Admin: user management ───────────────────────────────────────

// CreateUser creates a user with a specific role.
// CallerLevel is the level of the user making the request.
func (s *Service) CreateUser(callerLevel int, req *CreateUserRequest) (*UserResponse, error) {
	targetRole, err := s.store.GetRoleByID(req.RoleID)
	if err != nil {
		return nil, err
	}
	if targetRole == nil {
		return nil, errors.New("role not found")
	}

	// Only SUPER_ADMIN (level 0) can create ADMIN (level 1)
	if targetRole.Level == 1 && callerLevel > 0 {
		return nil, ErrCannotCreateAdmin
	}

	// ADMIN (level 1) can only create roles with level >= 2
	if callerLevel == 1 && targetRole.Level < 2 {
		return nil, ErrInsufficientRole
	}

	// User must not try to create someone with higher privileges than themselves
	if targetRole.Level < callerLevel {
		return nil, ErrInsufficientRole
	}

	existing, err := s.store.GetUserByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	userName := req.UserName
	if userName == "" {
		userName = req.Username
	}

	user, err := s.store.CreateUser(req.Username, string(hash), req.Name, req.Surname, userName, req.Email, targetRole.ID)
	if err != nil {
		return nil, err
	}

	return s.userToResponse(user)
}

func (s *Service) ListUsers() ([]UserResponse, error) {
	users, err := s.store.ListUsers()
	if err != nil {
		return nil, err
	}

	var resp []UserResponse
	for _, u := range users {
		r, err := s.userToResponse(&u)
		if err != nil {
			return nil, err
		}
		resp = append(resp, *r)
	}
	return resp, nil
}

func (s *Service) GetUserProfile(userID int64) (*UserResponse, error) {
	user, err := s.store.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	return s.userToResponse(user)
}

// ─── Helpers ───────────────────────────────────────────────────────

func (s *Service) userToResponse(u *User) (*UserResponse, error) {
	role, err := s.store.GetRoleByID(u.RoleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("user role not found")
	}
	return &UserResponse{
		ID:       u.ID,
		Username: u.Username,
		Name:     u.Name,
		Surname:  u.Surname,
		UserName: u.UserName,
		Email:    u.Email,
		Role:     *role,
		Active:   u.Active,
	}, nil
}
