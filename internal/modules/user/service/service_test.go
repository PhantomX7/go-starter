package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	casbinv2 "github.com/casbin/casbin/v3"
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
	"github.com/PhantomX7/athleton/libs/transaction_manager"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"
)

type mockUserRepository struct {
	findAllFn           func(context.Context, *pagination.Pagination) ([]*models.User, error)
	countFn             func(context.Context, *pagination.Pagination) (int64, error)
	findByIDFn          func(context.Context, uint, ...repository.Association) (*models.User, error)
	findByIDForUpdateFn func(context.Context, uint) (*models.User, error)
	updateFn            func(context.Context, *models.User) error
	findByUsernameFn    func(context.Context, string) (*models.User, error)
	findByEmailFn       func(context.Context, string) (*models.User, error)
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

func (m *mockUserRepository) FindByIDForUpdate(ctx context.Context, id uint) (*models.User, error) {
	if m.findByIDForUpdateFn == nil {
		panic("unexpected FindByIDForUpdate call")
	}
	return m.findByIDForUpdateFn(ctx, id)
}

var _ userrepository.UserRepository = (*mockUserRepository)(nil)

// mockAdminRoleRepository is the minimal admin-role repository stand-in the
// user service needs: AssignAdminRole locks the target role row via
// FindByIDForUpdate. Every other method panics so an unexpected call fails
// the test loudly.
type mockAdminRoleRepository struct {
	findByIDForUpdateFn func(context.Context, uint) (*models.AdminRole, error)
}

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
func (m *mockAdminRoleRepository) FindByIDForUpdate(ctx context.Context, id uint) (*models.AdminRole, error) {
	if m.findByIDForUpdateFn == nil {
		panic("unexpected FindByIDForUpdate call")
	}
	return m.findByIDForUpdateFn(ctx, id)
}

var _ adminrolerepository.AdminRoleRepository = (*mockAdminRoleRepository)(nil)

// existingAdminRoleRepo returns a mock whose locked find succeeds for the
// given role ID, mimicking an existing, lockable admin-role row.
func existingAdminRoleRepo(t *testing.T, roleID uint) *mockAdminRoleRepository {
	t.Helper()
	return &mockAdminRoleRepository{
		findByIDForUpdateFn: func(_ context.Context, id uint) (*models.AdminRole, error) {
			require.Equal(t, roleID, id)
			return &models.AdminRole{ID: id, Name: "Manager", IsActive: true}, nil
		},
	}
}

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
func (m *mockRefreshTokenRepository) RevokeOldestActiveByUserID(context.Context, uint, int) error {
	panic("unexpected RevokeOldestActiveByUserID call")
}
func (m *mockRefreshTokenRepository) UpdateTokenHashIfActive(context.Context, string, string) (bool, error) {
	panic("unexpected UpdateTokenHashIfActive call")
}
func (m *mockRefreshTokenRepository) RevokeByToken(context.Context, string) error {
	panic("unexpected RevokeByToken call")
}
func (m *mockRefreshTokenRepository) RevokeByTokenIfActive(context.Context, string) (bool, error) {
	panic("unexpected RevokeByTokenIfActive call")
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

type mockTxManager struct {
	executeFn func(context.Context, func(context.Context) error) error
}

func (m *mockTxManager) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	if m.executeFn == nil {
		panic("unexpected ExecuteInTransaction call")
	}
	return m.executeFn(ctx, fn)
}

var _ transaction_manager.TransactionManager = (*mockTxManager)(nil)

// passthroughTxManager returns a mock transaction manager that simply invokes
// the closure with the original context, mimicking a committed transaction.
func passthroughTxManager() *mockTxManager {
	return &mockTxManager{
		executeFn: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}
}

func TestUserServiceIndexReturnsUsersAndMeta(t *testing.T) {
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

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{}, &mockTxManager{}, zap.NewNop())
	ctx := utils.SetRequestIDToContext(context.Background(), "req-1")

	users, meta, err := svc.Index(ctx, pg)

	require.NoError(t, err)
	require.Len(t, users, 2)
	require.Equal(t, int64(2), meta.Total)
}

