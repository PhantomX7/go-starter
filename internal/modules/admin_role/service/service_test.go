package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
	adminrolemocks "github.com/PhantomX7/athleton/internal/modules/admin_role/repository/mocks"
	"github.com/PhantomX7/athleton/internal/modules/admin_role/service"
	logmocks "github.com/PhantomX7/athleton/internal/modules/log/repository/mocks"
	casbinmocks "github.com/PhantomX7/athleton/libs/casbin/mocks"
	txmocks "github.com/PhantomX7/athleton/libs/transaction_manager/mocks"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/pagination"
	"github.com/PhantomX7/athleton/pkg/repository"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"
)

// passthroughTxManager returns a mock transaction manager that simply invokes
// the closure with the original context, mimicking a committed transaction.
func passthroughTxManager() *txmocks.TransactionManagerMock {
	return &txmocks.TransactionManagerMock{
		ExecuteInTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}
}

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
	repo := &adminrolemocks.AdminRoleRepositoryMock{
		FindAllFunc: func(ctx context.Context, gotPg *pagination.Pagination) ([]*models.AdminRole, error) {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return []*models.AdminRole{
				{ID: 1, Name: "Manager", IsActive: true},
				{ID: 2, Name: "Support", IsActive: true},
			}, nil
		},
		CountFunc: func(ctx context.Context, gotPg *pagination.Pagination) (int64, error) {
			require.Equal(t, "req-1", utils.GetRequestIDFromContext(ctx))
			require.Same(t, pg, gotPg)
			return 2, nil
		},
	}
	casbinClient := &casbinmocks.ClientMock{
		GetRolePermissionsFunc: func(roleID uint) []string {
			if roleID == 1 {
				return []string{permissions.LogRead.String()}
			}
			return []string{permissions.UserRead.String()}
		},
	}

	svc := service.NewAdminRoleService(repo, &logmocks.LogRepositoryMock{}, casbinClient, &txmocks.TransactionManagerMock{})
	ctx := utils.SetRequestIDToContext(context.Background(), "req-1")

	roles, meta, err := svc.Index(ctx, pg)

	require.NoError(t, err)
	require.Len(t, roles, 2)
	require.Equal(t, []string{permissions.LogRead.String()}, roles[0].Permissions)
	require.Equal(t, []string{permissions.UserRead.String()}, roles[1].Permissions)
	require.Equal(t, int64(2), meta.Total)
}

func TestAdminRoleServiceCreateRejectsInvalidPermissions(t *testing.T) {
	setupLogger(t)

	svc := service.NewAdminRoleService(&adminrolemocks.AdminRoleRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, &txmocks.TransactionManagerMock{})

	role, err := svc.Create(context.Background(), &dto.CreateAdminRoleRequest{
		Name:        "Manager",
		Description: "test",
		Permissions: []string{"not:valid"},
	})

	require.Nil(t, role)
	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrInvalidInput))
}

func TestAdminRoleServiceCreateRejectsPermissionsCallerDoesNotHold(t *testing.T) {
	setupLogger(t)

	callerRoleID := uint(5)
	casbinClient := &casbinmocks.ClientMock{
		CheckPermissionWithRootFunc: func(userRole string, adminRoleID *uint, permission string) (bool, error) {
			require.Equal(t, "admin", userRole)
			require.Equal(t, callerRoleID, *adminRoleID)
			// The caller only holds log:read.
			return permission == permissions.LogRead.String(), nil
		},
	}
	repo := &adminrolemocks.AdminRoleRepositoryMock{
		CreateFunc: func(context.Context, *models.AdminRole) error {
			t.Fatal("Create must not be called when the caller lacks a requested permission")
			return nil
		},
	}

	svc := service.NewAdminRoleService(repo, &logmocks.LogRepositoryMock{}, casbinClient, &txmocks.TransactionManagerMock{})
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{
		UserID: 11, UserName: "Alice", Role: "admin", AdminRoleID: &callerRoleID,
	})

	role, err := svc.Create(ctx, &dto.CreateAdminRoleRequest{
		Name:        "Manager",
		Description: "test",
		Permissions: []string{permissions.LogRead.String(), permissions.UserRead.String()},
	})

	require.Nil(t, role)
	require.ErrorIs(t, err, cerrors.ErrForbidden)
}

