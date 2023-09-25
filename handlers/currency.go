package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/serg666/service-repository"

	"github.com/serg666/ecom-service/httputil"
)

type CreateCurrencyRequest struct {
	NumericCode *int    `json:"numericcode" binding:"required"`
	Name        *string `json:"name" binding:"required,notempty"`
	CharCode    *string `json:"charcode" binding:"required,notempty"`
	Exponent    *int    `json:"exponent"`
}

type UpdateCurrencyRequest struct {
	NumericCode *int    `json:"numericcode" binding:"required_without_all=Name CharCode Exponent"`
	Name        *string `json:"name" binding:"omitempty,required_without_all=NumericCode CharCode Exponent,notempty"`
	CharCode    *string `json:"charcode" binding:"omitempty,required_without_all=NumericCode Name Exponent,notempty"`
	Exponent    *int    `json:"exponent" binding:"required_without_all=NumericCode Name CharCode"`
}

type currencyHandler struct {
	loggerFunc repository.LoggerFunc
	store      repository.CurrencyRepository
}

// @BasePath /api/v1

// CreateCurrencyHandler godoc
// @Summary     Create a currency
// @Description Add currency by json
// @Tags        currencies
// @Accept      json
// @Produce     json
// @Param       currency body     CreateCurrencyRequest true "Add currency"
// @Success     200      {object} repository.Currency
// @Failure     400      {object} httputil.HTTPError
// @Failure     500      {object} httputil.HTTPError
// @Router      /currencies       [post]
func (ch *currencyHandler) CreateCurrencyHandler(c *gin.Context) {
	var req CreateCurrencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	currency := &repository.Currency{
		NumericCode: req.NumericCode,
		Name:        req.Name,
		CharCode:    req.CharCode,
		Exponent:    req.Exponent,
	}
	if err := ch.store.Add(c, currency); err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, currency)
}

// @BasePath /api/v1

// GetCurrenciesHandler godoc
// @Summary     List currencies
// @Description Get currencies
// @Tags        currencies
// @Accept      json
// @Produce     json
// @Param       limit  query    int false "Maximum items to return"      default(100)
// @Param       offset query    int false "Return items from this point" default(0)
// @Success     200    {object} httputil.CurrenciesResponse
// @Failure     400    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /currencies     [get]
func (ch *currencyHandler) GetCurrenciesHandler(c *gin.Context) {
	var req LimitAndOffsetRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, overall, currencies := ch.store.Query(c, repository.NewCurrencySpecificationWithLimitAndOffset(
		req.Limit,
		req.Offset,
	))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	httputil.CurrenciesList(c, http.StatusOK, overall, currencies)
}

// @BasePath /api/v1

// GetCurrencyHandler godoc
// @Summary     Show a currency
// @Description Get currency by ID
// @Tags        currencies
// @Accept      json
// @Produce     json
// @Param       id     path      int true "Currency ID" Format(int64)
// @Success     200    {object}  repository.Currency
// @Failure     400    {object}  httputil.HTTPError
// @Failure     404    {object}  httputil.HTTPError
// @Failure     500    {object}  httputil.HTTPError
// @Router      /currencies/{id} [get]
func (ch *currencyHandler) GetCurrencyHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, _, currencies := ch.store.Query(c, repository.NewCurrencySpecificationByID(id))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(currencies) == 0 {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Currency with id=%v not found", id))
		return
	}

	c.JSON(http.StatusOK, currencies[0])
}

// @BasePath /api/v1

// PatchCurrencyHandler godoc
// @Summary     Update a currency
// @Description Update by currency ID
// @Tags        currencies
// @Accept      json
// @Produce     json
// @Param       id       path     int                   true "Currency ID" Format(int64)
// @Param       currency body     UpdateCurrencyRequest true "Update currency"
// @Success     200      {object} repository.Currency
// @Failure     400      {object} httputil.HTTPError
// @Failure     404      {object} httputil.HTTPError
// @Failure     500      {object} httputil.HTTPError
// @Router      /currencies/{id}  [patch]
func (ch *currencyHandler) PatchCurrencyHandler(c *gin.Context) {
	var req UpdateCurrencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	currency := &repository.Currency{
		Id:          &id,
		NumericCode: req.NumericCode,
		Name:        req.Name,
		CharCode:    req.CharCode,
		Exponent:    req.Exponent,
	}

	err, notfound := ch.store.Update(c, currency)

	if notfound {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Currency with id=%d not found", id))
		return
	}

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, currency)
}

// @BasePath /api/v1

// DeleteCurrencyHandler godoc
// @Summary     Delete a currency
// @Description Delete by currency ID
// @Tags        currencies
// @Accept      json
// @Produce     json
// @Param       id     path      int true "Currency ID" Format(int64)
// @Success     200    {object}  repository.Currency
// @Failure     400    {object}  httputil.HTTPError
// @Failure     404    {object}  httputil.HTTPError
// @Failure     500    {object}  httputil.HTTPError
// @Router      /currencies/{id} [delete]
func (ch *currencyHandler) DeleteCurrencyHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	currency := &repository.Currency{Id: &id}

	err, notfound := ch.store.Delete(c, currency)

	if notfound {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Currency with id=%d not found", id))
		return
	}

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, currency)
}

func NewCurrencyHandler(
	store repository.CurrencyRepository,
	loggerFunc repository.LoggerFunc,
) *currencyHandler {
	return &currencyHandler{
		loggerFunc: loggerFunc,
		store:      store,
	}
}
