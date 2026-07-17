package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	adminrolemocks "github.com/PhantomX7/athleton/internal/modules/admin_role/repository/mocks"
	logmocks "github.com/PhantomX7/athleton/internal/modules/log/repository/mocks"
	refreshtokenmocks "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository/mocks"
	usermocks "github.com/PhantomX7/athleton/internal/modules/user/repository/mocks"
	"github.com/PhantomX7/athleton/internal/modules/user/service"
	casbinmocks "github.com/PhantomX7/athleton/libs/casbin/mocks"
	txmocks "github.com/PhantomX7/athleton/libs/transaction_manager/mocks"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"
)

// existingAdminRoleRepo returns a mock whose locked find succeeds for the
// given role ID, mimicking an existing, lockable admin-role row.
func existingAdminRoleRepo(t *testing.T, roleID uint) *adminrolemocks.AdminRoleRepositoryMock {
	t.Helper()
	return &adminrolemocks.AdminRoleRepositoryMock{
		FindByIDForUpdateFunc: func(_ context.Context, id uint) (*models.AdminRole, error) {
			require.Equal(t, roleID, id)
			return &models.AdminRole{ID: id, Name: "Manager", IsActive: true}, nil
		},
	}
}

// passthroughTxManager returns a mock transaction manager that simply invokes
// the closure with the original context, mimicking a committed transaction.
func passthroughTxManager() *txmocks.TransactionManagerMock {
	return &txmocks.TransactionManagerMock{
		ExecuteInTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}
}

func TestUserServiceIndexReturnsUsersAndMeta(t *testing.T) {
	pg := pagination.NewPagination(map[string][]string{"limit": {"2"}}, nil, pagination.PaginationOptions{})
	repo := &usermocks.UserRepositoryMock{
		FindAllFunc: func(ctx context.Context, gotPg *pagination.Pagination) ([]*models.User, error) {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return []*models.User{
				{ID: 1, Username: "alice"},
				{ID: 2, Username: "bob"},
			}, nil
		},
		CountFunc: func(ctx context.Context, gotPg *pagination.Pagination) (int64, error) {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return 2, nil
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, &txmocks.TransactionManagerMock{}, zap.NewNop())
	ctx := utils.SetRequestIDToContext(context.Background(), "req-1")

	users, meta, err := svc.Index(ctx, pg)

	require.NoError(t, err)
	require.Len(t, users, 2)
	require.Equal(t, int64(2), meta.Total)
}

func TestUserServiceFindByIDHydratesAdminRolePermissions(t *testing.T) {
	roleID := uint(9)
	repo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, preloads ...repository.Association) (*models.User, error) {
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
	casbinClient := &casbinmocks.ClientMock{
		GetRolePermissionsFunc: func(gotRoleID uint) []string {
			require.Equal(t, roleID, gotRoleID)
			return []string{permissions.UserRead.String()}
		},
		// Mirror the real client: root bypasses permission checks.
		CheckPermissionWithRootFunc: func(role string, _ *uint, _ string) (bool, error) {
			return role == models.UserRoleRoot.ToString(), nil
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, casbinClient, &txmocks.TransactionManagerMock{}, zap.NewNop())
	ctx := utils.SetRequestIDToContext(context.Background(), "req-2")
	// Root caller: bypasses the admin_user:read check for the admin target.
	ctx = utils.NewContextWithValues(ctx, utils.ContextValues{UserID: 1, UserName: "Root", Role: models.UserRoleRoot.ToString()})

	user, err := svc.FindByID(ctx, 7)

	require.NoError(t, err)
	require.NotNil(t, user.AdminRole)
	require.Equal(t, []string{permissions.UserRead.String()}, user.AdminRole.Permissions)
}

func TestUserServiceCreateCreatesAdminAccount(t *testing.T) {
	logCh := make(chan *models.Log, 1)
	roleID := uint(5)

	roleLocked := false
	adminRoleRepo := &adminrolemocks.AdminRoleRepositoryMock{
		FindByIDForUpdateFunc: func(_ context.Context, id uint) (*models.AdminRole, error) {
			require.Equal(t, roleID, id)
			roleLocked = true
			return &models.AdminRole{ID: roleID, Name: "Manager"}, nil
		},
	}
	repo := &usermocks.UserRepositoryMock{
		CreateFunc: func(_ context.Context, entity *models.User) error {
			require.True(t, roleLocked, "the admin-role row must be locked before the user insert")
			// The created account is always a plain admin — role is never
			// taken from the request, so root can never be created.
			require.Equal(t, models.UserRoleAdmin, entity.Role)
			require.Equal(t, &roleID, entity.AdminRoleID)
			require.True(t, entity.IsActive)
			require.Equal(t, "new.admin@test.local", entity.Email, "email must be normalized to lowercase")
			require.Equal(t, "new-admin", entity.Username)
			// The password was chosen by the creator, not the account owner:
			// leave PasswordChangedAt nil so the must-change-default-password
			// gate forces a rotation on first login.
			require.Nil(t, entity.PasswordChangedAt)
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(entity.Password), []byte("initial-pass-123")))
			entity.ID = 9
			return nil
		},
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(_ context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}

	svc := service.NewUserService(repo, adminRoleRepo, &refreshtokenmocks.RefreshTokenRepositoryMock{}, logRepo, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 1, UserName: "Root"})

	user, err := svc.Create(ctx, &dto.AdminUserCreateRequest{
		Username:    "new-admin",
		Name:        "New Admin",
		Email:       "  New.Admin@Test.Local ",
		Phone:       "+620000000007",
		Password:    "initial-pass-123",
		AdminRoleID: roleID,
	})

	require.NoError(t, err)
	require.Equal(t, uint(9), user.ID)

	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionCreate, entry.Action)
		require.Equal(t, models.LogEntityTypeUser, entry.EntityType)
		require.Equal(t, uint(9), entry.EntityID)
	case <-time.After(2 * time.Second):
		t.Fatal("creating an admin account must produce an audit log")
	}
}