func TestAdminRoleServiceCreateFailsClosedWithoutCallerIdentity(t *testing.T) {
	setupLogger(t)

	repo := &adminrolemocks.AdminRoleRepositoryMock{
		CreateFunc: func(context.Context, *models.AdminRole) error {
			t.Fatal("Create must not be called when the caller identity is missing")
			return nil
		},
	}

	svc := service.NewAdminRoleService(repo, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, &txmocks.TransactionManagerMock{})

	// No ContextValues at all: an unauthenticated/internal context must not be
	// able to mint roles with permissions.
	role, err := svc.Create(context.Background(), &dto.CreateAdminRoleRequest{
		Name:        "Manager",
		Permissions: []string{permissions.LogRead.String()},
	})

	require.Nil(t, role)
	require.ErrorIs(t, err, cerrors.ErrForbidden)
}

func TestAdminRoleServiceUpdateRejectsAddedPermissionsCallerDoesNotHold(t *testing.T) {
	setupLogger(t)

	callerRoleID := uint(5)
	casbinClient := &casbinmocks.ClientMock{
		// Target role currently holds only log:read.
		GetRolePermissionsFunc: func(roleID uint) []string {
			require.Equal(t, uint(8), roleID)
			return []string{permissions.LogRead.String()}
		},
		CheckPermissionWithRootFunc: func(userRole string, adminRoleID *uint, permission string) (bool, error) {
			// The caller holds nothing beyond role management.
			return false, nil
		},
	}
	repo := &adminrolemocks.AdminRoleRepositoryMock{
		UpdateFunc: func(context.Context, *models.AdminRole) error {
			t.Fatal("Update must not be called when the caller lacks an added permission")
			return nil
		},
	}

	svc := service.NewAdminRoleService(repo, &logmocks.LogRepositoryMock{}, casbinClient, passthroughTxManager())
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{
		UserID: 11, UserName: "Alice", Role: "admin", AdminRoleID: &callerRoleID,
	})

	role, err := svc.Update(ctx, 8, &dto.UpdateAdminRoleRequest{
		// log:read is kept (already on the role); user:read is an escalation.
		Permissions: []string{permissions.LogRead.String(), permissions.UserRead.String()},
	})

	require.Nil(t, role)
	require.ErrorIs(t, err, cerrors.ErrForbidden)
}

