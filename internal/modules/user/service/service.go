package service

import (
	"context"

	"github.com/PhantomX7/go-starter/internal/models"
	"github.com/PhantomX7/go-starter/internal/modules/user/dto"
	"github.com/PhantomX7/go-starter/internal/modules/user/repository"
	"github.com/PhantomX7/go-starter/pkg/pagination"
	"github.com/PhantomX7/go-starter/pkg/response"

	"github.com/jinzhu/copier"
)

type UserService interface {
	Index(ctx context.Context, req *pagination.Pagination) ([]*models.User, response.Meta, error)
	Create(ctx context.Context, req *dto.UserCreateRequest) (*models.User, error)
	Update(ctx context.Context, userId uint, req *dto.UserUpdateRequest) (*models.User, error)
	Delete(ctx context.Context, userId uint) error
	FindById(ctx context.Context, userId uint) (*models.User, error)
}

type userService struct {
	userRepository repository.UserRepository
}

func NewUserService(userRepository repository.UserRepository) UserService {
	return &userService{
		userRepository: userRepository,
	}
}

// Index implements UserService.
func (s *userService) Index(ctx context.Context, pg *pagination.Pagination) ([]*models.User, response.Meta, error) {
	users, err := s.userRepository.FindAll(ctx, pg)
	if err != nil {
		return nil, response.Meta{}, err
	}

	count, err := s.userRepository.Count(ctx, pg)
	if err != nil {
		return nil, response.Meta{}, err
	}

	return users, response.Meta{
		Total:  count,
		Offset: pg.Offset,
		Limit:  pg.Limit,
	}, nil
}

// Create implements UserService.
func (s *userService) Create(ctx context.Context, req *dto.UserCreateRequest) (*models.User, error) {
	var user *models.User

	err := copier.Copy(user, req)
	if err != nil {
		return user, err
	}

	err = s.userRepository.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

// Update implements UserService.
func (s *userService) Update(ctx context.Context, userId uint, req *dto.UserUpdateRequest) (*models.User, error) {
	user, err := s.userRepository.FindById(ctx, userId)
	if err != nil {
		return user, err
	}

	err = copier.Copy(&user, &req)
	if err != nil {
		return user, err
	}

	err = s.userRepository.Update(ctx, user)
	if err != nil {
		return user, err
	}

	return user, nil
}

// Delete implements UserService.
func (s *userService) Delete(ctx context.Context, userId uint) error {
	var user models.User

	user.ID = userId

	return s.userRepository.Delete(ctx, &user)
}

// FindById implements UserService.
func (s *userService) FindById(ctx context.Context, userId uint) (*models.User, error) {
	user, err := s.userRepository.FindById(ctx, userId)
	if err != nil {
		return user, err
	}
	return user, nil
}