func TestUserServiceCreateFailsWhenRoleMissing(t *testing.T) {
	adminRoleRepo := &adminrolemocks.AdminRoleRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.AdminRole, error) {
			return nil, cerrors.NewNotFoundError("admin role not found")
		},
	}

	svc := service.NewUserService(&usermocks.UserRepositoryMock{}, adminRoleRepo, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())

	user, err := svc.Create(context.Background(), &dto.AdminUserCreateRequest{
		Username:    "new-admin",
		Name:        "New Admin",
		Email:       "new.admin@test.local",
		Phone:       "+620000000007",
		Password:    "initial-pass-123",
		AdminRoleID: 99,
	})

	require.Nil(t, user)
	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

// adminCallerValues returns context values for an admin caller holding admin
// role 3, so CheckPermissionWithRoot consults the mock instead of bypassing.
func adminCallerValues() utils.ContextValues {
	roleID := uint(3)
	return utils.ContextValues{
		UserID:      2,
		UserName:    "Caller",
		Role:        models.UserRoleAdmin.ToString(),
		AdminRoleID: &roleID,
	}
}

func TestUserServiceUpdateRequiresAdminUserGrantForAdminTargets(t *testing.T) {
	roleID := uint(5)
	target := &models.User{ID: 6, Name: "Other Admin", Role: models.UserRoleAdmin, AdminRoleID: &roleID}

	newSvc := func(granted bool, updated *bool) service.UserService {
		repo := &usermocks.UserRepositoryMock{
			FindByIDForUpdateFunc: func(context.Context, uint) (*models.User, error) {
				return target, nil
			},
			UpdateFunc: func(context.Context, *models.User) error {
				*updated = true
				return nil
			},
		}
		casbinClient := &casbinmocks.ClientMock{
			CheckPermissionWithRootFunc: func(role string, adminRoleID *uint, perm string) (bool, error) {
				require.Equal(t, models.UserRoleAdmin.ToString(), role)
				require.Equal(t, permissions.AdminUserUpdate.String(), perm)
				return granted, nil
			},
		}
		logRepo := &logmocks.LogRepositoryMock{CreateFunc: func(context.Context, *models.Log) error { return nil }}
		return service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, logRepo, casbinClient, passthroughTxManager(), zap.NewNop())
	}

	t.Run("denied without admin_user:update", func(t *testing.T) {
		updated := false
		svc := newSvc(false, &updated)
		ctx := utils.NewContextWithValues(context.Background(), adminCallerValues())

		name := "renamed"
		user, err := svc.Update(ctx, 6, &dto.UserUpdateRequest{Name: &name})

		require.Nil(t, user)
		require.True(t, errors.Is(err, cerrors.ErrForbidden))
		require.False(t, updated, "the row must not be written when the grant is missing")
	})

	t.Run("allowed with admin_user:update", func(t *testing.T) {
		updated := false
		svc := newSvc(true, &updated)
		ctx := utils.NewContextWithValues(context.Background(), adminCallerValues())

		name := "renamed"
		user, err := svc.Update(ctx, 6, &dto.UserUpdateRequest{Name: &name})

		require.NoError(t, err)
		require.NotNil(t, user)
		require.True(t, updated)
	})
}