func TestAdminRoleServiceUpdateAllowsKeepingPermissionsCallerDoesNotHold(t *testing.T) {
	setupLogger(t)

	logCh := make(chan *models.Log, 1)
	callerRoleID := uint(5)
	// The target role already holds user:read, which the caller does not hold.
	// Re-submitting the unchanged set grants nothing new, so a limited admin
	// can still maintain (rename etc.) a role broader than their own.
	currentPerms := []string{permissions.LogRead.String(), permissions.UserRead.String()}
	casbinClient := &casbinmocks.ClientMock{
		GetRolePermissionsFunc: func(roleID uint) []string {
			require.Equal(t, uint(8), roleID)
			return currentPerms
		},
		CheckPermissionWithRootFunc: func(string, *uint, string) (bool, error) {
			t.Fatal("no permission check should run when no permission is added")
			return false, nil
		},
		SetRolePermissionsFunc: func(roleID uint, perms []string) error {
			require.Equal(t, uint(8), roleID)
			require.Equal(t, currentPerms, perms)
			return nil
		},
	}
	repo := &adminrolemocks.AdminRoleRepositoryMock{
		FindByIDForUpdateFunc: func(ctx context.Context, id uint) (*models.AdminRole, error) {
			return &models.AdminRole{ID: 8, Name: "Manager"}, nil
		},
		UpdateFunc: func(context.Context, *models.AdminRole) error {
			return nil
		},
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}

	svc := service.NewAdminRoleService(repo, logRepo, casbinClient, passthroughTxManager())
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{
		UserID: 11, UserName: "Alice", Role: "admin", AdminRoleID: &callerRoleID,
	})

	name := "Supervisor"
	role, err := svc.Update(ctx, 8, &dto.UpdateAdminRoleRequest{
		Name:        &name,
		Permissions: currentPerms,
	})

	require.NoError(t, err)
	require.NotNil(t, role)
	select {
	case <-logCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestAdminRoleServiceCreateAllowsRootToGrantAnyPermission(t *testing.T) {
	setupLogger(t)

	logCh := make(chan *models.Log, 1)
	repo := &adminrolemocks.AdminRoleRepositoryMock{
		CreateFunc: func(ctx context.Context, role *models.AdminRole) error {
			role.ID = 8
			return nil
		},
	}
	casbinClient := &casbinmocks.ClientMock{
		CheckPermissionWithRootFunc: func(string, *uint, string) (bool, error) {
			t.Fatal("root must bypass the grant check entirely")
			return false, nil
		},
		AddRolePermissionsFunc: func(uint, []string) error { return nil },
		GetRolePermissionsFunc: func(uint) []string {
			return []string{permissions.UserRead.String()}
		},
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}

	svc := service.NewAdminRoleService(repo, logRepo, casbinClient, &txmocks.TransactionManagerMock{})
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{
		UserID: 1, UserName: "Root", Role: "root",
	})

	role, err := svc.Create(ctx, &dto.CreateAdminRoleRequest{
		Name:        "Manager",
		Permissions: []string{permissions.UserRead.String()},
	})

	require.NoError(t, err)
	require.NotNil(t, role)
	select {
	case <-logCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestAdminRoleServiceCreateReturnsRoleAndAuditLog(t *testing.T) {
	setupLogger(t)

	logCh := make(chan *models.Log, 1)
	repo := &adminrolemocks.AdminRoleRepositoryMock{
		CreateFunc: func(ctx context.Context, role *models.AdminRole) error {
			require.Equal(t, "req-2", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, "Manager", role.Name)
			require.True(t, role.IsActive)
			role.ID = 8
			return nil
		},
	}
	casbinClient := &casbinmocks.ClientMock{
		AddRolePermissionsFunc: func(roleID uint, perms []string) error {
			require.Equal(t, uint(8), roleID)
			require.Equal(t, []string{permissions.LogRead.String()}, perms)
			return nil
		},
		GetRolePermissionsFunc: func(roleID uint) []string {
			require.Equal(t, uint(8), roleID)
			return []string{permissions.LogRead.String()}
		},
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}

	svc := service.NewAdminRoleService(repo, logRepo, casbinClient, &txmocks.TransactionManagerMock{})
	ctx := utils.SetRequestIDToContext(context.Background(), "req-2")
	// Root caller: granting permissions requires the caller to hold them.
	ctx = utils.NewContextWithValues(ctx, utils.ContextValues{UserID: 11, UserName: "Alice", Role: "root"})

	role, err := svc.Create(ctx, &dto.CreateAdminRoleRequest{
		Name:        "Manager",
		Description: "Can manage products",
		Permissions: []string{permissions.LogRead.String()},
	})

	require.NoError(t, err)
	require.NotNil(t, role)
	require.Equal(t, uint(8), role.ID)
	require.Equal(t, []string{permissions.LogRead.String()}, role.Permissions)

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

func TestAdminRoleServiceUpdateUpdatesRoleInTransactionAndSyncsCasbinAfterCommit(t *testing.T) {
	setupLogger(t)

	logCh := make(chan *models.Log, 1)
	current := &models.AdminRole{ID: 8, Name: "Manager", Description: "old"}
	dbUpdated := false
	repo := &adminrolemocks.AdminRoleRepositoryMock{
		// Update must lock the row (FOR UPDATE) so concurrent writers cannot
		// interleave with the read→modify→save sequence.
		FindByIDForUpdateFunc: func(ctx context.Context, id uint) (*models.AdminRole, error) {
			require.Equal(t, uint(8), id)
			return current, nil
		},
		UpdateFunc: func(ctx context.Context, role *models.AdminRole) error {
			require.Same(t, current, role)
			require.Equal(t, "Supervisor", role.Name)
			require.Equal(t, "new", role.Description)
			dbUpdated = true
			return nil
		},
	}
	casbinClient := &casbinmocks.ClientMock{
		SetRolePermissionsFunc: func(roleID uint, perms []string) error {
			// The Casbin sync must happen only after the DB write inside the
			// transaction has completed.
			require.True(t, dbUpdated, "casbin sync must run after the DB update")
			require.Equal(t, uint(8), roleID)
			require.Equal(t, []string{permissions.LogRead.String()}, perms)
			return nil
		},
		GetRolePermissionsFunc: func(roleID uint) []string {
			require.Equal(t, uint(8), roleID)
			return []string{permissions.LogRead.String()}
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

	svc := service.NewAdminRoleService(repo, logRepo, casbinClient, txManager)
	// Root caller: granting permissions requires the caller to hold them.
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 11, UserName: "Alice", Role: "root"})

	name := "Supervisor"
	description := "new"
	role, err := svc.Update(ctx, 8, &dto.UpdateAdminRoleRequest{
		Name:        &name,
		Description: &description,
		Permissions: []string{permissions.LogRead.String()},
	})

	require.NoError(t, err)
	require.Same(t, current, role)
	require.Equal(t, 1, txCalls)
	require.Equal(t, []string{permissions.LogRead.String()}, role.Permissions)

	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionUpdate, entry.Action)
		require.Equal(t, models.LogEntityTypeAdminRole, entry.EntityType)
		require.Equal(t, uint(8), entry.EntityID)
		require.Equal(t, "Alice updated admin role: Supervisor", entry.Message)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestAdminRoleServiceUpdateReturnsErrorWhenCasbinSyncFails(t *testing.T) {
	setupLogger(t)

	casbinErr := errors.New("casbin unavailable")
	dbUpdated := false
	repo := &adminrolemocks.AdminRoleRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.AdminRole, error) {
			return &models.AdminRole{ID: 8, Name: "Manager"}, nil
		},
		UpdateFunc: func(context.Context, *models.AdminRole) error {
			dbUpdated = true
			return nil
		},
	}
	casbinClient := &casbinmocks.ClientMock{
		SetRolePermissionsFunc: func(uint, []string) error {
			return casbinErr
		},
	}

	svc := service.NewAdminRoleService(repo, &logmocks.LogRepositoryMock{}, casbinClient, passthroughTxManager())
	// Root caller: granting permissions requires the caller to hold them.
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 11, UserName: "Alice", Role: "root"})

	role, err := svc.Update(ctx, 8, &dto.UpdateAdminRoleRequest{
		Permissions: []string{permissions.LogRead.String()},
	})

	require.Nil(t, role)
	require.ErrorIs(t, err, casbinErr)
	// DB commit happens before the casbin sync, so the role row was updated
	// even though the permission sync failed (residual gap by design).
	require.True(t, dbUpdated)
}

