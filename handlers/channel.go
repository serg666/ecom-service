package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/serg666/service-repository"

	"github.com/serg666/ecom-service/httputil"
)

type channelHandler struct {
	loggerFunc repository.LoggerFunc
	store      repository.ChannelRepository
}

// @BasePath /api/v1

// GetChannelsHandler godoc
// @Summary     List channels
// @Description Get channels
// @Tags        channels
// @Accept      json
// @Produce     json
// @Param       limit  query    int false "Maximum items to return"      default(100)
// @Param       offset query    int false "Return items from this point" default(0)
// @Success     200    {object} httputil.ChannelsResponse
// @Failure     400    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /channels     [get]
func (ch *channelHandler) GetChannelsHandler(c *gin.Context) {
	var req LimitAndOffsetRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, overall, channels := ch.store.Query(c, repository.NewChannelSpecificationWithLimitAndOffset(req.Limit, req.Offset))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	httputil.ChannelsList(c, http.StatusOK, overall, channels)
}

func NewChannelHandler(
	store repository.ChannelRepository,
	loggerFunc repository.LoggerFunc,
) *channelHandler {
	return &channelHandler{
		loggerFunc: loggerFunc,
		store:      store,
	}
}
