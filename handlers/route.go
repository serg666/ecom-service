package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/httputil"
)

type CreateRouteRequest struct {
	Profile    *int                `json:"profile_id" binding:"required"`
	Instrument *string             `json:"instrument_key" binding:"required,notempty"`
	Account    *int                `json:"account_id" binding:"required"`
	Channel    *int                `json:"channel_id" binding:"required"`
	Router     *string             `json:"router_key" binding:"omitempty,notempty"`
	Settings   *ecom.RouterSettings `json:"settings"`
}

type UpdateRouteRequest struct {
	Profile    *int                `json:"profile_id" binding:"required_without_all=Instrument Account Channel Router Settings"`
	Instrument *string             `json:"instrument_key" binding:"omitempty,required_without_all=Profile Account Channel Router Settings,notempty"`
	Account    *int                `json:"account_id" binding:"required_without_all=Profile Instrument Channel Router Settings"`
	Channel    *int                `json:"channel_id" binding:"required_without_all=Profile Instrument Account Router Settings"`
	Router     *string             `json:"router_key" binding:"required"`
	Settings   *ecom.RouterSettings `json:"settings" binding:"required_without_all=Profile Instrument Account Channel Router"`
}

type routeHandler struct {
	loggerFunc      repository.LoggerFunc
	routeStore      ecom.RouteRepository
	profileStore    repository.ProfileRepository
	instrumentStore repository.InstrumentRepository
	accountStore    ecom.AccountRepository
	routerStore     repository.RouterRepository
	currencyStore   repository.CurrencyRepository
	channelStore    repository.ChannelRepository
}

// @BasePath /api/v1

// CreateRouteHandler godoc
// @Summary     Create a route
// @Description Add route by json
// @Tags        routes
// @Accept      json
// @Produce     json
// @Param       route   body     CreateRouteRequest true "Add route"
// @Success     200     {object} ecom.Route
// @Failure     400     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /routes [post]
func (rh *routeHandler) CreateRouteHandler(c *gin.Context) {
	var req CreateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, _, profiles := rh.profileStore.Query(c, repository.NewProfileSpecificationByID(*req.Profile))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(profiles) == 0 {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Profile with id %d not found", *req.Profile))
		return
	}

	err, _, instruments := rh.instrumentStore.Query(c, repository.NewInstrumentSpecificationByKey(*req.Instrument))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(instruments) == 0 {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Instrument with key %s not found", *req.Instrument))
		return
	}

	var account *ecom.Account
	var channel *repository.Channel
	var router *repository.Router

	if req.Account != nil {
		err, _, accounts := rh.accountStore.Query(c, ecom.NewAccountSpecificationByID(*req.Account))

		if err != nil {
			httputil.NewError(c, http.StatusInternalServerError, err)
			return
		}

		if len(accounts) == 0 {
			httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Account with ID=%d not found", *req.Account))
			return
		}

		account = accounts[0]
	}

	if req.Channel != nil {
		err, _, channels := rh.channelStore.Query(c, repository.NewChannelSpecificationByID(*req.Channel))

		if err != nil {
			httputil.NewError(c, http.StatusInternalServerError, err)
			return
		}

		if len(channels) == 0 {
			httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Channel with ID=%d not found", *req.Channel))
			return
		}

		channel = channels[0]
	}

	if req.Router != nil {
		err, _, routers := rh.routerStore.Query(c, repository.NewRouterSpecificationByKey(*req.Router))

		if err != nil {
			httputil.NewError(c, http.StatusInternalServerError, err)
			return
		}

		if len(routers) == 0 {
			httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Router with key=%s not found", *req.Router))
			return
		}

		router = routers[0]
	}

	route := &ecom.Route{
		Profile:    profiles[0],
		Instrument: instruments[0],
		Account:    account,
		Channel:    channel,
		Router:     router,
		Settings:   req.Settings,
	}
	if err := rh.routeStore.Add(c, route); err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, route)
}

// @BasePath /api/v1

// GetRoutesHandler godoc
// @Summary     List routes
// @Description Get routes
// @Tags        routes
// @Accept      json
// @Produce     json
// @Param       limit  query    int false "Maximum items to return"      default(100)
// @Param       offset query    int false "Return items from this point" default(0)
// @Success     200    {object} httputil.RoutesResponse
// @Failure     400    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /routes         [get]
func (rh *routeHandler) GetRoutesHandler(c *gin.Context) {
	var req LimitAndOffsetRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, overall, routes := rh.routeStore.Query(c, ecom.NewRouteSpecificationWithLimitAndOffset(
		req.Limit,
		req.Offset,
	))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	httputil.RoutesList(c, http.StatusOK, overall, routes)
}

// @BasePath /api/v1

