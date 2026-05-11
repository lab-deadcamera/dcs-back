package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists      = errors.New("username already exists")
	ErrInvalidCreds    = errors.New("invalid username or password")
)

type Service struct {
	store     *Store
	jwtSecret string
}

func NewService(store *Store, jwtSecret string) *Service {
	return &Service{
		store:     store,
		jwtSecret: jwtSecret,
	}
}

func (s *Service) Register(username, password, name, surname string) (*User, error) {
	existingUser, err := s.store.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.store.CreateUser(username, string(hashedPassword), name, surname)
}

func (s *Service) Login(username, password string) (string, error) {
	user, err := s.store.GetUserByUsername(username)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrInvalidCreds
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", ErrInvalidCreds
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // 24 hours
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
