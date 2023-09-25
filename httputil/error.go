package httputil

import "github.com/gin-gonic/gin"

type HTTPError struct {
	Message string `json:"message" example:"some error message"`
}

func NewError(ctx *gin.Context, status int, err error) {
	er := HTTPError{
		Message: err.Error(),
	}
	ctx.JSON(status, er)
}