// GetRouteHandler godoc
// @Summary     Show a route
// @Description Get route by ID
// @Tags        routes
// @Accept      json
// @Produce     json
// @Param       id     path     int true "Route ID" Format(int64)
// @Success     200    {object} ecom.Route
// @Failure     400    {object} httputil.HTTPError
// @Failure     404    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /routes/{id}    [get]
func (rh *routeHandler) GetRouteHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, _, routes := rh.routeStore.Query(c, ecom.NewRouteSpecificationByID(id))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(routes) == 0 {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Route with id=%d not found", id))
		return
	}

	c.JSON(http.StatusOK, routes[0])
}

// @BasePath /api/v1

// DeleteRouteHandler godoc
// @Summary     Delete a route
// @Description Delete by route ID
// @Tags        routes
// @Accept      json
// @Produce     json
// @Param       id     path     int true "Route ID" Format(int64)
// @Success     200    {object} ecom.Route
// @Failure     400    {object} httputil.HTTPError
// @Failure     404    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /routes/{id}    [delete]
func (rh *routeHandler) DeleteRouteHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	route := &ecom.Route{
		Id: &id,
	}

	err, notfound := rh.routeStore.Delete(c, route)

	if notfound {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Route with id=%d not found", id))
		return
	}

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, route)
}

// @BasePath /api/v1

// PatchRouteHandler godoc
// @Summary     Update a route
// @Description Update by route ID
// @Tags        routes
// @Accept      json
// @Produce     json
// @Param       id      path     int                true "Route ID" Format(int64)
// @Param       route   body     UpdateRouteRequest true "Update route"
// @Success     200     {object} ecom.Route
// @Failure     400     {object} httputil.HTTPError
// @Failure     404     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /routes/{id}     [patch]
func (rh *routeHandler) PatchRouteHandler(c *gin.Context) {
	var req UpdateRouteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	var profile *repository.Profile
	var instrument *repository.Instrument
	var account *ecom.Account
	var channel *repository.Channel
	var router *repository.Router

	if req.Profile != nil {
		err, _, profiles := rh.profileStore.Query(c, repository.NewProfileSpecificationByID(*req.Profile))

		if err != nil {
			httputil.NewError(c, http.StatusInternalServerError, err)
			return
		}

		if len(profiles) == 0 {
			httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Profile with id=%d not found", *req.Profile))
			return
		}
		profile = profiles[0]
	}

	if req.Instrument != nil {
		err, _, instruments := rh.instrumentStore.Query(c, repository.NewInstrumentSpecificationByKey(*req.Instrument))

		if err != nil {
			httputil.NewError(c, http.StatusInternalServerError, err)
			return
		}

		if len(instruments) == 0 {
			httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Instrument with key=%v not found", *req.Instrument))
			return
		}
		instrument = instruments[0]
	}

	if req.Account != nil {
		err, _, accounts := rh.accountStore.Query(c, ecom.NewAccountSpecificationByID(*req.Account))

		if err != nil {
			httputil.NewError(c, http.StatusInternalServerError, err)
			return
		}

		if len(accounts) == 0 {
			httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Account with ID=%v not found", *req.Account))
			return
		}
		account = accounts[0]
	}

	if req.Channel != nil {
		err, _, channels := rh.channelStore.Query(c, repository.NewChannelSpecificationByID(*req.Channel))

		if err != nil {
			httputil.NewError(c, http.StatusInternalServerError, err)
			return
		}

		if len(channels) == 0 {
			httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Channel with ID=%v not found", *req.Channel))
			return
		}
		channel = channels[0]
	}

	if req.Router != nil && *req.Router != "" {
		err, _, routers := rh.routerStore.Query(c, repository.NewRouterSpecificationByKey(*req.Router))

		if err != nil {
			httputil.NewError(c, http.StatusInternalServerError, err)
			return
		}

		if len(routers) == 0 {
			httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Router with key=%v not found", *req.Router))
			return
		}
		router = routers[0]
	}

	route := &ecom.Route{
		Id:         &id,
		Profile:    profile,
		Instrument: instrument,
		Account:    account,
		Channel:    channel,
		Router:     router,
		Settings:   req.Settings,
	}

	err, notfound := rh.routeStore.Update(c, route)

	if notfound {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Route with id=%d not found", id))
		return
	}

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, route)
}

func NewRouteHandler(
	routeStore ecom.RouteRepository,
	profileStore repository.ProfileRepository,
	instrumentStore repository.InstrumentRepository,
	accountStore ecom.AccountRepository,
	routerStore repository.RouterRepository,
	currencyStore repository.CurrencyRepository,
	channelStore repository.ChannelRepository,
	loggerFunc repository.LoggerFunc,
) *routeHandler {
	return &routeHandler{
		loggerFunc:      loggerFunc,
		routeStore:      routeStore,
		profileStore:    profileStore,
		instrumentStore: instrumentStore,
		accountStore:    accountStore,
		routerStore:     routerStore,
		currencyStore:   currencyStore,
		channelStore:    channelStore,
	}
}
