package usecase

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"

	"developer.dev/web-3/internal/model"
)

const passwordIterations = 600000

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, 32)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(key), nil
}

func checkPassword(password, stored string) bool {
	saltHex, keyHex, _ := strings.Cut(stored, ":")
	salt, saltErr := hex.DecodeString(saltHex)
	expected, keyErr := hex.DecodeString(keyHex)
	valid := saltErr == nil && keyErr == nil && len(salt) == 16 && len(expected) == 32
	if !valid {
		salt, expected = make([]byte, 16), make([]byte, 32)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, 32)
	return err == nil && subtle.ConstantTimeCompare(key, expected) == 1 && valid
}

func tokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *Service) createUser(ctx context.Context, user model.User, password string) (model.User, error) {
	user.Name = strings.TrimSpace(user.Name)
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	if !length(user.Name, 2, 100) {
		return model.User{}, invalid("name", "Имя должно содержать от 2 до 100 символов")
	}
	address, err := mail.ParseAddress(user.Email)
	if err != nil || address.Address != user.Email || len(user.Email) > 254 {
		return model.User{}, invalid("email", "Укажите корректный email")
	}
	if !length(password, 8, 128) {
		return model.User{}, invalid("password", "Пароль должен содержать от 8 до 128 символов")
	}
	user.PasswordHash, err = hashPassword(password)
	if err != nil {
		return model.User{}, err
	}
	return s.repo.CreateUser(ctx, user)
}

func (s *Service) Register(ctx context.Context, user model.User, password string) (model.Session, error) {
	user.Role = model.RoleUser
	user, err := s.createUser(ctx, user, password)
	if err != nil {
		return model.Session{}, err
	}
	return s.newSession(ctx, user)
}

func (s *Service) Login(ctx context.Context, email, password string) (model.Session, error) {
	if len(password) > 512 {
		return model.Session{}, model.ErrCredentials
	}
	user, err := s.repo.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return model.Session{}, err
	}
	if !checkPassword(password, user.PasswordHash) || errors.Is(err, model.ErrNotFound) {
		return model.Session{}, model.ErrCredentials
	}
	return s.newSession(ctx, user)
}

func (s *Service) newSession(ctx context.Context, user model.User) (model.Session, error) {
	user.PasswordHash = ""
	session := model.Session{User: user, Token: rand.Text(), ExpiresAt: time.Now().Add(7 * 24 * time.Hour)}
	if err := s.repo.CreateSession(ctx, tokenHash(session.Token), user.ID, session.ExpiresAt); err != nil {
		return model.Session{}, err
	}
	return session, nil
}

func (s *Service) CurrentUser(ctx context.Context, token string) (model.User, error) {
	if token == "" {
		return model.User{}, model.ErrUnauthorized
	}
	session, err := s.repo.Session(ctx, tokenHash(token))
	if errors.Is(err, model.ErrNotFound) || err == nil && !session.ExpiresAt.After(time.Now()) {
		return model.User{}, model.ErrUnauthorized
	}
	return session.User, err
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repo.DeleteSession(ctx, tokenHash(token))
}

func (s *Service) EnsureAdmin(ctx context.Context, email, password string) error {
	user, err := s.repo.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err == nil {
		if user.Role != model.RoleAdmin {
			return errors.New("admin email is already used by a regular user")
		}
		return nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return err
	}
	_, err = s.createUser(ctx, model.User{Name: "Администратор", Email: email, Role: model.RoleAdmin}, password)
	return err
}
