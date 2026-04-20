package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	adminrolerepository "github.com/PhantomX7/athleton/internal/modules/admin_role/repository"
	"github.com/PhantomX7/athleton/internal/modules/admin_role/service"
	logrepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"
	casbinv2 "github.com/casbin/casbin/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockAdminRoleRepository struct {
	findAllFn            func(context.Context, *pagination.Pagination) ([]*models.AdminRole, error)
	countFn              func(context.Context, *pagination.Pagination) (int64, error)
	createFn             func(context.Context, *models.AdminRole) error
	findByIDFn           func(context.Context, uint, ...repository.Association) (*models.AdminRole, error)
	updateFn             func(context.Context, *models.AdminRole) error
	deleteFn             func(context.Context, *models.AdminRole) error
	findByNameFn         func(context.Context, string) (*models.AdminRole, error)
	countUsersWithRoleFn func(context.Context, uint) (int64, error)
}

func (m *mockAdminRoleRepository) Create(ctx context.Context, entity *models.AdminRole) error {
	if m.createFn == nil {
		panic("unexpected Create call")
	}
	return m.createFn(ctx, entity)
}

func (m *mockAdminRoleRepository) Update(ctx context.Context, entity *models.AdminRole) error {
	if m.updateFn == nil {
		panic("unexpected Update call")
	}
	return m.updateFn(ctx, entity)
}

func (m *mockAdminRoleRepository) Delete(ctx context.Context, entity *models.AdminRole) error {
	if m.deleteFn == nil {
		panic("unexpected Delete call")
	}
	return m.deleteFn(ctx, entity)
}

func (m *mockAdminRoleRepository) FindById(ctx context.Context, id uint, preloads ...repository.Association) (*models.AdminRole, error) {
	if m.findByIDFn == nil {
		panic("unexpected FindById call")
	}
	return m.findByIDFn(ctx, id, preloads...)
}

func (m *mockAdminRoleRepository) FindAll(ctx context.Context, pg *pagination.Pagination) ([]*models.AdminRole, error) {
	if m.findAllFn == nil {
		panic("unexpected FindAll call")
	}
	return m.findAllFn(ctx, pg)
}

func (m *mockAdminRoleRepository) Count(ctx context.Context, pg *pagination.Pagination) (int64, error) {
	if m.countFn == nil {
		panic("unexpected Count call")
	}
	return m.countFn(ctx, pg)
}

func (m *mockAdminRoleRepository) FindByName(ctx context.Context, name string) (*models.AdminRole, error) {
	if m.findByNameFn == nil {
		panic("unexpected FindByName call")
	}
	return m.findByNameFn(ctx, name)
}

func (m *mockAdminRoleRepository) CountUsersWithRole(ctx context.Context, roleID uint) (int64, error) {
	if m.countUsersWithRoleFn == nil {
		panic("unexpected CountUsersWithRole call")
	}
	return m.countUsersWithRoleFn(ctx, roleID)
}

var _ adminrolerepository.AdminRoleRepository = (*mockAdminRoleRepository)(nil)

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

func (m *mockLogRepository) FindById(context.Context, uint, ...repository.Association) (*models.Log, error) {
	panic("unexpected FindById call")
}

func (m *mockLogRepository) FindAll(context.Context, *pagination.Pagination) ([]*models.Log, error) {
	panic("unexpected FindAll call")
}

func (m *mockLogRepository) Count(context.Context, *pagination.Pagination) (int64, error) {
	panic("unexpected Count call")
}

var _ logrepository.LogRepository = (*mockLogRepository)(nil)

type mockCasbinClient struct {
	addRolePermissionsFn func(uint, []string) error
	setRolePermissionsFn func(uint, []string) error
	getRolePermissionsFn func(uint) []string
	deleteRoleFn         func(uint) error
}

func (m *mockCasbinClient) GetEnforcer() *casbinv2.Enforcer { return nil }

