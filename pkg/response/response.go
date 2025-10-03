package response

import (
	"github.com/PhantomX7/go-starter/pkg/utils"
	"github.com/go-playground/validator/v10"
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
}

type ModelResponse interface {
	ToResponse() any
}

func BuildResponseSuccess(message string, data any) Response {
	res := Response{
		Status:  true,
		Message: message,
		Data:    data,
	}
	
	return res
}

func BuildResponseFailed(message string, err string) Response {
	res := Response{
		Status:  false,
		Message: message,
		Error:   err,
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

func BuildPaginationResponse[Data ModelResponse](data []Data, meta Meta) Response {
	res := Response{
		Status:  true,
		Message: "Success",
		Data: utils.Map(data, func(item Data) any {
			return item.ToResponse()
		}),
		Meta: meta,
	}

	return res
}