func TestUserServiceFindByIDRequiresAdminUserGrantForAdminTargets(t *testing.T) {
	roleID := uint(5)
	repo := &usermocks.UserRepositoryMock{
		FindByIDFunc: func(context.Context, uint, ...repository.Association) (*models.User, error) {
			return &models.User{ID: 6, Role: models.UserRoleAdmin, AdminRoleID: &roleID}, nil
		},
	}
	casbinClient := &casbinmocks.ClientMock{
		CheckPermissionWithRootFunc: func(_ string, _ *uint, perm string) (bool, error) {
			require.Equal(t, permissions.AdminUserRead.String(), perm)
			return false, nil
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, casbinClient, &txmocks.TransactionManagerMock{}, zap.NewNop())
	ctx := utils.NewContextWithValues(context.Background(), adminCallerValues())

	user, err := svc.FindByID(ctx, 6)

	require.Nil(t, user)
	require.True(t, errors.Is(err, cerrors.ErrForbidden),
		"an admin target must be hidden from callers without admin_user:read")
}

func TestUserServiceUpdateRejectsRootUser(t *testing.T) {
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 1, Role: models.UserRoleRoot, Name: "Root"}, nil
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())

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
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(ctx context.Context, id uint) (*models.User, error) {
			require.Equal(t, uint(6), id)
			return current, nil
		},
		UpdateFunc: func(ctx context.Context, entity *models.User) error {
			require.Same(t, current, entity)
			require.Equal(t, "New Name", entity.Name)
			// Role omitted from the request: role and admin-role assignment
			// must be untouched.
			require.Equal(t, models.UserRoleAdmin, entity.Role)
			require.Equal(t, &roleID, entity.AdminRoleID)
			return nil
		},
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}
	txCalls := 0
	txManager := &txmocks.TransactionManagerMock{
		ExecuteInTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			txCalls++
			return fn(ctx)
		},
	}

	casbinClient := &casbinmocks.ClientMock{
		// Mirror the real client: root bypasses permission checks.
		CheckPermissionWithRootFunc: func(role string, _ *uint, _ string) (bool, error) {
			return role == models.UserRoleRoot.ToString(), nil
		},
	}
	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, logRepo, casbinClient, txManager, zap.NewNop())
	// Root caller: bypasses the admin_user:update check for the admin target.
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 1, UserName: "Root", Role: models.UserRoleRoot.ToString()})

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
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(ctx context.Context, id uint) (*models.User, error) {
			require.Equal(t, uint(6), id)
			return current, nil
		},
		UpdateFunc: func(ctx context.Context, entity *models.User) error {
			require.Same(t, current, entity)
			require.Equal(t, models.UserRoleUser, entity.Role)
			require.Nil(t, entity.AdminRoleID, "demoting to a non-admin role must clear AdminRoleID in the same write")
			return nil
		},
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}

	casbinClient := &casbinmocks.ClientMock{
		// Mirror the real client: root bypasses permission checks (the target
		// is an admin, so demotion requires the admin_user:update grant).
		CheckPermissionWithRootFunc: func(role string, _ *uint, _ string) (bool, error) {
			return role == models.UserRoleRoot.ToString(), nil
		},
	}
	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, logRepo, casbinClient, passthroughTxManager(), zap.NewNop())
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 1, UserName: "Root", Role: models.UserRoleRoot.ToString()})

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
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.User, error) {
			return nil, expectedErr
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())

	user, err := svc.Update(context.Background(), 6, &dto.UserUpdateRequest{})

	require.Nil(t, user)
	require.ErrorIs(t, err, expectedErr)
}