func TestUserServiceFindByIDHydratesAdminRolePermissions(t *testing.T) {
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

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, &mockLogRepository{}, casbinClient, &mockTxManager{}, zap.NewNop())
	ctx := utils.SetRequestIDToContext(context.Background(), "req-2")

	user, err := svc.FindByID(ctx, 7)

	require.NoError(t, err)
	require.NotNil(t, user.AdminRole)
	require.Equal(t, []string{permissions.UserRead.String()}, user.AdminRole.Permissions)
}

func TestUserServiceUpdateRejectsRootUser(t *testing.T) {
	repo := &mockUserRepository{
		findByIDForUpdateFn: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 1, Role: models.UserRoleRoot, Name: "Root"}, nil
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{}, passthroughTxManager(), zap.NewNop())

	name := "renamed"
	user, err := svc.Update(context.Background(), 1, &dto.UserUpdateRequest{Name: &name})

	require.Nil(t, user)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrForbidden))
}

func TestUserServiceUpdateAppliesPatchSemanticsInTransaction(t *testing.T) {
	logCh := make(chan *models.Log, 1)
	roleID := uint(5)
	current := &models.User{ID: 6, Name: "Old Name", Role: models.UserRoleAdmin, AdminRoleID: &roleID}
	repo := &mockUserRepository{
		findByIDForUpdateFn: func(ctx context.Context, id uint) (*models.User, error) {
			require.Equal(t, uint(6), id)
			return current, nil
		},
		updateFn: func(ctx context.Context, entity *models.User) error {
			require.Same(t, current, entity)
			require.Equal(t, "New Name", entity.Name)
			// Role omitted from the request: role and admin-role assignment
			// must be untouched.
			require.Equal(t, models.UserRoleAdmin, entity.Role)
			require.Equal(t, &roleID, entity.AdminRoleID)
			return nil
		},
	}
	logRepo := &mockLogRepository{
		createFn: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}
	txCalls := 0
	txManager := &mockTxManager{
		executeFn: func(ctx context.Context, fn func(context.Context) error) error {
			txCalls++
			return fn(ctx)
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, logRepo, &mockCasbinClient{}, txManager, zap.NewNop())
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 1, UserName: "Root"})

	name := "New Name"
	user, err := svc.Update(ctx, 6, &dto.UserUpdateRequest{Name: &name})

	require.NoError(t, err)
	require.Same(t, current, user)
	require.Equal(t, 1, txCalls)
	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionUpdate, entry.Action)
		require.Equal(t, uint(6), entry.EntityID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestUserServiceUpdateDemotionClearsAdminRoleID(t *testing.T) {
	logCh := make(chan *models.Log, 1)
	roleID := uint(5)
	current := &models.User{ID: 6, Name: "Admin User", Role: models.UserRoleAdmin, AdminRoleID: &roleID}
	repo := &mockUserRepository{
		findByIDForUpdateFn: func(ctx context.Context, id uint) (*models.User, error) {
			require.Equal(t, uint(6), id)
			return current, nil
		},
		updateFn: func(ctx context.Context, entity *models.User) error {
			require.Same(t, current, entity)
			require.Equal(t, models.UserRoleUser, entity.Role)
			require.Nil(t, entity.AdminRoleID, "demoting to a non-admin role must clear AdminRoleID in the same write")
			return nil
		},
	}
	logRepo := &mockLogRepository{
		createFn: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, logRepo, &mockCasbinClient{}, passthroughTxManager(), zap.NewNop())
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 1, UserName: "Root"})

	role := "user"
	user, err := svc.Update(ctx, 6, &dto.UserUpdateRequest{Role: &role})

	require.NoError(t, err)
	require.Same(t, current, user)
	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionUpdate, entry.Action)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestUserServiceUpdatePropagatesFindError(t *testing.T) {
	expectedErr := errors.New("find failed")
	repo := &mockUserRepository{
		findByIDForUpdateFn: func(context.Context, uint) (*models.User, error) {
			return nil, expectedErr
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{}, passthroughTxManager(), zap.NewNop())

	user, err := svc.Update(context.Background(), 6, &dto.UserUpdateRequest{})

	require.Nil(t, user)
	require.ErrorIs(t, err, expectedErr)
}

func TestUserServiceAssignAdminRoleRejectsRootUser(t *testing.T) {
	repo := &mockUserRepository{
		findByIDForUpdateFn: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 3, Role: models.UserRoleRoot, Name: "Root"}, nil
		},
	}

	svc := service.NewUserService(repo, existingAdminRoleRepo(t, 5), &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{}, passthroughTxManager(), zap.NewNop())

	user, err := svc.AssignAdminRole(context.Background(), 3, &dto.UserAssignAdminRoleRequest{AdminRoleID: 5})

	require.Nil(t, user)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrForbidden))
}

