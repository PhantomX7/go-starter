package utils_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/utils"
)

func TestValuesFromContextRoundTrip(t *testing.T) {
	roleID := uint(3)
	values := utils.ContextValues{
		UserID:      7,
		UserName:    "alice",
		Role:        "admin",
		AdminRoleID: &roleID,
		RequestID:   "req-1",
	}

	ctx := utils.NewContextWithValues(context.Background(), values)

	got, err := utils.ValuesFromContext(ctx)
	require.NoError(t, err)
	require.Equal(t, values, *got)
}

func TestValuesFromContextRejectsMissingValues(t *testing.T) {
	got, err := utils.ValuesFromContext(context.Background())

	require.Nil(t, got)
	require.Error(t, err)
	require.ErrorIs(t, err, cerrors.ErrInvalidInput)
}

func TestIndividualGettersReturnStoredValues(t *testing.T) {
	roleID := uint(3)
	ctx := utils.NewContextWithValues(context.Background(), utils.ContextValues{
		UserID:      7,
		UserName:    "alice",
		Role:        "admin",
		AdminRoleID: &roleID,
	})

	userID, ok := utils.GetUserIDFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, uint(7), userID)

	userName, ok := utils.GetUserNameFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "alice", userName)

	role, ok := utils.GetRoleFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "admin", role)

	require.Equal(t, &roleID, utils.GetAdminRoleIDFromContext(ctx))
}

func TestIndividualGettersReportMissingValues(t *testing.T) {
	ctx := context.Background()

	_, ok := utils.GetUserIDFromContext(ctx)
	require.False(t, ok)

	_, ok = utils.GetUserNameFromContext(ctx)
	require.False(t, ok)

	_, ok = utils.GetRoleFromContext(ctx)
	require.False(t, ok)

	require.Nil(t, utils.GetAdminRoleIDFromContext(ctx))
}

func TestRequestIDRoundTrip(t *testing.T) {
	ctx := utils.SetRequestIDToContext(context.Background(), "req-42")

	require.Equal(t, "req-42", utils.GetRequestIDFromContext(ctx))
	require.Empty(t, utils.GetRequestIDFromContext(context.Background()))
}

func TestTxRoundTrip(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	ctx := utils.SetTxToContext(context.Background(), db)

	require.Same(t, db, utils.GetTxFromContext(ctx))
	require.Nil(t, utils.GetTxFromContext(context.Background()))
}

func TestStripTxClearsInheritedTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	ctx := utils.SetTxToContext(context.Background(), db)
	stripped := utils.StripTx(ctx)

	// The stripped context must not expose the parent's transaction...
	require.Nil(t, utils.GetTxFromContext(stripped))
	// ...while the parent context keeps it (contexts are immutable).
	require.Same(t, db, utils.GetTxFromContext(ctx))

	// Stripping a context that never carried a tx stays nil.
	require.Nil(t, utils.GetTxFromContext(utils.StripTx(context.Background())))

	// Non-tx values survive the strip: only the transaction slot is cleared.
	withReq := utils.SetRequestIDToContext(ctx, "req-9")
	require.Equal(t, "req-9", utils.GetRequestIDFromContext(utils.StripTx(withReq)))
}

func TestMapTransformsSlice(t *testing.T) {
	got := utils.Map([]int{1, 2, 3}, func(n int) int { return n * 2 })
	require.Equal(t, []int{2, 4, 6}, got)

	require.Equal(t, []string{}, utils.Map([]int{}, func(n int) string { return "" }))
}
