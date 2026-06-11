package errors_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	cerrors "github.com/PhantomX7/athleton/pkg/errors"
)

func TestAppErrorErrorPrefersWrappedError(t *testing.T) {
	err := cerrors.NewAppError(http.StatusBadRequest, "friendly message", errors.New("underlying cause"))

	require.Equal(t, "underlying cause", err.Error())
}

func TestAppErrorErrorFallsBackToMessage(t *testing.T) {
	err := cerrors.NewAppError(http.StatusBadRequest, "friendly message", nil)

	require.Equal(t, "friendly message", err.Error())
}

func TestAppErrorUnwrapSupportsErrorsIs(t *testing.T) {
	sentinel := errors.New("sentinel")
	err := cerrors.NewAppError(http.StatusInternalServerError, "wrapped", sentinel)

	require.ErrorIs(t, err, sentinel)
}

func TestAppErrorSupportsErrorsAsThroughWrapping(t *testing.T) {
	inner := cerrors.NewNotFoundError("post not found")
	wrapped := fmt.Errorf("handler: %w", inner)

	var ae *cerrors.AppError
	require.ErrorAs(t, wrapped, &ae)
	require.Equal(t, http.StatusNotFound, ae.Code)
	require.Equal(t, "post not found", ae.Message)
}

func TestConstructorsSetCodeAndSentinel(t *testing.T) {
	cases := []struct {
		name     string
		err      *cerrors.AppError
		code     int
		sentinel error
	}{
		{"not found", cerrors.NewNotFoundError("missing"), http.StatusNotFound, cerrors.ErrNotFound},
		{"bad request", cerrors.NewBadRequestError("invalid"), http.StatusBadRequest, cerrors.ErrInvalidInput},
		{"unauthorized", cerrors.NewUnauthorizedError("denied"), http.StatusUnauthorized, cerrors.ErrUnauthorized},
		{"forbidden", cerrors.NewForbiddenError("forbidden"), http.StatusForbidden, cerrors.ErrForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.code, tc.err.Code)
			require.ErrorIs(t, tc.err, tc.sentinel)
		})
	}
}

func TestNewInternalServerErrorWrapsCause(t *testing.T) {
	cause := errors.New("db down")
	err := cerrors.NewInternalServerError("something failed", cause)

	require.Equal(t, http.StatusInternalServerError, err.Code)
	require.ErrorIs(t, err, cause)
}

func TestIsNotFound(t *testing.T) {
	require.True(t, cerrors.IsNotFound(cerrors.NewNotFoundError("missing")))
	require.True(t, cerrors.IsNotFound(fmt.Errorf("repo: %w", cerrors.ErrNotFound)))
	require.False(t, cerrors.IsNotFound(cerrors.NewBadRequestError("invalid")))
	require.False(t, cerrors.IsNotFound(nil))
}
