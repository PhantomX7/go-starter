package response

import (
	"github.com/go-playground/validator/v10"

	"github.com/PhantomX7/athleton/pkg/utils"
)

type Response struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Error   any    `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
}

// Response structures
type Meta struct {
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Total  int64 `json:"total"`
	Facet  any   `json:"facet,omitempty"`
}

type ModelResponse[T any] interface {
	ToResponse() T
}

func BuildResponseSuccess(message string, data any) Response {
	res := Response{
		Status:  true,
		Message: message,
		Data:    data,
	}

	return res
}

func BuildResponseFailed(message string) Response {
	res := Response{
		Status:  false,
		Message: message,
	}

	return res
}

func BuildResponseValidationError(err validator.ValidationErrors) Response {
	res := Response{
		Status:  false,
		Message: "Validation failed",
		Error:   utils.FormatValidationErrors(err),
	}

	return res
}

func BuildPaginationResponse[Data ModelResponse[T], T any](data []Data, meta Meta) Response {
	res := Response{
		Status:  true,
		Message: "Success",
		Data: utils.Map(data, func(item Data) T {
			return item.ToResponse()
		}),
		Meta: meta,
	}

	return res
}