func (m *mockCasbinClient) AddRolePermissions(roleID uint, perms []string) error {
	if m.addRolePermissionsFn == nil {
		panic("unexpected AddRolePermissions call")
	}
	return m.addRolePermissionsFn(roleID, perms)
}

func (m *mockCasbinClient) RemoveRolePermissions(uint, []string) error {
	panic("unexpected RemoveRolePermissions call")
}

func (m *mockCasbinClient) SetRolePermissions(roleID uint, perms []string) error {
	if m.setRolePermissionsFn == nil {
		panic("unexpected SetRolePermissions call")
	}
	return m.setRolePermissionsFn(roleID, perms)
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

func (m *mockCasbinClient) DeleteRole(roleID uint) error {
	if m.deleteRoleFn == nil {
		panic("unexpected DeleteRole call")
	}
	return m.deleteRoleFn(roleID)
}

var _ casbin.Client = (*mockCasbinClient)(nil)

func setupLogger(t *testing.T) {
	t.Helper()

	prev := logger.Log
	logger.Log = zap.NewNop()
	t.Cleanup(func() {
		logger.Log = prev
	})
}

func TestAdminRoleServiceIndexReturnsRolesWithPermissions(t *testing.T) {
	setupLogger(t)

	pg := pagination.NewPagination(map[string][]string{"limit": {"2"}}, nil, pagination.PaginationOptions{})
	repo := &mockAdminRoleRepository{
		findAllFn: func(ctx context.Context, gotPg *pagination.Pagination) ([]*models.AdminRole, error) {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return []*models.AdminRole{
				{ID: 1, Name: "Manager", IsActive: true},
				{ID: 2, Name: "Support", IsActive: true},
			}, nil
		},
		countFn: func(ctx context.Context, gotPg *pagination.Pagination) (int64, error) {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return 2, nil
		},
	}
	casbinClient := &mockCasbinClient{
		getRolePermissionsFn: func(roleID uint) []string {
			if roleID == 1 {
				return []string{permissions.ProductRead.String()}
			}
			return []string{permissions.UserRead.String()}
		},
	}

	svc := service.NewAdminRoleService(repo, &mockLogRepository{}, casbinClient)
	ctx := utils.SetRequestIDToContext(context.Background(), "req-1")

	roles, meta, err := svc.Index(ctx, pg)

	require.NoError(t, err)
	require.Len(t, roles, 2)
	require.Equal(t, []string{permissions.ProductRead.String()}, roles[0].Permissions)
	require.Equal(t, []string{permissions.UserRead.String()}, roles[1].Permissions)
	require.Equal(t, int64(2), meta.Total)
}

func TestAdminRoleServiceCreateRejectsInvalidPermissions(t *testing.T) {
	setupLogger(t)

	svc := service.NewAdminRoleService(&mockAdminRoleRepository{}, &mockLogRepository{}, &mockCasbinClient{})

	role, err := svc.Create(context.Background(), &dto.CreateAdminRoleRequest{
		Name:        "Manager",
		Description: "test",
		Permissions: []string{"not:valid"},
	})

	require.Nil(t, role)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrInvalidInput))
}

