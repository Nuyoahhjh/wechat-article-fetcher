package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"wechat-article-fetcher/server/internal/auth"
	"wechat-article-fetcher/server/internal/domain"
	"wechat-article-fetcher/server/internal/store"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidPassword    = errors.New("current password is incorrect")
	ErrUserDisabled       = errors.New("user is disabled")
)

type AuthService struct {
	store  *store.Store
	tokens *auth.Manager
	logger *slog.Logger
}

type LoginResult struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresAt   time.Time   `json:"expires_at"`
	User        domain.User `json:"user"`
}

func NewAuthService(store *store.Store, tokens *auth.Manager, logger *slog.Logger) *AuthService {
	return &AuthService{store: store, tokens: tokens, logger: logger}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (LoginResult, error) {
	username = strings.TrimSpace(username)
	user, err := s.store.GetUserByUsername(ctx, username)
	if errors.Is(err, store.ErrNotFound) || (err == nil && !auth.CheckPassword(user.PasswordHash, password)) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}
	if user.Status != domain.UserActive {
		return LoginResult{}, ErrUserDisabled
	}
	token, expiresAt, err := s.tokens.Issue(user.ID, user.Role, user.DisplayName)
	if err != nil {
		return LoginResult{}, err
	}
	now := time.Now().UTC()
	if err := s.store.UpdateUserLastLogin(ctx, user.ID, now); err != nil {
		s.logger.Warn("failed to update last login", "user_id", user.ID, "error", err)
	}
	user.LastLoginAt = &now
	user.PasswordHash = ""
	return LoginResult{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        user,
	}, nil
}

func (s *AuthService) CurrentUser(ctx context.Context, userID string) (domain.User, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *AuthService) BootstrapAdmin(ctx context.Context, username, password string) error {
	count, err := s.store.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	user := domain.User{
		ID:           uuid.NewString(),
		Username:     strings.TrimSpace(username),
		PasswordHash: hash,
		DisplayName:  "Administrator",
		Role:         domain.RoleAdmin,
		Status:       domain.UserActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if user.Username == "" {
		user.Username = "admin"
	}
	if err := s.store.CreateUser(ctx, user); err != nil {
		return err
	}
	s.logger.Warn("bootstrap administrator created; change the password before deployment", "username", user.Username)
	return nil
}

func (s *AuthService) ChangePassword(
	ctx context.Context,
	userID, currentPassword, newPassword string,
) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !auth.CheckPassword(user.PasswordHash, currentPassword) {
		return ErrInvalidPassword
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	user.UpdatedAt = time.Now().UTC()
	return s.store.UpdateUser(ctx, user)
}