func TestUserServiceAssignAdminRoleRejectsRootUser(t *testing.T) {
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 3, Role: models.UserRoleRoot, Name: "Root"}, nil
		},
	}

	svc := service.NewUserService(repo, existingAdminRoleRepo(t, 5), &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())

	user, err := svc.AssignAdminRole(context.Background(), 3, &dto.UserAssignAdminRoleRequest{AdminRoleID: 5})

	require.Nil(t, user)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrForbidden))
}

func TestUserServiceAssignAdminRoleFailsWhenRoleRowMissing(t *testing.T) {
	// The locked role read inside the transaction is the authoritative
	// existence check: when the role was deleted concurrently, the assignment
	// must fail before the user row is even read.
	adminRoleRepo := &adminrolemocks.AdminRoleRepositoryMock{
		FindByIDForUpdateFunc: func(_ context.Context, id uint) (*models.AdminRole, error) {
			require.Equal(t, uint(5), id)
			return nil, cerrors.NewNotFoundError("admin role not found")
		},
	}
	repo := &usermocks.UserRepositoryMock{} // any user-repo call panics the test

	svc := service.NewUserService(repo, adminRoleRepo, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())

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
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(ctx context.Context, id uint) (*models.User, error) {
			require.Equal(t, uint(6), id)
			return current, nil
		},
		UpdateFunc: func(ctx context.Context, entity *models.User) error {
			require.Same(t, current, entity)
			require.NotNil(t, entity.AdminRoleID)
			require.Equal(t, uint(5), *entity.AdminRoleID)
			require.Equal(t, models.UserRoleAdmin, entity.Role, "AssignAdminRole must set Role to admin in the same write")
			return nil
		},
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}
	txCalls := 0
	txManager := &txmocks.TransactionManagerMock{
		ExecuteInTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			txCalls++
			return fn(ctx)
		},
	}

	svc := service.NewUserService(repo, existingAdminRoleRepo(t, 5), &refreshtokenmocks.RefreshTokenRepositoryMock{}, logRepo, &casbinmocks.ClientMock{}, txManager, zap.NewNop())
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
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 6, Role: models.UserRoleAdmin}, nil
		},
		UpdateFunc: func(context.Context, *models.User) error {
			return expectedErr
		},
	}

	svc := service.NewUserService(repo, existingAdminRoleRepo(t, 5), &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())

	user, err := svc.AssignAdminRole(context.Background(), 6, &dto.UserAssignAdminRoleRequest{AdminRoleID: 5})

	require.Nil(t, user)
	require.ErrorIs(t, err, expectedErr)
}