func TestAdminRoleServiceCreateReturnsRoleAndAuditLog(t *testing.T) {
	setupLogger(t)

	logCh := make(chan *models.Log, 1)
	repo := &mockAdminRoleRepository{
		createFn: func(ctx context.Context, role *models.AdminRole) error {
			require.Equal(t, "req-2", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, "Manager", role.Name)
			require.True(t, role.IsActive)
			role.ID = 8
			return nil
		},
	}
	casbinClient := &mockCasbinClient{
		addRolePermissionsFn: func(roleID uint, perms []string) error {
			require.Equal(t, uint(8), roleID)
			require.Equal(t, []string{permissions.ProductRead.String()}, perms)
			return nil
		},
		getRolePermissionsFn: func(roleID uint) []string {
			require.Equal(t, uint(8), roleID)
			return []string{permissions.ProductRead.String()}
		},
	}
	logRepo := &mockLogRepository{
		createFn: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}

	svc := service.NewAdminRoleService(repo, logRepo, casbinClient)
	ctx := utils.SetRequestIDToContext(context.Background(), "req-2")
	ctx = utils.NewContextWithValues(ctx, utils.ContextValues{UserID: 11, UserName: "Alice"})

	role, err := svc.Create(ctx, &dto.CreateAdminRoleRequest{
		Name:        "Manager",
		Description: "Can manage products",
		Permissions: []string{permissions.ProductRead.String()},
	})

	require.NoError(t, err)
	require.NotNil(t, role)
	require.Equal(t, uint(8), role.ID)
	require.Equal(t, []string{permissions.ProductRead.String()}, role.Permissions)

	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionCreate, entry.Action)
		require.Equal(t, models.LogEntityTypeAdminRole, entry.EntityType)
		require.Equal(t, uint(8), entry.EntityID)
		require.Equal(t, "Alice created admin role: Manager", entry.Message)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestAdminRoleServiceDeleteRejectsAssignedUsers(t *testing.T) {
	setupLogger(t)

	repo := &mockAdminRoleRepository{
		findByIDFn: func(context.Context, uint, ...repository.Association) (*models.AdminRole, error) {
			return &models.AdminRole{ID: 3, Name: "Manager"}, nil
		},
		countUsersWithRoleFn: func(context.Context, uint) (int64, error) {
			return 1, nil
		},
	}

	svc := service.NewAdminRoleService(repo, &mockLogRepository{}, &mockCasbinClient{})

	err := svc.Delete(context.Background(), 3)

	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrInvalidInput))
}

func TestAdminRoleServiceFindByIDReturnsRoleWithPermissions(t *testing.T) {
	setupLogger(t)

	repo := &mockAdminRoleRepository{
		findByIDFn: func(ctx context.Context, id uint, _ ...repository.Association) (*models.AdminRole, error) {
			require.Equal(t, "req-3", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(4), id)
			return &models.AdminRole{ID: 4, Name: "Support"}, nil
		},
	}
	casbinClient := &mockCasbinClient{
		getRolePermissionsFn: func(roleID uint) []string {
			require.Equal(t, uint(4), roleID)
			return []string{permissions.UserRead.String()}
		},
	}

	svc := service.NewAdminRoleService(repo, &mockLogRepository{}, casbinClient)
	ctx := utils.SetRequestIDToContext(context.Background(), "req-3")

	role, err := svc.FindById(ctx, 4)

	require.NoError(t, err)
	require.NotNil(t, role)
	require.Equal(t, []string{permissions.UserRead.String()}, role.Permissions)
}

func TestAdminRoleServiceGetAllPermissionsReturnsFrontendMap(t *testing.T) {
	setupLogger(t)

	svc := service.NewAdminRoleService(&mockAdminRoleRepository{}, &mockLogRepository{}, &mockCasbinClient{})

	got := svc.GetAllPermissions(context.Background())

	require.NotEmpty(t, got)
	require.Contains(t, got, permissions.ResourceAdminRole)
}

func TestAdminRoleServiceIndexReturnsRepositoryError(t *testing.T) {
	setupLogger(t)

	expectedErr := errors.New("find all failed")
	repo := &mockAdminRoleRepository{
		findAllFn: func(context.Context, *pagination.Pagination) ([]*models.AdminRole, error) {
			return nil, expectedErr
		},
		countFn: func(context.Context, *pagination.Pagination) (int64, error) {
			t.Fatal("Count should not be called when FindAll fails")
			return 0, nil
		},
	}

	svc := service.NewAdminRoleService(repo, &mockLogRepository{}, &mockCasbinClient{})

	roles, meta, err := svc.Index(context.Background(), pagination.NewPagination(nil, nil, pagination.PaginationOptions{}))

	require.Nil(t, roles)
	require.Equal(t, response.Meta{}, meta)
	require.ErrorIs(t, err, expectedErr)
}
