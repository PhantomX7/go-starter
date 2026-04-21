package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	casbinv2 "github.com/casbin/casbin/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	adminrolerepository "github.com/PhantomX7/athleton/internal/modules/admin_role/repository"
	logrepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	refreshtokenrepository "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	userrepository "github.com/PhantomX7/athleton/internal/modules/user/repository"
	"github.com/PhantomX7/athleton/internal/modules/user/service"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"
)

type mockUserRepository struct {
	findAllFn        func(context.Context, *pagination.Pagination) ([]*models.User, error)
	countFn          func(context.Context, *pagination.Pagination) (int64, error)
	findByIDFn       func(context.Context, uint, ...repository.Association) (*models.User, error)
	updateFn         func(context.Context, *models.User) error
	findByUsernameFn func(context.Context, string) (*models.User, error)
	findByEmailFn    func(context.Context, string) (*models.User, error)
}

func (m *mockUserRepository) Create(context.Context, *models.User) error {
	panic("unexpected Create call")
}

func (m *mockUserRepository) Update(ctx context.Context, entity *models.User) error {
	if m.updateFn == nil {
		panic("unexpected Update call")
	}
	return m.updateFn(ctx, entity)
}

func (m *mockUserRepository) Delete(context.Context, *models.User) error {
	panic("unexpected Delete call")
}

func (m *mockUserRepository) FindByID(ctx context.Context, id uint, preloads ...repository.Association) (*models.User, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindByID call")
	}
	return m.findByIDFn(ctx, id, preloads...)
}

func (m *mockUserRepository) FindAll(ctx context.Context, pg *pagination.Pagination) ([]*models.User, error) {
	if m.findAllFn == nil {
		panic("unexpected FindAll call")
	}
	return m.findAllFn(ctx, pg)
}

func (m *mockUserRepository) Count(ctx context.Context, pg *pagination.Pagination) (int64, error) {
	if m.countFn == nil {
		panic("unexpected Count call")
	}
	return m.countFn(ctx, pg)
}

func (m *mockUserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	if m.findByUsernameFn == nil {
		panic("unexpected FindByUsername call")
	}
	return m.findByUsernameFn(ctx, username)
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.findByEmailFn == nil {
		panic("unexpected FindByEmail call")
	}
	return m.findByEmailFn(ctx, email)
}

var _ userrepository.UserRepository = (*mockUserRepository)(nil)

type mockAdminRoleRepository struct{}

func (m *mockAdminRoleRepository) Create(context.Context, *models.AdminRole) error {
	panic("unexpected Create call")
}
func (m *mockAdminRoleRepository) Update(context.Context, *models.AdminRole) error {
	panic("unexpected Update call")
}
func (m *mockAdminRoleRepository) Delete(context.Context, *models.AdminRole) error {
	panic("unexpected Delete call")
}
func (m *mockAdminRoleRepository) FindByID(context.Context, uint, ...repository.Association) (*models.AdminRole, error) {
	panic("unexpected FindByID call")
}
func (m *mockAdminRoleRepository) FindAll(context.Context, *pagination.Pagination) ([]*models.AdminRole, error) {
	panic("unexpected FindAll call")
}
func (m *mockAdminRoleRepository) Count(context.Context, *pagination.Pagination) (int64, error) {
	panic("unexpected Count call")
}
func (m *mockAdminRoleRepository) FindByName(context.Context, string) (*models.AdminRole, error) {
	panic("unexpected FindByName call")
}
func (m *mockAdminRoleRepository) CountUsersWithRole(context.Context, uint) (int64, error) {
	panic("unexpected CountUsersWithRole call")
}

var _ adminrolerepository.AdminRoleRepository = (*mockAdminRoleRepository)(nil)

type mockRefreshTokenRepository struct {
	revokeAllByUserIDFn func(context.Context, uint) error
}

