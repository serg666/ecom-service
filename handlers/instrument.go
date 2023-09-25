package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/serg666/service-repository"

	"github.com/serg666/ecom-service/httputil"
)

type instrumentHandler struct {
	loggerFunc repository.LoggerFunc
	store      repository.InstrumentRepository
}

// @BasePath /api/v1

// GetInstrumentsHandler godoc
// @Summary     List instruments
// @Description Get instruments
// @Tags        instruments
// @Accept      json
// @Produce     json
// @Param       limit  query    int false "Maximum items to return"      default(100)
// @Param       offset query    int false "Return items from this point" default(0)
// @Success     200    {object} httputil.InstrumentsResponse
// @Failure     400    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /instruments     [get]
func (ch *instrumentHandler) GetInstrumentsHandler(c *gin.Context) {
	var req LimitAndOffsetRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, overall, instruments := ch.store.Query(c, repository.NewInstrumentSpecificationWithLimitAndOffset(req.Limit, req.Offset))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	httputil.InstrumentsList(c, http.StatusOK, overall, instruments)
}

func NewInstrumentHandler(
	store repository.InstrumentRepository,
	loggerFunc repository.LoggerFunc,
) *instrumentHandler {
	return &instrumentHandler{
		loggerFunc: loggerFunc,
		store:      store,
	}
}
