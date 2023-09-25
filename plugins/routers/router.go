package routers

import (
	"github.com/gin-gonic/gin"

	ecom "github.com/serg666/service-repository/ecom"
)

type Router interface {
	Route(c *gin.Context, route *ecom.Route, request interface{}) error
}
