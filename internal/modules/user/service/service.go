package service

import (
	"context"

	"github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/internal/modules/user/repository"
)

type UserService interface {
	Create(ctx context.Context, user *models.User) error
	FindByToken(ctx context.Context, user *models.User, token string) error
}

type userService struct {
	userRepository repository.UserRepository
}

func NewUserService(userRepository repository.UserRepository) UserService {
	return &userService{
		userRepository: userRepository,
	}
}

func (s *userService) FindByToken(ctx context.Context, user *models.User, token string) error {
	return s.userRepository.FindByToken(ctx, user, token)
}

func (s *userService) Create(ctx context.Context, user *models.User) error {
	return s.userRepository.Create(ctx, user)
}
