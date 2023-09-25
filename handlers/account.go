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

type CreateAccountRequest struct {
	IsEnabled                 *bool                `json:"is_enabled"`
	IsTest                    *bool                `json:"is_test"`
	RebillEnabled             *bool                `json:"rebill_enabled"`
	RefundEnabled             *bool                `json:"refund_enabled"`
	ReversalEnabled           *bool                `json:"reversal_enabled"`
	PartialConfirmEnabled     *bool                `json:"partial_confirm_enabled"`
	PartialReversalEnabled    *bool                `json:"partial_reversal_enabled"`
	PartialRefundEnabled      *bool                `json:"partial_refund_enabled"`
	CurrencyConversionEnabled *bool                `json:"currency_conversion_enabled"`
	CurrencyCode              *int                 `json:"currency_code" binding:"required"`
	Settings                  *ecom.AccountSettings `json:"settings"`
	Name                      *string              `json:"name" binding:"omitempty,notempty"`
}

type UpdateAccountRequest struct {
	IsEnabled                 *bool                `json:"is_enabled" binding:"required_without_all=IsTest RebillEnabled RefundEnabled ReversalEnabled PartialConfirmEnabled PartialReversalEnabled PartialRefundEnabled CurrencyConversionEnabled CurrencyCode Settings Name"`
	IsTest                    *bool                `json:"is_test" binding:"required_without_all=IsEnabled RebillEnabled RefundEnabled ReversalEnabled PartialConfirmEnabled PartialReversalEnabled PartialRefundEnabled CurrencyConversionEnabled CurrencyCode Settings Name"`
	RebillEnabled             *bool                `json:"rebill_enabled" binding:"required_without_all=IsEnabled IsTest RefundEnabled ReversalEnabled PartialConfirmEnabled PartialReversalEnabled PartialRefundEnabled CurrencyConversionEnabled CurrencyCode Settings Name"`
	RefundEnabled             *bool                `json:"refund_enabled" binding:"required_without_all=IsEnabled IsTest RebillEnabled ReversalEnabled PartialConfirmEnabled PartialReversalEnabled PartialRefundEnabled CurrencyConversionEnabled CurrencyCode Settings Name"`
	ReversalEnabled           *bool                `json:"reversal_enabled" binding:"required_without_all=IsEnabled IsTest RebillEnabled RefundEnabled PartialConfirmEnabled PartialReversalEnabled PartialRefundEnabled CurrencyConversionEnabled CurrencyCode Settings Name"`
	PartialConfirmEnabled     *bool                `json:"partial_confirm_enabled" binding:"required_without_all=IsEnabled IsTest RebillEnabled RefundEnabled ReversalEnabled PartialReversalEnabled PartialRefundEnabled CurrencyConversionEnabled CurrencyCode Settings Name"`
	PartialReversalEnabled    *bool                `json:"partial_reversal_enabled" binding:"required_without_all=IsEnabled IsTest RebillEnabled RefundEnabled ReversalEnabled PartialConfirmEnabled PartialRefundEnabled CurrencyConversionEnabled CurrencyCode Settings Name"`
	PartialRefundEnabled      *bool                `json:"partial_refund_enabled" binding:"required_without_all=IsEnabled IsTest RebillEnabled RefundEnabled ReversalEnabled PartialConfirmEnabled PartialReversalEnabled CurrencyConversionEnabled CurrencyCode Settings Name"`
	CurrencyConversionEnabled *bool                `json:"currency_conversion_enabled" binding:"required_without_all=IsEnabled IsTest RebillEnabled RefundEnabled ReversalEnabled PartialConfirmEnabled PartialReversalEnabled PartialRefundEnabled CurrencyCode Settings Name"`
	CurrencyCode              *int                 `json:"currency_code" binding:"required_without_all=IsEnabled IsTest RebillEnabled RefundEnabled ReversalEnabled PartialConfirmEnabled PartialReversalEnabled PartialRefundEnabled CurrencyConversionEnabled Settings Name"`
	Settings                  *ecom.AccountSettings `json:"settings" binding:"required_without_all=IsEnabled IsTest RebillEnabled RefundEnabled ReversalEnabled PartialConfirmEnabled PartialReversalEnabled PartialRefundEnabled CurrencyConversionEnabled CurrencyCode Name"`
	Name                      *string              `json:"name" binding:"omitempty,required_without_all=IsEnabled IsTest RebillEnabled RefundEnabled ReversalEnabled PartialConfirmEnabled PartialReversalEnabled PartialRefundEnabled CurrencyConversionEnabled CurrencyCode Settings,notempty"`
}

type accountHandler struct {
	loggerFunc    repository.LoggerFunc
	store         ecom.AccountRepository
	currencyStore repository.CurrencyRepository
}

// @BasePath /api/v1

