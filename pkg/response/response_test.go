package response_test

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"
)

type item struct {
	ID   uint
	Name string
}

type itemDTO struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

func (i item) ToResponse() itemDTO {
	return itemDTO{ID: i.ID, Name: i.Name}
}

func TestBuildResponseSuccess(t *testing.T) {
	res := response.BuildResponseSuccess("Created", map[string]string{"key": "value"})

	require.True(t, res.Status)
	require.Equal(t, "Created", res.Message)
	require.Equal(t, map[string]string{"key": "value"}, res.Data)
	require.Nil(t, res.Error)
	require.Nil(t, res.Meta)
}

func TestBuildResponseFailed(t *testing.T) {
	res := response.BuildResponseFailed("something went wrong")

	require.False(t, res.Status)
	require.Equal(t, "something went wrong", res.Message)
	require.Nil(t, res.Data)
}

func TestBuildResponseValidationError(t *testing.T) {
	type payload struct {
		Email string `validate:"required,email"`
	}

	err := validator.New().Struct(payload{})
	var ve validator.ValidationErrors
	require.ErrorAs(t, err, &ve)

	res := response.BuildResponseValidationError(ve)

	require.False(t, res.Status)
	require.Equal(t, "Validation failed", res.Message)

	formatted, ok := res.Error.(utils.ValidationErrorResponse)
	require.True(t, ok)
	require.Equal(t, 1, formatted.TotalErrors)
	require.Contains(t, formatted.Fields, "email")
}

func TestBuildPaginationResponseMapsModelsToDTOs(t *testing.T) {
	data := []item{{ID: 1, Name: "first"}, {ID: 2, Name: "second"}}
	meta := response.Meta{Limit: 20, Offset: 0, Total: 2}

	res := response.BuildPaginationResponse(data, meta)

	require.True(t, res.Status)
	require.Equal(t, "Success", res.Message)
	require.Equal(t, meta, res.Meta)
	require.Equal(t, []itemDTO{{ID: 1, Name: "first"}, {ID: 2, Name: "second"}}, res.Data)
}

func TestBuildPaginationResponseHandlesEmptySlice(t *testing.T) {
	res := response.BuildPaginationResponse([]item{}, response.Meta{Limit: 20})

	require.True(t, res.Status)
	require.Equal(t, []itemDTO{}, res.Data, "empty list must serialize as [] rather than null")
}
