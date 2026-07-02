// Package ginx holds small Gin helpers shared across HTTP controllers.
package ginx

import (
	"fmt"
	"strconv"

	cerrors "github.com/PhantomX7/athleton/pkg/errors"

	"github.com/gin-gonic/gin"
)

// ParseUintParam reads an unsigned-integer path parameter (e.g. :id) and
// returns it as a uint. On a malformed value it records a public gin error —
// letting the error middleware shape the response — and returns ok=false, so
// the caller's whole error branch collapses to:
//
//	id, ok := ginx.ParseUintParam(ctx, "id")
//	if !ok {
//		return
//	}
//
// Centralizing this removes the strconv boilerplate (and the uint() cast) that
// every CRUD controller would otherwise repeat per id-bearing handler.
func ParseUintParam(ctx *gin.Context, name string) (uint, bool) {
	v, err := strconv.ParseUint(ctx.Param(name), 10, 32)
	if err != nil {
		// A malformed path param is the client's mistake: record a 400
		// AppError so the error middleware doesn't classify it as a 500.
		_ = ctx.Error(cerrors.NewBadRequestError(fmt.Sprintf("invalid %s parameter", name))).
			SetType(gin.ErrorTypePublic)
		return 0, false
	}
	return uint(v), true
}