// CreateAccountHandler godoc
// @Summary     Create an account
// @Description Add account by json
// @Tags        accounts
// @Accept      json
// @Produce     json
// @Param       account body     CreateAccountRequest true "Add account"
// @Success     200     {object} ecom.Account
// @Failure     400     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /accounts        [post]
func (ah *accountHandler) CreateAccountHandler(c *gin.Context) {
	// @note: set default values here
	isEnabled := true
	isTest := false
	rebillEnabled := false
	refundEnabled := true
	reversalEnabled := true
	partialConfirmEnabled := false
	partialReversalEnabled := false
	partialRefundEnabled := false
	currencyConversionEnabled := false
	req := CreateAccountRequest{
		IsEnabled:                 &isEnabled,
		IsTest:                    &isTest,
		RebillEnabled:             &rebillEnabled,
		RefundEnabled:             &refundEnabled,
		ReversalEnabled:           &reversalEnabled,
		PartialConfirmEnabled:     &partialConfirmEnabled,
		PartialReversalEnabled:    &partialReversalEnabled,
		PartialRefundEnabled:      &partialRefundEnabled,
		CurrencyConversionEnabled: &currencyConversionEnabled,
		Settings:                  &ecom.AccountSettings{},
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, _, currencies := ah.currencyStore.Query(c, repository.NewCurrencySpecificationByNumericCode(*req.CurrencyCode))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(currencies) == 0 {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Currency with CurrencyCode=%v not found", *req.CurrencyCode))
		return
	}

	account := &ecom.Account{
		IsEnabled:                 req.IsEnabled,
		IsTest:                    req.IsTest,
		RebillEnabled:             req.RebillEnabled,
		RefundEnabled:             req.RefundEnabled,
		ReversalEnabled:           req.ReversalEnabled,
		PartialConfirmEnabled:     req.PartialConfirmEnabled,
		PartialReversalEnabled:    req.PartialReversalEnabled,
		PartialRefundEnabled:      req.PartialRefundEnabled,
		CurrencyConversionEnabled: req.CurrencyConversionEnabled,
		Currency:                  currencies[0],
		Settings:                  req.Settings,
		Name:                      req.Name,
	}
	if err := ah.store.Add(c, account); err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, account)
}

// @BasePath /api/v1

// GetAccountsHandler godoc
// @Summary     List accounts
// @Description Get accounts
// @Tags        accounts
// @Accept      json
// @Produce     json
// @Param       limit  query    int false "Maximum items to return"      default(100)
// @Param       offset query    int false "Return items from this point" default(0)
// @Success     200    {object} httputil.AccountsResponse
// @Failure     400    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /accounts       [get]
func (ah *accountHandler) GetAccountsHandler(c *gin.Context) {
	var req LimitAndOffsetRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, overall, accounts := ah.store.Query(c, ecom.NewAccountSpecificationWithLimitAndOffset(
		req.Limit,
		req.Offset,
	))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	httputil.AccountsList(c, http.StatusOK, overall, accounts)
}

// @BasePath /api/v1

// GetAccountHandler godoc
// @Summary     Show an account
// @Description Get account by ID
// @Tags        accounts
// @Accept      json
// @Produce     json
// @Param       id     path     int true "Account ID" Format(int64)
// @Success     200    {object} ecom.Account
// @Failure     400    {object} httputil.HTTPError
// @Failure     404    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /accounts/{id}  [get]
func (ah *accountHandler) GetAccountHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, _, accounts := ah.store.Query(c, ecom.NewAccountSpecificationByID(id))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(accounts) == 0 {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Account with id=%d not found", id))
		return
	}

	c.JSON(http.StatusOK, accounts[0])
}

// @BasePath /api/v1

// DeleteAccountHandler godoc
// @Summary     Delete an account
// @Description Delete by account ID
// @Tags        accounts
// @Accept      json
// @Produce     json
// @Param       id     path     int true "Account ID" Format(int64)
// @Success     200    {object} ecom.Account
// @Failure     400    {object} httputil.HTTPError
// @Failure     404    {object} httputil.HTTPError
// @Failure     500    {object} httputil.HTTPError
// @Router      /accounts/{id}  [delete]
func (ah *accountHandler) DeleteAccountHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	account := &ecom.Account{
		Id: &id,
	}

	err, notfound := ah.store.Delete(c, account)

	if notfound {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Account with id=%d not found", id))
		return
	}

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, account)
}

// @BasePath /api/v1

// PatchAccountHandler godoc
// @Summary     Update an account
// @Description Update by account ID
// @Tags        accounts
// @Accept      json
// @Produce     json
// @Param       id      path     int                  true "Account ID" Format(int64)
// @Param       account body     UpdateAccountRequest true "Update account"
// @Success     200     {object} ecom.Account
// @Failure     400     {object} httputil.HTTPError
// @Failure     404     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /accounts/{id}   [patch]
func (ah *accountHandler) PatchAccountHandler(c *gin.Context) {
	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	id, err := strconv.Atoi(c.Params.ByName("id"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	var currency *repository.Currency

	if req.CurrencyCode != nil {
		err, _, currencies := ah.currencyStore.Query(c, repository.NewCurrencySpecificationByNumericCode(*req.CurrencyCode))

		if err != nil {
			httputil.NewError(c, http.StatusInternalServerError, err)
			return
		}

		if len(currencies) == 0 {
			httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Currency with CurrencyCode=%v not found", *req.CurrencyCode))
			return
		}
		currency = currencies[0]
	}

	account := &ecom.Account{
		Id:                        &id,
		IsEnabled:                 req.IsEnabled,
		IsTest:                    req.IsTest,
		RebillEnabled:             req.RebillEnabled,
		RefundEnabled:             req.RefundEnabled,
		ReversalEnabled:           req.ReversalEnabled,
		PartialConfirmEnabled:     req.PartialConfirmEnabled,
		PartialReversalEnabled:    req.PartialReversalEnabled,
		PartialRefundEnabled:      req.PartialRefundEnabled,
		CurrencyConversionEnabled: req.CurrencyConversionEnabled,
		Settings:                  req.Settings,
		Currency:                  currency,
		Name:                      req.Name,
	}

	err, notfound := ah.store.Update(c, account)

	if notfound {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Account with id=%d not found", id))
		return
	}

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, account)
}

func NewAccountHandler(
	store ecom.AccountRepository,
	currencyStore repository.CurrencyRepository,
	loggerFunc repository.LoggerFunc,
) *accountHandler {
	return &accountHandler{
		loggerFunc:    loggerFunc,
		store:         store,
		currencyStore: currencyStore,
	}
}