func (m *mockRefreshTokenRepository) Create(context.Context, *models.RefreshToken) error {
	panic("unexpected Create call")
}
func (m *mockRefreshTokenRepository) Update(context.Context, *models.RefreshToken) error {
	panic("unexpected Update call")
}
func (m *mockRefreshTokenRepository) Delete(context.Context, *models.RefreshToken) error {
	panic("unexpected Delete call")
}
func (m *mockRefreshTokenRepository) FindByID(context.Context, uint, ...repository.Association) (*models.RefreshToken, error) {
	panic("unexpected FindByID call")
}
func (m *mockRefreshTokenRepository) FindAll(context.Context, *pagination.Pagination) ([]*models.RefreshToken, error) {
	panic("unexpected FindAll call")
}
func (m *mockRefreshTokenRepository) Count(context.Context, *pagination.Pagination) (int64, error) {
	panic("unexpected Count call")
}
func (m *mockRefreshTokenRepository) FindByToken(context.Context, string) (*models.RefreshToken, error) {
	panic("unexpected FindByToken call")
}
func (m *mockRefreshTokenRepository) FindActiveByID(context.Context, uuid.UUID) (*models.RefreshToken, error) {
	panic("unexpected FindActiveByID call")
}
func (m *mockRefreshTokenRepository) GetValidCountByUserID(context.Context, uint) (int64, error) {
	panic("unexpected GetValidCountByUserID call")
}
func (m *mockRefreshTokenRepository) DeleteInvalidToken(context.Context) error {
	panic("unexpected DeleteInvalidToken call")
}
func (m *mockRefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uint) error {
	if m.revokeAllByUserIDFn == nil {
		panic("unexpected RevokeAllByUserID call")
	}
	return m.revokeAllByUserIDFn(ctx, userID)
}
func (m *mockRefreshTokenRepository) RevokeAllByUserIDExcept(context.Context, uint, string) error {
	panic("unexpected RevokeAllByUserIDExcept call")
}
func (m *mockRefreshTokenRepository) RevokeByToken(context.Context, string) error {
	panic("unexpected RevokeByToken call")
}

var _ refreshtokenrepository.RefreshTokenRepository = (*mockRefreshTokenRepository)(nil)

type mockLogRepository struct {
	createFn func(context.Context, *models.Log) error
}

func (m *mockLogRepository) Create(ctx context.Context, entity *models.Log) error {
	if m.createFn == nil {
		panic("unexpected Create call")
	}
	return m.createFn(ctx, entity)
}
func (m *mockLogRepository) Update(context.Context, *models.Log) error {
	panic("unexpected Update call")
}
func (m *mockLogRepository) Delete(context.Context, *models.Log) error {
	panic("unexpected Delete call")
}
func (m *mockLogRepository) FindByID(context.Context, uint, ...repository.Association) (*models.Log, error) {
	panic("unexpected FindByID call")
}
func (m *mockLogRepository) FindAll(context.Context, *pagination.Pagination) ([]*models.Log, error) {
	panic("unexpected FindAll call")
}
func (m *mockLogRepository) Count(context.Context, *pagination.Pagination) (int64, error) {
	panic("unexpected Count call")
}

var _ logrepository.LogRepository = (*mockLogRepository)(nil)

type mockCasbinClient struct {
	getRolePermissionsFn func(uint) []string
}

func (m *mockCasbinClient) GetEnforcer() *casbinv2.Enforcer { return nil }
func (m *mockCasbinClient) AddRolePermissions(uint, []string) error {
	panic("unexpected AddRolePermissions call")
}
func (m *mockCasbinClient) RemoveRolePermissions(uint, []string) error {
	panic("unexpected RemoveRolePermissions call")
}
func (m *mockCasbinClient) SetRolePermissions(uint, []string) error {
	panic("unexpected SetRolePermissions call")
}
func (m *mockCasbinClient) GetRolePermissions(roleID uint) []string {
	if m.getRolePermissionsFn == nil {
		panic("unexpected GetRolePermissions call")
	}
	return m.getRolePermissionsFn(roleID)
}
func (m *mockCasbinClient) CheckPermission(uint, string) (bool, error) {
	panic("unexpected CheckPermission call")
}
func (m *mockCasbinClient) CheckPermissionWithRoot(string, *uint, string) (bool, error) {
	panic("unexpected CheckPermissionWithRoot call")
}
func (m *mockCasbinClient) DeleteRole(uint) error { panic("unexpected DeleteRole call") }

var _ casbin.Client = (*mockCasbinClient)(nil)

func setupLogger(t *testing.T) {
	t.Helper()

	prev := logger.Log
	logger.Log = zap.NewNop()
	t.Cleanup(func() {
		logger.Log = prev
	})
}