func TestUserServiceAssignAdminRoleFailsWhenRoleRowMissing(t *testing.T) {
	// The locked role read inside the transaction is the authoritative
	// existence check: when the role was deleted concurrently, the assignment
	// must fail before the user row is even read.
	adminRoleRepo := &mockAdminRoleRepository{
		findByIDForUpdateFn: func(_ context.Context, id uint) (*models.AdminRole, error) {
			require.Equal(t, uint(5), id)
			return nil, cerrors.NewNotFoundError("admin role not found")
		},
	}
	repo := &mockUserRepository{} // any user-repo call panics the test

	svc := service.NewUserService(repo, adminRoleRepo, &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{}, passthroughTxManager(), zap.NewNop())

	user, err := svc.AssignAdminRole(context.Background(), 6, &dto.UserAssignAdminRoleRequest{AdminRoleID: 5})

	require.Nil(t, user)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

func TestUserServiceAssignAdminRoleUpdatesUserInTransaction(t *testing.T) {
	logCh := make(chan *models.Log, 1)
	// Start from a plain user: assignment must both attach the admin role and
	// promote Role to admin, otherwise Casbin ignores the AdminRoleID entirely.
	current := &models.User{ID: 6, Name: "Admin User", Role: models.UserRoleUser}
	repo := &mockUserRepository{
		findByIDForUpdateFn: func(ctx context.Context, id uint) (*models.User, error) {
			require.Equal(t, uint(6), id)
			return current, nil
		},
		updateFn: func(ctx context.Context, entity *models.User) error {
			require.Same(t, current, entity)
			require.NotNil(t, entity.AdminRoleID)
			require.Equal(t, uint(5), *entity.AdminRoleID)
			require.Equal(t, models.UserRoleAdmin, entity.Role, "AssignAdminRole must set Role to admin in the same write")
			return nil
		},
	}
	logRepo := &mockLogRepository{
		createFn: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}
	txCalls := 0
	txManager := &mockTxManager{
		executeFn: func(ctx context.Context, fn func(context.Context) error) error {
			txCalls++
			return fn(ctx)
		},
	}

	svc := service.NewUserService(repo, existingAdminRoleRepo(t, 5), &mockRefreshTokenRepository{}, logRepo, &mockCasbinClient{}, txManager, zap.NewNop())
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 1, UserName: "Root"})

	user, err := svc.AssignAdminRole(ctx, 6, &dto.UserAssignAdminRoleRequest{AdminRoleID: 5})

	require.NoError(t, err)
	require.Same(t, current, user)
	require.Equal(t, 1, txCalls)
	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionUpdate, entry.Action)
		require.Equal(t, uint(6), entry.EntityID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestUserServiceAssignAdminRolePropagatesUpdateErrorFromTransaction(t *testing.T) {
	expectedErr := errors.New("update failed")
	repo := &mockUserRepository{
		findByIDForUpdateFn: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 6, Role: models.UserRoleAdmin}, nil
		},
		updateFn: func(context.Context, *models.User) error {
			return expectedErr
		},
	}

	svc := service.NewUserService(repo, existingAdminRoleRepo(t, 5), &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{}, passthroughTxManager(), zap.NewNop())

	user, err := svc.AssignAdminRole(context.Background(), 6, &dto.UserAssignAdminRoleRequest{AdminRoleID: 5})

	require.Nil(t, user)
	require.ErrorIs(t, err, expectedErr)
}

