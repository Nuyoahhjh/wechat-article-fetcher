package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"wechat-article-fetcher/server/internal/auth"
	"wechat-article-fetcher/server/internal/domain"
	"wechat-article-fetcher/server/internal/store"
)

type UserService struct {
	store *store.Store
}

type CreateUserInput struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type UpdateUserInput struct {
	DisplayName *string `json:"display_name"`
	Role        *string `json:"role"`
	Status      *string `json:"status"`
	Password    *string `json:"password"`
}

func NewUserService(store *store.Store) *UserService {
	return &UserService{store: store}
}

func (s *UserService) List(ctx context.Context, limit, offset int) ([]domain.User, error) {
	users, err := s.store.ListUsers(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	for i := range users {
		users[i].PasswordHash = ""
	}
	return users, nil
}

func (s *UserService) Get(ctx context.Context, id string) (domain.User, error) {
	user, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *UserService) Create(ctx context.Context, input CreateUserInput) (domain.User, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.Username == "" {
		return domain.User{}, errors.New("username is required")
	}
	if !domain.ValidUserRoles[input.Role] {
		return domain.User{}, errors.New("invalid role")
	}
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return domain.User{}, err
	}
	now := time.Now().UTC()
	user := domain.User{
		ID:           uuid.NewString(),
		Username:     input.Username,
		PasswordHash: hash,
		DisplayName:  input.DisplayName,
		Role:         input.Role,
		Status:       domain.UserActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}
	if err := s.store.CreateUser(ctx, user); err != nil {
		return domain.User{}, err
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *UserService) Update(ctx context.Context, id string, input UpdateUserInput) (domain.User, error) {
	user, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	if input.DisplayName != nil {
		user.DisplayName = strings.TrimSpace(*input.DisplayName)
	}
	if input.Role != nil {
		if !domain.ValidUserRoles[*input.Role] {
			return domain.User{}, errors.New("invalid role")
		}
		user.Role = *input.Role
	}
	if input.Status != nil {
		if *input.Status != domain.UserActive && *input.Status != domain.UserDisabled {
			return domain.User{}, errors.New("invalid status")
		}
		user.Status = *input.Status
	}
	if input.Password != nil {
		hash, err := auth.HashPassword(*input.Password)
		if err != nil {
			return domain.User{}, err
		}
		user.PasswordHash = hash
	}
	user.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateUser(ctx, user); err != nil {
		return domain.User{}, err
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *UserService) Delete(ctx context.Context, id, actorID string) error {
	if id == actorID {
		return errors.New("cannot delete the current user")
	}
	return s.store.DeleteUser(ctx, id)
}