func TestAdminRoleServiceDeleteDeletesRoleThenCasbinPolicies(t *testing.T) {
	setupLogger(t)

	logCh := make(chan *models.Log, 1)
	dbDeleted := false
	repo := &adminrolemocks.AdminRoleRepositoryMock{
		// Delete must use the locked find so the assigned-users check cannot
		// race a concurrent AssignAdminRole (which locks the same row).
		FindByIDForUpdateFunc: func(ctx context.Context, id uint) (*models.AdminRole, error) {
			require.Equal(t, uint(3), id)
			return &models.AdminRole{ID: 3, Name: "Manager"}, nil
		},
		CountUsersWithRoleFunc: func(ctx context.Context, roleID uint) (int64, error) {
			require.Equal(t, uint(3), roleID)
			return 0, nil
		},
		DeleteFunc: func(ctx context.Context, role *models.AdminRole) error {
			require.Equal(t, uint(3), role.ID)
			dbDeleted = true
			return nil
		},
	}
	casbinClient := &casbinmocks.ClientMock{
		DeleteRoleFunc: func(roleID uint) error {
			// Casbin cleanup must run only after the DB delete has committed.
			require.True(t, dbDeleted, "casbin cleanup must run after the DB delete")
			require.Equal(t, uint(3), roleID)
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

	svc := service.NewAdminRoleService(repo, logRepo, casbinClient, txManager)
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{UserID: 11, UserName: "Alice"})

	err := svc.Delete(ctx, 3)

	require.NoError(t, err)
	require.Equal(t, 1, txCalls)
	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionDelete, entry.Action)
		require.Equal(t, uint(3), entry.EntityID)
		require.Equal(t, "Alice deleted admin role: Manager", entry.Message)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestAdminRoleServiceDeleteSucceedsWhenCasbinCleanupFails(t *testing.T) {
	setupLogger(t)

	logCh := make(chan *models.Log, 1)
	repo := &adminrolemocks.AdminRoleRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.AdminRole, error) {
			return &models.AdminRole{ID: 3, Name: "Manager"}, nil
		},
		CountUsersWithRoleFunc: func(context.Context, uint) (int64, error) {
			return 0, nil
		},
		DeleteFunc: func(context.Context, *models.AdminRole) error {
			return nil
		},
	}
	casbinClient := &casbinmocks.ClientMock{
		DeleteRoleFunc: func(uint) error {
			return errors.New("casbin unavailable")
		},
	}
	logRepo := &logmocks.LogRepositoryMock{
		CreateFunc: func(ctx context.Context, entry *models.Log) error {
			logCh <- entry
			return nil
		},
	}

	svc := service.NewAdminRoleService(repo, logRepo, casbinClient, passthroughTxManager())

	// The role delete already committed; a failed casbin cleanup leaves only
	// inert orphaned policies, so the operation still reports success.
	err := svc.Delete(context.Background(), 3)

	require.NoError(t, err)
	select {
	case entry := <-logCh:
		require.Equal(t, models.LogActionDelete, entry.Action)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for audit log")
	}
}