func TestUserServiceChangePasswordFailsWhenTokenRevocationFails(t *testing.T) {
	expectedErr := errors.New("revoke failed")
	repo := &mockUserRepository{
		findByIDForUpdateFn: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 10, Role: models.UserRoleAdmin, Password: "old"}, nil
		},
		updateFn: func(context.Context, *models.User) error {
			return nil
		},
	}
	refreshRepo := &mockRefreshTokenRepository{
		revokeAllByUserIDFn: func(ctx context.Context, userID uint) error {
			require.Equal(t, uint(10), userID)
			return expectedErr
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, refreshRepo, &mockLogRepository{}, &mockCasbinClient{}, passthroughTxManager(), zap.NewNop())

	err := svc.ChangePassword(context.Background(), 10, &dto.ChangeAdminPasswordRequest{NewPassword: "new-password"})

	require.ErrorIs(t, err, expectedErr)
}

func TestUserServiceChangePasswordUpdatesHashRevokesTokensAndLogs(t *testing.T) {
	logCh := make(chan *models.Log, 1)
	current := &models.User{
		ID:       10,
		Name:     "Admin User",
		Username: "admin",
		Role:     models.UserRoleAdmin,
		Password: "old",
	}
	repo := &mockUserRepository{
		findByIDForUpdateFn: func(ctx context.Context, id uint) (*models.User, error) {
			require.Equal(t, "req-3", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(10), id)
			return current, nil
		},
		updateFn: func(ctx context.Context, entity *models.User) error {
			require.Equal(t, "req-3", utils.GetRequestIDFromContext(ctx))
			require.Same(t, current, entity)
			require.NotEqual(t, "old", entity.Password)
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(entity.Password), []byte("new-password")))
			require.NotNil(t, entity.PasswordChangedAt, "ChangePassword must clear the must-change-default-password gate")
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

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, refreshRepo, logRepo, &mockCasbinClient{}, passthroughTxManager(), zap.NewNop())
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
	repo := &mockUserRepository{
		findByIDForUpdateFn: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 4, Role: models.UserRoleUser}, nil
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{}, passthroughTxManager(), zap.NewNop())

	err := svc.ChangePassword(context.Background(), 4, &dto.ChangeAdminPasswordRequest{NewPassword: "new-password"})

	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrInvalidInput))
}

func TestUserServiceChangePasswordRejectsRootUser(t *testing.T) {
	repo := &mockUserRepository{
		findByIDForUpdateFn: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 1, Role: models.UserRoleRoot}, nil
		},
	}

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{}, passthroughTxManager(), zap.NewNop())

	err := svc.ChangePassword(context.Background(), 1, &dto.ChangeAdminPasswordRequest{NewPassword: "new-password"})

	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrForbidden))
}

func TestUserServiceIndexReturnsRepositoryError(t *testing.T) {
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

	svc := service.NewUserService(repo, &mockAdminRoleRepository{}, &mockRefreshTokenRepository{}, &mockLogRepository{}, &mockCasbinClient{}, &mockTxManager{}, zap.NewNop())

	users, meta, err := svc.Index(context.Background(), pagination.NewPagination(nil, nil, pagination.PaginationOptions{}))

	require.Nil(t, users)
	require.Equal(t, response.Meta{}, meta)
	require.ErrorIs(t, err, expectedErr)
}