func TestUserServiceChangePasswordFailsWhenTokenRevocationFails(t *testing.T) {
	expectedErr := errors.New("revoke failed")
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 10, Role: models.UserRoleAdmin, Password: "old"}, nil
		},
		UpdateFunc: func(context.Context, *models.User) error {
			return nil
		},
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		RevokeAllByUserIDFunc: func(ctx context.Context, userID uint) error {
			require.Equal(t, uint(10), userID)
			return expectedErr
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, refreshRepo, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())

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
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(ctx context.Context, id uint) (*models.User, error) {
			require.Equal(t, "req-3", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(10), id)
			return current, nil
		},
		UpdateFunc: func(ctx context.Context, entity *models.User) error {
			require.Equal(t, "req-3", utils.GetRequestIDFromContext(ctx))
			require.Same(t, current, entity)
			require.NotEqual(t, "old", entity.Password)
			require.NoError(t, bcrypt.CompareHashAndPassword([]byte(entity.Password), []byte("new-password")))
			require.NotNil(t, entity.PasswordChangedAt, "ChangePassword must clear the must-change-default-password gate")
			return nil
		},
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		RevokeAllByUserIDFunc: func(ctx context.Context, userID uint) error {
			require.Equal(t, "req-3", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(10), userID)
			return nil
		},
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, refreshRepo, logRepo, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())
	ctx := utils.SetRequestIDToContext(context.Background(), "req-3")
	ctx = utils.NewContextWithValues(ctx, utils.ContextValues{UserID: 1, UserName: "Root"})

	err := svc.ChangePassword(ctx, 10, &dto.ChangeAdminPasswordRequest{NewPassword: "new-password"})

	require.NoError(t, err)
	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionChangePassword, entry.Action)
		require.Equal(t, models.LogEntityTypeUser, entry.EntityType)
		require.Equal(t, uint(10), entry.EntityID)
		require.Equal(t, "Root changed password for: Admin User", entry.Message)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestUserServiceChangePasswordRejectsNonAdmin(t *testing.T) {
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 4, Role: models.UserRoleUser}, nil
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())

	err := svc.ChangePassword(context.Background(), 4, &dto.ChangeAdminPasswordRequest{NewPassword: "new-password"})

	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrInvalidInput))
}

func TestUserServiceChangePasswordRejectsRootUser(t *testing.T) {
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 1, Role: models.UserRoleRoot}, nil
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())

	err := svc.ChangePassword(context.Background(), 1, &dto.ChangeAdminPasswordRequest{NewPassword: "new-password"})

	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrForbidden))
}

func TestUserServiceDeleteRejectsRootUser(t *testing.T) {
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 1, Role: models.UserRoleRoot, Name: "Root"}, nil
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())

	err := svc.Delete(context.Background(), 1)

	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrForbidden))
}

func TestUserServiceDeleteRejectsSelfDeletion(t *testing.T) {
	deleted := false
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(_ context.Context, id uint) (*models.User, error) {
			return &models.User{ID: id, Role: models.UserRoleAdmin, Name: "Caller"}, nil
		},
		DeleteFunc: func(context.Context, *models.User) error {
			deleted = true
			return nil
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())
	// adminCallerValues has UserID 2 — target the same account.
	ctx := utils.NewContextWithValues(context.Background(), adminCallerValues())

	err := svc.Delete(ctx, 2)

	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrForbidden))
	require.False(t, deleted, "an account must not be able to delete itself")
}

func TestUserServiceDeleteRequiresAdminUserGrantForAdminTargets(t *testing.T) {
	roleID := uint(5)

	newSvc := func(granted bool, deleted *bool) service.UserService {
		repo := &usermocks.UserRepositoryMock{
			FindByIDForUpdateFunc: func(context.Context, uint) (*models.User, error) {
				return &models.User{ID: 6, Name: "Other Admin", Role: models.UserRoleAdmin, AdminRoleID: &roleID}, nil
			},
			DeleteFunc: func(context.Context, *models.User) error {
				*deleted = true
				return nil
			},
		}
		casbinClient := &casbinmocks.ClientMock{
			CheckPermissionWithRootFunc: func(role string, _ *uint, perm string) (bool, error) {
				require.Equal(t, models.UserRoleAdmin.ToString(), role)
				require.Equal(t, permissions.AdminUserDelete.String(), perm)
				return granted, nil
			},
		}
		refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
			RevokeAllByUserIDFunc: func(context.Context, uint) error { return nil },
		}
		logRepo := &logmocks.LogRepositoryMock{CreateFunc: func(context.Context, *models.Log) error { return nil }}
		return service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, refreshRepo, logRepo, casbinClient, passthroughTxManager(), zap.NewNop())
	}

	t.Run("denied without admin_user:delete", func(t *testing.T) {
		deleted := false
		svc := newSvc(false, &deleted)
		ctx := utils.NewContextWithValues(context.Background(), adminCallerValues())

		err := svc.Delete(ctx, 6)

		require.True(t, errors.Is(err, cerrors.ErrForbidden))
		require.False(t, deleted, "the row must not be deleted when the grant is missing")
	})

	t.Run("allowed with admin_user:delete", func(t *testing.T) {
		deleted := false
		svc := newSvc(true, &deleted)
		ctx := utils.NewContextWithValues(context.Background(), adminCallerValues())

		err := svc.Delete(ctx, 6)

		require.NoError(t, err)
		require.True(t, deleted)
	})
}

