package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/serg666/service-repository"

	"github.com/serg666/ecom-service/httputil"
)

type routerHandler struct {
	loggerFunc  repository.LoggerFunc
	routerStore repository.RouterRepository
}

// @BasePath /api/v1

// GetRoutersHandler godoc
// @Summary     List routers
// @Description Get routers
// @Tags        routers
// @Accept      json
// @Produce     json
// @Param       limit  query    int false "Maximum items to return"      default(100)
// @Param       offset query    int false "Return items from this point" default(0)
// @Success     200    {object} httputil.RoutersResponse
// @Failure     400    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /routers         [get]
func (rh *routerHandler) GetRoutersHandler(c *gin.Context) {
	var req LimitAndOffsetRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, overall, routers := rh.routerStore.Query(c, repository.NewRouterSpecificationWithLimitAndOffset(
		req.Limit,
		req.Offset,
	))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	httputil.RoutersList(c, http.StatusOK, overall, routers)
}

func NewRouterHandler(
	routerStore repository.RouterRepository,
	loggerFunc repository.LoggerFunc,
) *routerHandler {
	return &routerHandler{
		loggerFunc:  loggerFunc,
		routerStore: routerStore,
	}
}
