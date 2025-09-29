package service

import (
	"github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/internal/modules/user/repository"
)

type UserService interface {
	Create(user *models.User) error
	FindByToken(user *models.User, token string) error
}

type userService struct {
	userRepository repository.UserRepository
}

func NewUserService(userRepository repository.UserRepository) UserService {
	return &userService{
		userRepository: userRepository,
	}
}

func (s *userService) FindByToken(user *models.User, token string) error {
	return s.userRepository.FindByToken(user, token)
}

func (s *userService) Create(user *models.User) error {
	return s.userRepository.Create(user)
}