func TestAdminRoleServiceDeleteRejectsAssignedUsers(t *testing.T) {
	setupLogger(t)

	repo := &adminrolemocks.AdminRoleRepositoryMock{
		FindByIDForUpdateFunc: func(context.Context, uint) (*models.AdminRole, error) {
			return &models.AdminRole{ID: 3, Name: "Manager"}, nil
		},
		CountUsersWithRoleFunc: func(context.Context, uint) (int64, error) {
			return 1, nil
		},
	}

	svc := service.NewAdminRoleService(repo, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, passthroughTxManager())

	err := svc.Delete(context.Background(), 3)

	require.Error(t, err)
	require.True(t, errors.Is(err, cerrors.ErrInvalidInput))
}

func TestAdminRoleServiceFindByIDReturnsRoleWithPermissions(t *testing.T) {
	setupLogger(t)

	repo := &adminrolemocks.AdminRoleRepositoryMock{
		FindByIDFunc: func(ctx context.Context, id uint, _ ...repository.Association) (*models.AdminRole, error) {
			require.Equal(t, "req-3", utils.GetRequestIDFromContext(ctx))
			require.Equal(t, uint(4), id)
			return &models.AdminRole{ID: 4, Name: "Support"}, nil
		},
	}
	casbinClient := &casbinmocks.ClientMock{
		GetRolePermissionsFunc: func(roleID uint) []string {
			require.Equal(t, uint(4), roleID)
			return []string{permissions.UserRead.String()}
		},
	}

	svc := service.NewAdminRoleService(repo, &logmocks.LogRepositoryMock{}, casbinClient, &txmocks.TransactionManagerMock{})
	ctx := utils.SetRequestIDToContext(context.Background(), "req-3")

	role, err := svc.FindByID(ctx, 4)

	require.NoError(t, err)
	require.NotNil(t, role)
	require.Equal(t, []string{permissions.UserRead.String()}, role.Permissions)
}

func TestAdminRoleServiceGetAllPermissionsReturnsFrontendMap(t *testing.T) {
	setupLogger(t)

	svc := service.NewAdminRoleService(&adminrolemocks.AdminRoleRepositoryMock{}, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, &txmocks.TransactionManagerMock{})

	got := svc.GetAllPermissions(context.Background())

	require.NotEmpty(t, got)
	require.Contains(t, got, permissions.ResourceAdminRole)
}

func TestAdminRoleServiceIndexReturnsRepositoryError(t *testing.T) {
	setupLogger(t)

	expectedErr := errors.New("find all failed")
	repo := &adminrolemocks.AdminRoleRepositoryMock{
		FindAllFunc: func(context.Context, *pagination.Pagination) ([]*models.AdminRole, error) {
			return nil, expectedErr
		},
		CountFunc: func(context.Context, *pagination.Pagination) (int64, error) {
			t.Fatal("Count should not be called when FindAll fails")
			return 0, nil
		},
	}

	svc := service.NewAdminRoleService(repo, &logmocks.LogRepositoryMock{}, &casbinmocks.ClientMock{}, &txmocks.TransactionManagerMock{})

	roles, meta, err := svc.Index(context.Background(), pagination.NewPagination(nil, nil, pagination.PaginationOptions{}))

	require.Nil(t, roles)
	require.Equal(t, response.Meta{}, meta)
	require.ErrorIs(t, err, expectedErr)
}