func TestUserServiceIndexReturnsUsersAndMeta(t *testing.T) {
	setupLogger(t)

	pg := pagination.NewPagination(map[string][]string{"limit": {"2"}}, nil, pagination.PaginationOptions{})
	repo := &mockUserRepository{
		findAllFn: func(ctx context.Context, gotPg *pagination.Pagination) ([]*models.User, error) {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return []*models.User{
				{ID: 1, Username: "alice"},
				{ID: 2, Username: "bob"},
			}, nil
		},
		countFn: func(ctx context.Context, gotPg *pagination.Pagination) (int64, error) {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return 2, nil
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{})
	ctx := utils.SetRequestIDToContext(context.Background(), "req-1")

	users, meta, err := svc.Index(ctx, pg)

	require.NoError(t, err)
	require.Len(t, users, 2)
	require.Equal(t, int64(2), meta.Total)
}

func TestUserServiceFindByIDHydratesAdminRolePermissions(t *testing.T) {
	setupLogger(t)

	roleID := uint(9)
	repo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, preloads ...repository.Association) (*models.User, error) {
			require.Equal(t, "req-2", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(7), id)
			require.NotEmpty(t, preloads)
			return &models.User{
				ID:       7,
				Username: "admin",
				Role:     models.UserRoleAdmin,
				AdminRole: &models.AdminRole{
					ID:   roleID,
					Name: "Manager",
				},
			}, nil
		},
	}
	casbinClient := &mockCasbinClient{
		getRolePermissionsFn: func(gotRoleID uint) []string {
			require.Equal(t, roleID, gotRoleID)
			return []string{permissions.UserRead.String()}
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, &mockLogRepository{}, casbinClient)
	ctx := utils.SetRequestIDToContext(context.Background(), "req-2")

	user, err := svc.FindByID(ctx, 7)

	require.NoError(t, err)
	require.NotNil(t, user.AdminRole)
	require.Equal(t, []string{permissions.UserRead.String()}, user.AdminRole.Permissions)
}

func TestUserServiceAssignAdminRoleRejectsRootUser(t *testing.T) {
	setupLogger(t)

	repo := &mockUserRepository{
		findByIDFn: func(context.Context, uint, ...repository.Association) (*models.User, error) {
			return &models.User{ID: 3, Role: models.UserRoleRoot, Name: "Root"}, nil
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{})

	user, err := svc.AssignAdminRole(context.Background(), 3, &dto.UserAssignAdminRoleRequest{AdminRoleID: 5})

	require.Nil(t, user)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrForbidden))
}

func TestUserServiceChangePasswordUpdatesHashRevokesTokensAndLogs(t *testing.T) {
	setupLogger(t)

	logCh := make(chan *models.Log, 1)
	current := &models.User{
		ID:       10,
		Name:     "Admin User",
		Username: "admin",
		Role:     models.UserRoleAdmin,
		Password: "old",
	}
	repo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.User, error) {
			require.Equal(t, "req-3", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(10), id)
			return current, nil
		},
		updateFn: func(ctx context.Context, entity *models.User) error {
			require.Equal(t, "req-3", utils.GetRequestIDFromContext(ctx))
			require.Same(t, current, entity)
			require.NotEqual(t, "old", entity.Password)
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(entity.Password), []byte("new-password")))
			return nil
		},
	}
	refreshRepo := &mockRefreshTokenRepository{
		revokeAllByUserIDFn: func(ctx context.Context, userID uint) error {
			require.Equal(t, "req-3", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(10), userID)
			return nil
		},
	}
	logRepo := &mockLogRepository{
		createFn: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, refreshRepo, logRepo, &mockCasbinClient{})
	ctx := utils.SetRequestIDToContext(context.Background(), "req-3")
	ctx = utils.NewContextWithValues(ctx, utils.ContextValues{UserID: 1, UserName: "Root"})

	err := svc.ChangePassword(ctx, 10, &dto.ChangeAdminPasswordRequest{NewPassword: "new-password"})

	require.NoError(t, err)
	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionChangePassword, entry.Action)
		require.Equal(t, models.LogEntityTypeUser, entry.EntityType)
		require.Equal(t, uint(10), entry.EntityID)
		require.Equal(t, "Root performed change_password on user: Admin User", entry.Message)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestUserServiceChangePasswordRejectsNonAdmin(t *testing.T) {
	setupLogger(t)

	repo := &mockUserRepository{
		findByIDFn: func(context.Context, uint, ...repository.Association) (*models.User, error) {
			return &models.User{ID: 4, Role: models.UserRoleUser}, nil
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{})

	err := svc.ChangePassword(context.Background(), 4, &dto.ChangeAdminPasswordRequest{NewPassword: "new-password"})

	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrInvalidInput))
}

func TestUserServiceIndexReturnsRepositoryError(t *testing.T) {
	setupLogger(t)

	expectedErr := errors.New("find all failed")
	repo := &mockUserRepository{
		findAllFn: func(context.Context, *pagination.Pagination) ([]*models.User, error) {
			return nil, expectedErr
		},
		countFn: func(context.Context, *pagination.Pagination) (int64, error) {
			t.Fatal("Count should not be called when FindAll fails")
			return 0, nil
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{})

	users, meta, err := svc.Index(context.Background(), pagination.NewPagination(nil, nil, pagination.PaginationOptions{}))

	require.Nil(t, users)
	require.Equal(t, response.Meta{}, meta)
	require.ErrorIs(t, err, expectedErr)
}