func TestUserServiceDeleteSoftDeletesRevokesTokensAndLogs(t *testing.T) {
	logCh := make(chan *models.Log, 1)
	current := &models.User{ID: 6, Name: "Plain User", Username: "plain", Role: models.UserRoleUser}
	deleted := false
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(ctx context.Context, id uint) (*models.User, error) {
			require.Equal(t, "req-4", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(6), id)
			return current, nil
		},
		DeleteFunc: func(ctx context.Context, entity *models.User) error {
			require.Equal(t, "req-4", utils.GetRequestIDFromContext(ctx))
			require.Same(t, current, entity)
			deleted = true
			return nil
		},
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		RevokeAllByUserIDFunc: func(ctx context.Context, userID uint) error {
			require.Equal(t, "req-4", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(6), userID)
			require.True(t, deleted, "sessions are revoked in the same transaction, after the delete")
			return nil
		},
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(_ context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}
	txCalls := 0
	txManager := &txmocks.TransactionManagerMock{
		ExecuteInTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			txCalls++
			return fn(ctx)
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, refreshRepo, logRepo, &casbinmocks.ClientMock{}, txManager, zap.NewNop())
	ctx := utils.SetRequestIDToContext(context.Background(), "req-4")
	ctx = utils.NewContextWithValues(ctx, utils.ContextValues{UserID: 1, UserName: "Root", Role: models.UserRoleRoot.ToString()})

	err := svc.Delete(ctx, 6)

	require.NoError(t, err)
	require.True(t, deleted)
	require.Equal(t, 1, txCalls)
	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionDelete, entry.Action)
		require.Equal(t, models.LogEntityTypeUser, entry.EntityType)
		require.Equal(t, uint(6), entry.EntityID)
	case <-time.After(2 * time.Second):
		t.Fatal("deleting a user must produce an audit log")
	}
}

func TestUserServiceDeleteFailsWhenTokenRevocationFails(t *testing.T) {
	expectedErr := errors.New("revoke failed")
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.User, error) {
			return &models.User{ID: 6, Role: models.UserRoleUser}, nil
		},
		DeleteFunc: func(context.Context, *models.User) error { return nil },
	}
	refreshRepo := &refreshtokenmocks.RefreshTokenRepositoryMock{
		RevokeAllByUserIDFunc: func(context.Context, uint) error { return expectedErr },
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, refreshRepo, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 1, UserName: "Root", Role: models.UserRoleRoot.ToString()})

	err := svc.Delete(ctx, 6)

	require.ErrorIs(t, err, expectedErr)
}

func TestUserServiceDeletePropagatesFindError(t *testing.T) {
	repo := &usermocks.UserRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.User, error) {
			return nil, cerrors.NewNotFoundError("user with id 99 not found")
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager(), zap.NewNop())

	err := svc.Delete(context.Background(), 99)

	require.True(t, errors.Is(err, cerrors.ErrNotFound))
}

func TestUserServiceIndexReturnsRepositoryError(t *testing.T) {
	expectedErr := errors.New("find all failed")
	repo := &usermocks.UserRepositoryMock{
		FindAllFunc: func(context.Context, *pagination.Pagination) ([]*models.User, error) {
			return nil, expectedErr
		},
		CountFunc: func(context.Context, *pagination.Pagination) (int64, error) {
			t.Fatal("Count should not be called when FindAll fails")
			return 0, nil
		},
	}

	svc := service.NewUserService(repo, &adminrolemocks.AdminRoleRepositoryMock{}, &refreshtokenmocks.RefreshTokenRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, &txmocks.TransactionManagerMock{}, zap.NewNop())

	users, meta, err := svc.Index(context.Background(), pagination.NewPagination(nil, nil, pagination.PaginationOptions{}))

	require.Nil(t, users)
	require.Equal(t, response.Meta{}, meta)
	require.ErrorIs(t, err, expectedErr)
}
