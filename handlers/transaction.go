package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	repository "github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/config"
	"github.com/serg666/ecom-service/httputil"
	"github.com/serg666/ecom-service/plugins"
	"github.com/serg666/ecom-service/plugins/channels"
	"github.com/serg666/ecom-service/plugins/instruments"
	bankcard "github.com/serg666/ecom-service/plugins/instruments/card"
	"github.com/serg666/ecom-service/validators"
)

type LimitAndOffsetWithFiltersRequest struct {
	OrderId        string `form:"order_id"`
	ProfileId      int    `form:"profile_id"`
	ProfileIds     string `form:"profile_ids"`
	TransactionIds string `form:"transaction_ids"`
	RRN            string `form:"rrn"`
	AuthCode       string `form:"authcode"`
	RemoteId       string `form:"remote_id"`
	CardMask       string `form:"card_mask"`
	Limit          int    `form:"limit,default=100"`
	Offset         int    `form:"offset,default=0"`
}

func (r LimitAndOffsetWithFiltersRequest) GetProfileIds() []string {
	profileIds := []string{}
	if r.ProfileIds != "" {
		profileIds = strings.Split(r.ProfileIds, ",")
	}
	if r.ProfileId > 0 {
		profileIds = append(profileIds, fmt.Sprintf("%d", r.ProfileId))
	}
	return profileIds
}

func (r LimitAndOffsetWithFiltersRequest) GetTransactionIds() []string {
	transactionIds := []string{}
	if r.TransactionIds != "" {
		transactionIds = strings.Split(r.TransactionIds, ",")
	}

	return transactionIds
}

type transactionHandler struct {
	cfg              *config.Config
	loggerFunc       repository.LoggerFunc
	profileStore     repository.ProfileRepository
	accountStore     ecom.AccountRepository
	currencyStore    repository.CurrencyRepository
	channelStore     repository.ChannelRepository
	instrumentStore  repository.InstrumentRepository
	routeStore       ecom.RouteRepository
	routerStore      repository.RouterRepository
	cardStore        repository.CardRepository
	cardMaskStore    repository.CardMaskRepository
	transactionStore ecom.TransactionRepository
	sessionStore     repository.SessionRepository
	rateStore        repository.RateRepository
}

func (th *transactionHandler) route(
	c *gin.Context,
	profile *repository.Profile,
	instrument *repository.Instrument,
	instrumentStore interface{},
	instrumentRequester instruments.InstrumentRequesterFunc,
	request interface{},
) (error, *ecom.Route) {
	err, _, routes := th.routeStore.Query(c, ecom.NewRouteSpecificationByProfileAndInstrument(
		profile,
		instrument,
	))

	if err != nil {
		return fmt.Errorf("Error quering route by profile and instrument: %v", err), nil
	}

	if len(routes) == 0 {
		return fmt.Errorf("Route by profile and instrument not found: %v", err), nil
	}

	route := routes[0]

	if route.Router != nil {
		err, routerApi := plugins.RouterApi(route, th.accountStore, th.channelStore, instrumentStore, instrumentRequester, th.loggerFunc)
		if err != nil {
			return fmt.Errorf("Can not get router: %v", err), nil
		}

		if err := routerApi.Route(c, route, request); err != nil {
			return fmt.Errorf("Can not get route: %v", err), nil
		}
	}

	return nil, route
}

func (th *transactionHandler) validate(c *gin.Context) (error, *ecom.Transaction, channels.BankChannel) {
	pid, err := strconv.Atoi(c.Params.ByName("pid"))
	if err != nil {
		return fmt.Errorf("invalid profile id: %v", err), nil, nil
	}

	err, _, profiles := th.profileStore.Query(c, repository.NewProfileSpecificationByID(pid))

	if err != nil {
		return fmt.Errorf("faild to query profile store: %v", err), nil, nil
	}

	if len(profiles) == 0 {
		return fmt.Errorf("profile with id=%v not found", pid), nil, nil
	}

	profile := profiles[0]

	tid, err := strconv.Atoi(c.Params.ByName("tid"))
	if err != nil {
		return fmt.Errorf("invalid transaction id: %v", err), nil, nil
	}

	err, _, transactions := th.transactionStore.Query(c, ecom.NewTransactionSpecificationByID(tid))

	if err != nil {
		return fmt.Errorf("faild to query transaction store: %v", err), nil, nil
	}

	if len(transactions) == 0 {
		return fmt.Errorf("transaction with id=%v not found", tid), nil, nil
	}

	transaction := transactions[0]

	if *transaction.Profile.Id != *profile.Id {
		return fmt.Errorf("incorrect transaction id: %v", tid), nil, nil
	}

	var instrumentStore interface{}

	// @note: depends on instrument type
	switch *transaction.Instrument.Key {
	case "card":
		instrumentStore = bankcard.CardInstrumentStore{
			CardRepository:     th.cardStore,
			CardMaskRepository: th.cardMaskStore,
		}
	}

	err, instrumentApi := plugins.InstrumentApi(
		transaction.Instrument,
		instrumentStore,
		th.loggerFunc,
	)
	if err != nil {
		return fmt.Errorf("failed to get instrument api: %v", err), nil, nil
	}

	transaction.InstrumentInstance = instruments.NewInstrumentInstance(c, instrumentApi, transaction.InstrumentId)

	err, bankApi := plugins.BankApi(th.cfg, transaction.Account, transaction.Channel, transaction.Instrument, instrumentStore, th.sessionStore, th.transactionStore, th.loggerFunc)
	if err != nil {
		return fmt.Errorf("faild to get bank api: %v", err), nil, nil
	}

	return nil, transaction, bankApi
}

// @BasePath /api/v1

// ProcessParesHandler godoc
// @Summary     Process PaRes for transaction
// @Description Make authorization request for transaction, when return from ACS with PaRes
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid     path     int true "Profile ID" Format(int64)
// @Param       tid     path     int true "Transaction ID" Format(int64)
// @Param       request body     validators.ProcessParesRequest true "Process PaRes"
// @Success     200     {object} ecom.Transaction
// @Failure     400     {object} httputil.HTTPError
// @Router      /profiles/{pid}/transactions/{tid}/processpares [post]
func (th *transactionHandler) ProcessParesHandler(c *gin.Context) {
	var req validators.ProcessParesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, transaction, bankApi := th.validate(c)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	if !transaction.Is3DSWaiting() {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Transaction has wrong state: %s", *transaction.Status))
		return
	}

	th.loggerFunc(c).Printf("using account: %v", transaction.Account)

	if err := bankApi.ProcessPares(c, transaction, req.Pares); err != nil {
		mess := err.Error()
		transaction.Declined(&mess)
	}

	if err, notfound := th.transactionStore.Update(c, transaction); err != nil {
		th.loggerFunc(c).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
	}

	c.JSON(http.StatusOK, transaction)
}

// @BasePath /api/v1

// ProcessCresHandler godoc
// @Summary     Process CRES for transaction
// @Description Make authorization request for transaction, when return from ACS with CRES
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid     path     int true "Profile ID" Format(int64)
// @Param       tid     path     int true "Transaction ID" Format(int64)
// @Param       request body     validators.ProcessCresRequest true "Process CRES"
// @Success     200     {object} ecom.Transaction
// @Failure     400     {object} httputil.HTTPError
// @Router      /profiles/{pid}/transactions/{tid}/processcres [post]
func (th *transactionHandler) ProcessCresHandler(c *gin.Context) {
	var req validators.ProcessCresRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, transaction, bankApi := th.validate(c)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	if !transaction.Is3DSWaiting() {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Transaction has wrong state: %s", *transaction.Status))
		return
	}

	th.loggerFunc(c).Printf("using account: %v", transaction.Account)

	if err := bankApi.ProcessCres(c, transaction, req.Cres); err != nil {
		mess := err.Error()
		transaction.Declined(&mess)
	}

	if err, notfound := th.transactionStore.Update(c, transaction); err != nil {
		th.loggerFunc(c).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
	}

	c.JSON(http.StatusOK, transaction)
}

// @BasePath /api/v1

// UpdateTransactionHandler godoc
// @Summary     Update transaction
// @Description Sync transaction info with bank
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid     path     int true "Profile ID" Format(int64)
// @Param       tid     path     int true "Transaction ID" Format(int64)
// @Success     200     {object} ecom.Transaction
// @Failure     400     {object} httputil.HTTPError
// @Router      /profiles/{pid}/transactions/{tid}/update [post]
func (th *transactionHandler) UpdateTransactionHandler(c *gin.Context) {
	err, transaction, bankApi := th.validate(c)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	th.loggerFunc(c).Printf("using account: %v", transaction.Account)

	bankApi.Update(c, transaction, true)

	if err, notfound := th.transactionStore.Update(c, transaction); err != nil {
		th.loggerFunc(c).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
	}

	c.JSON(http.StatusOK, transaction)
}

// @BasePath /api/v1

// CompleteMethodUrlHandler godoc
// @Summary     Complete MethodUrl for transaction
// @Description Make request when hidden frame has been loaded or has not been loaded
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid     path     int true "Profile ID" Format(int64)
// @Param       tid     path     int true "Transaction ID" Format(int64)
// @Param       request body     validators.CompleteMethodUrlRequest true "Complete MethodUrl"
// @Success     200     {object} ecom.Transaction
// @Failure     400     {object} httputil.HTTPError
// @Router      /profiles/{pid}/transactions/{tid}/completemethodurl [post]
func (th *transactionHandler) CompleteMethodUrlHandler(c *gin.Context) {
	var req validators.CompleteMethodUrlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, transaction, bankApi := th.validate(c)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	if !transaction.IsMethodUrlWaiting() {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Transaction has wrong state: %s", *transaction.Status))
		return
	}

	th.loggerFunc(c).Printf("using account: %v", transaction.Account)

	if err := bankApi.CompleteMethodUrl(c, transaction, *req.Completed); err != nil {
		mess := err.Error()
		transaction.Declined(&mess)
	}

	if err, notfound := th.transactionStore.Update(c, transaction); err != nil {
		th.loggerFunc(c).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
	}

	c.JSON(http.StatusOK, transaction)
}

// @BasePath /api/v1

// ReverseHandler godoc
// @Summary     Reverse transaction
// @Description Make reverse transaction
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid     path     int true "Profile ID" Format(int64)
// @Param       tid     path     int true "Transaction ID" Format(int64)
// @Param       request body     validators.ReversalRequest true "Reverse transaction"
// @Success     200     {object} ecom.Transaction
// @Failure     400     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /profiles/{pid}/transactions/{tid}/reverse [post]
func (th *transactionHandler) ReverseHandler(c *gin.Context) {
	var req validators.ReversalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, transaction, bankApi := th.validate(c)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	if !transaction.IsSuccess() {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Transaction has wrong state: %s", *transaction.Status))
		return
	}

	if !transaction.IsPreAuth() {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Transaction has wrong type: %s", *transaction.Type))
		return
	}

	if req.TotalAmount() > transaction.TotalAmount() {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Incorrect amount: %d", req.TotalAmount()))
		return
	}

	if req.TotalAmount() < transaction.TotalAmount() && !*transaction.Account.PartialReversalEnabled {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Incorrect amount: %d, partial reversal not allowed", req.TotalAmount()))
		return
	}

	if !*transaction.Account.ReversalEnabled {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Reversal not allowed"))
		return
	}

	th.loggerFunc(c).Printf("using account: %v", transaction.Account)

	err, newTransaction := ecom.NewTransaction(c, transaction.InstrumentInstance,
		repository.REVERSAL,
		th.rateStore,
		transaction.OrderId,
		nil,
		transaction.Profile,
		transaction.Account,
		transaction.Channel,
		transaction.Instrument,
		transaction.InstrumentId,
		&req.Amount,
		&req.Commission,
		transaction.Customer,
		transaction,
		nil,
	)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := th.transactionStore.Add(c, newTransaction); err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := bankApi.Reverse(c, newTransaction); err != nil {
		mess := err.Error()
		newTransaction.Declined(&mess)
	}

	if err, notfound := th.transactionStore.Update(c, newTransaction); err != nil {
		th.loggerFunc(c).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
	}

	c.JSON(http.StatusOK, newTransaction)
}

// @BasePath /api/v1

// RefundHandler godoc
// @Summary     Refund transaction
// @Description Make refund transaction
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid     path     int true "Profile ID" Format(int64)
// @Param       tid     path     int true "Transaction ID" Format(int64)
// @Param       request body     validators.RefundRequest true "Refund transaction"
// @Success     200     {object} ecom.Transaction
// @Failure     400     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /profiles/{pid}/transactions/{tid}/refund [post]
func (th *transactionHandler) RefundHandler(c *gin.Context) {
	var req validators.RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, transaction, bankApi := th.validate(c)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	if !transaction.IsSuccess() {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Transaction has wrong state: %s", *transaction.Status))
		return
	}

	if !(transaction.IsAuth() || transaction.IsRebill() || transaction.IsConfirm()) {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Transaction has wrong type: %s", *transaction.Type))
		return
	}

	if req.TotalAmount() > transaction.TotalAmount() {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Incorrect amount: %d", req.TotalAmount()))
		return
	}

	if req.TotalAmount() < transaction.TotalAmount() && !*transaction.Account.PartialRefundEnabled {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Incorrect amount: %d, partial refund not allowed", req.TotalAmount()))
		return
	}

	if !*transaction.Account.RefundEnabled {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Refund not allowed"))
		return
	}

	th.loggerFunc(c).Printf("using account: %v", transaction.Account)

	err, newTransaction := ecom.NewTransaction(c, transaction.InstrumentInstance,
		repository.REFUND,
		th.rateStore,
		transaction.OrderId,
		nil,
		transaction.Profile,
		transaction.Account,
		transaction.Channel,
		transaction.Instrument,
		transaction.InstrumentId,
		&req.Amount,
		&req.Commission,
		transaction.Customer,
		transaction,
		nil,
	)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := th.transactionStore.Add(c, newTransaction); err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := bankApi.Refund(c, newTransaction); err != nil {
		mess := err.Error()
		newTransaction.Declined(&mess)
	}

	if err, notfound := th.transactionStore.Update(c, newTransaction); err != nil {
		th.loggerFunc(c).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
	}

	c.JSON(http.StatusOK, newTransaction)
}

// @BasePath /api/v1

// GetTransactionHandler godoc
// @Summary     Show a transaction
// @Description Get profile transaction by ID
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid    path                        int true "Profile ID"     Format(int64)
// @Param       tid    path                        int true "Transaction ID" Format(int64)
// @Success     200    {object}                    ecom.Transaction
// @Failure     400    {object}                    httputil.HTTPError
// @Router      /profiles/{pid}/transactions/{tid} [get]
func (th *transactionHandler) GetTransactionHandler(c *gin.Context) {
	err, transaction, _ := th.validate(c)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// @BasePath /api/v1

// GetTransactionsHandler godoc
// @Summary     Show transactions list
// @Description Get transactions by order_id and profile_id
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       order_id query    				   string false "Order id"
// @Param       profile_id query    			   int false "Profile id"
// @Success     200    {object}                    httputil.TransactionsResponse
// @Failure     400    {object}                    httputil.HTTPError
// @Failure     500    {object}                    httputil.HTTPError
// @Router      /transactions [get]
func (th *transactionHandler) GetTransactionsHandler(c *gin.Context) {
	var req LimitAndOffsetWithFiltersRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, overall, transactions := th.transactionStore.Query(c,
		&ecom.TransactionFilters{
			Paginate: ecom.SpecPaginate{
				Limit:  req.Limit,
				Offset: req.Offset,
			},
			OrderId:        req.OrderId,
			ProfileIds:     req.GetProfileIds(),
			TransactionIds: req.GetTransactionIds(),
			RRN:            req.RRN,
			AuthCode:       req.AuthCode,
			RemoteId:       req.RemoteId,
			CardMask:       req.CardMask,
		},
	)

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	for _, transaction := range transactions {
		var instrumentStore interface{}

		// @note: depends on instrument type
		switch *transaction.Instrument.Key {
		case "card":
			instrumentStore = bankcard.CardInstrumentStore{
				CardRepository:     th.cardStore,
				CardMaskRepository: th.cardMaskStore,
			}
		}

		err, instrumentApi := plugins.InstrumentApi(
			transaction.Instrument,
			instrumentStore,
			th.loggerFunc,
		)
		if err != nil {
			httputil.NewError(c, http.StatusInternalServerError, err)
			return
		}

		transaction.InstrumentInstance = instruments.NewInstrumentInstance(c, instrumentApi, transaction.InstrumentId)
	}

	httputil.TransactionsList(c, http.StatusOK, overall, transactions)
}

// @BasePath /api/v1

// RebillHandler godoc
// @Summary     Rebill transaction
// @Description Make recurring payment
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid     path     int true "Profile ID" Format(int64)
// @Param       tid     path     int true "Transaction ID" Format(int64)
// @Param       request body     validators.RebillRequest true "Rebill transaction"
// @Success     200     {object} ecom.Transaction
// @Failure     400     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /profiles/{pid}/transactions/{tid}/rebill [post]
func (th *transactionHandler) RebillHandler(c *gin.Context) {
	var req validators.RebillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, transaction, bankApi := th.validate(c)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	if !transaction.IsSuccess() {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Transaction has wrong state: %s", *transaction.Status))
		return
	}

	if !(transaction.IsAuth() || transaction.IsPreAuth()) {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Transaction has wrong type: %s", *transaction.Type))
		return
	}

	if !*transaction.Account.RebillEnabled {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Rebill not allowed"))
		return
	}

	th.loggerFunc(c).Printf("using account: %v", transaction.Account)

	err, newTransaction := ecom.NewTransaction(c, transaction.InstrumentInstance,
		repository.REBILL,
		th.rateStore,
		transaction.OrderId,
		transaction.Description,
		transaction.Profile,
		transaction.Account,
		transaction.Channel,
		transaction.Instrument,
		transaction.InstrumentId,
		&req.Amount,
		&req.Commission,
		transaction.Customer,
		transaction,
		nil,
	)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := th.transactionStore.Add(c, newTransaction); err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := bankApi.Rebill(c, newTransaction); err != nil {
		mess := err.Error()
		newTransaction.Declined(&mess)
	}

	if err, notfound := th.transactionStore.Update(c, newTransaction); err != nil {
		th.loggerFunc(c).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
	}

	c.JSON(http.StatusOK, newTransaction)
}

// @BasePath /api/v1

// ConfirmPreAuthHandler godoc
// @Summary     Confirm preauthorization
// @Description Complete preauthorization
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid     path     int true "Profile ID" Format(int64)
// @Param       tid     path     int true "Transaction ID" Format(int64)
// @Param       request body     validators.ConfirmPreAuthRequest true "Confirm preauthorization"
// @Success     200     {object} ecom.Transaction
// @Failure     400     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /profiles/{pid}/transactions/{tid}/confirm [post]
func (th *transactionHandler) ConfirmPreAuthHandler(c *gin.Context) {
	var req validators.ConfirmPreAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, transaction, bankApi := th.validate(c)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	if !transaction.IsSuccess() {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Transaction has wrong state: %s", *transaction.Status))
		return
	}

	if !transaction.IsPreAuth() {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Transaction has wrong type: %s", *transaction.Type))
		return
	}

	if req.TotalAmount() > transaction.TotalAmount() {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Incorrect amount: %d", req.TotalAmount()))
		return
	}

	if req.TotalAmount() < transaction.TotalAmount() && !*transaction.Account.PartialConfirmEnabled {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("Incorrect amount: %d, partial confirm not allowed", req.TotalAmount()))
		return
	}

	th.loggerFunc(c).Printf("using account: %v", transaction.Account)

	err, newTransaction := ecom.NewTransaction(c, transaction.InstrumentInstance,
		repository.CONFIRMAUTH,
		th.rateStore,
		transaction.OrderId,
		nil,
		transaction.Profile,
		transaction.Account,
		transaction.Channel,
		transaction.Instrument,
		transaction.InstrumentId,
		&req.Amount,
		&req.Commission,
		transaction.Customer,
		transaction,
		nil,
	)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := th.transactionStore.Add(c, newTransaction); err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := bankApi.Confirm(c, newTransaction); err != nil {
		mess := err.Error()
		newTransaction.Declined(&mess)
	}

	if err, notfound := th.transactionStore.Update(c, newTransaction); err != nil {
		th.loggerFunc(c).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
	}

	c.JSON(http.StatusOK, newTransaction)
}

// @BasePath /api/v1

// SbpAuthorizeHandler godoc
// @Summary     Authorize sbp
// @Description Make sbp authorization request
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid     path     int true "Profile ID" Format(int64)
// @Param       request body     validators.SbpAuthorizeRequest true "Authorize sbp"
// @Success     200     {object} ecom.Transaction
// @Failure     400     {object} httputil.HTTPError
// @Failure     404     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /profiles/{pid}/transactions/authorize/sbp [post]
func (th *transactionHandler) SbpAuthorizeHandler(c *gin.Context) {
	var req validators.SbpAuthorizeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	id, err := strconv.Atoi(c.Params.ByName("pid"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, _, profiles := th.profileStore.Query(c, repository.NewProfileSpecificationByID(id))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(profiles) == 0 {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Profile with id=%v not found", id))
		return
	}

	err, _, instances := th.instrumentStore.Query(c, repository.NewInstrumentSpecificationByKey("sbp"))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(instances) == 0 {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("SBP Instrument not found"))
		return
	}

	profile := profiles[0]
	instrument := instances[0]

	err, instrumentApi := plugins.InstrumentApi(
		instrument,
		nil,
		th.loggerFunc,
	)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	err, route := th.route(c, profile, instrument, nil, validators.SbpAuthorizationInstrumentRequester, req)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, bankApi := plugins.BankApi(th.cfg, route.Account, route.Channel, route.Instrument, nil, th.sessionStore, th.transactionStore, th.loggerFunc)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	th.loggerFunc(c).Printf("using account: %v", route.Account)

	err, transaction := ecom.NewTransaction(c, instruments.NewInstrumentInstance(c, instrumentApi, nil),
		repository.AUTH,
		th.rateStore,
		&req.OrderId,
		req.Description,
		profile,
		route.Account,
		route.Channel,
		instrument,
		nil,
		&req.Amount,
		&req.Commission,
		&req.Customer,
		nil,
		nil,
	)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := th.transactionStore.Add(c, transaction); err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := bankApi.Authorize(c, transaction, req); err != nil {
		mess := err.Error()
		transaction.Declined(&mess)
	}

	if err, notfound := th.transactionStore.Update(c, transaction); err != nil {
		th.loggerFunc(c).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
	}

	c.JSON(http.StatusOK, transaction)
}

// @BasePath /api/v1

// SbpPreAuthorizeHandler godoc
// @Summary     Preauthorize sbp
// @Description Make sbp preauthorization request
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid     path     int true "Profile ID" Format(int64)
// @Param       request body     validators.SbpPreAuthorizeRequest true "Preauthorize sbp"
// @Success     200     {object} ecom.Transaction
// @Failure     400     {object} httputil.HTTPError
// @Failure     404     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /profiles/{pid}/transactions/preauthorize/sbp [post]
func (th *transactionHandler) SbpPreAuthorizeHandler(c *gin.Context) {
	var req validators.SbpPreAuthorizeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	id, err := strconv.Atoi(c.Params.ByName("pid"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, _, profiles := th.profileStore.Query(c, repository.NewProfileSpecificationByID(id))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(profiles) == 0 {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Profile with id=%v not found", id))
		return
	}

	err, _, instances := th.instrumentStore.Query(c, repository.NewInstrumentSpecificationByKey("sbp"))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(instances) == 0 {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("SBP Instrument not found"))
		return
	}

	profile := profiles[0]
	instrument := instances[0]

	err, instrumentApi := plugins.InstrumentApi(
		instrument,
		nil,
		th.loggerFunc,
	)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	err, route := th.route(c, profile, instrument, nil, validators.SbpPreAuthorizationInstrumentRequester, req)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, bankApi := plugins.BankApi(th.cfg, route.Account, route.Channel, route.Instrument, nil, th.sessionStore, th.transactionStore, th.loggerFunc)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	th.loggerFunc(c).Printf("using account: %v", route.Account)

	err, transaction := ecom.NewTransaction(c, instruments.NewInstrumentInstance(c, instrumentApi, nil),
		repository.PREAUTH,
		th.rateStore,
		&req.OrderId,
		req.Description,
		profile,
		route.Account,
		route.Channel,
		instrument,
		nil,
		&req.Amount,
		&req.Commission,
		&req.Customer,
		nil,
		nil,
	)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := th.transactionStore.Add(c, transaction); err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := bankApi.PreAuthorize(c, transaction, req); err != nil {
		mess := err.Error()
		transaction.Declined(&mess)
	}

	if err, notfound := th.transactionStore.Update(c, transaction); err != nil {
		th.loggerFunc(c).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
	}

	c.JSON(http.StatusOK, transaction)
}

// @BasePath /api/v1

// CardAuthorizeHandler godoc
// @Summary     Authorize card
// @Description Make card authorization request
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid     path     int true "Profile ID" Format(int64)
// @Param       request body     validators.CardAuthorizeRequest true "Authorize card"
// @Success     200     {object} ecom.Transaction
// @Failure     400     {object} httputil.HTTPError
// @Failure     404     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /profiles/{pid}/transactions/authorize/card [post]
func (th *transactionHandler) CardAuthorizeHandler(c *gin.Context) {
	var req validators.CardAuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	id, err := strconv.Atoi(c.Params.ByName("pid"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, _, profiles := th.profileStore.Query(c, repository.NewProfileSpecificationByID(id))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(profiles) == 0 {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Profile with id=%v not found", id))
		return
	}

	err, _, instances := th.instrumentStore.Query(c, repository.NewInstrumentSpecificationByKey("card"))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(instances) == 0 {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Card Instrument not found"))
		return
	}

	profile := profiles[0]
	instrument := instances[0]

	cardInstrumentStore := bankcard.CardInstrumentStore{
		CardRepository:     th.cardStore,
		CardMaskRepository: th.cardMaskStore,
	}

	err, instrumentApi := plugins.InstrumentApi(
		instrument,
		cardInstrumentStore,
		th.loggerFunc,
	)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	instrumentInstance, err := instrumentApi.FromRequest(c, req, validators.CardAuthorizationInstrumentRequester)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	card, ok := instrumentInstance.(*repository.CardMask)
	if !ok {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("instrumentInstance has wrong type"))
		return
	}

	err, route := th.route(c, profile, instrument, cardInstrumentStore, validators.CardAuthorizationInstrumentRequester, req)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, bankApi := plugins.BankApi(th.cfg, route.Account, route.Channel, route.Instrument, cardInstrumentStore, th.sessionStore, th.transactionStore, th.loggerFunc)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	th.loggerFunc(c).Printf("using account: %v", route.Account)

	err, transaction := ecom.NewTransaction(c, instruments.NewInstrumentInstance(c, instrumentApi, card.Id),
		repository.AUTH,
		th.rateStore,
		&req.OrderId,
		req.Description,
		profile,
		route.Account,
		route.Channel,
		instrument,
		card.Id,
		&req.Amount,
		&req.Commission,
		&req.Customer,
		nil,
		&req.BrowserInfo,
	)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := th.transactionStore.Add(c, transaction); err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := bankApi.Authorize(c, transaction, req); err != nil {
		mess := err.Error()
		transaction.Declined(&mess)
	}

	if err, notfound := th.transactionStore.Update(c, transaction); err != nil {
		th.loggerFunc(c).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
	}

	c.JSON(http.StatusOK, transaction)
}

// @BasePath /api/v1

// CardPreAuthorizeHandler godoc
// @Summary     Preauthorize card
// @Description Make card preauthorization request
// @Tags        transactions
// @Accept      json
// @Produce     json
// @Param       pid     path     int true "Profile ID" Format(int64)
// @Param       request body     validators.CardPreAuthorizeRequest true "Preauthorize card"
// @Success     200     {object} ecom.Transaction
// @Failure     400     {object} httputil.HTTPError
// @Failure     404     {object} httputil.HTTPError
// @Failure     500     {object} httputil.HTTPError
// @Router      /profiles/{pid}/transactions/preauthorize/card [post]
func (th *transactionHandler) CardPreAuthorizeHandler(c *gin.Context) {
	var req validators.CardPreAuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	id, err := strconv.Atoi(c.Params.ByName("pid"))
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, _, profiles := th.profileStore.Query(c, repository.NewProfileSpecificationByID(id))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(profiles) == 0 {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Profile with id=%v not found", id))
		return
	}

	err, _, instances := th.instrumentStore.Query(c, repository.NewInstrumentSpecificationByKey("card"))

	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if len(instances) == 0 {
		httputil.NewError(c, http.StatusNotFound, fmt.Errorf("Card Instrument not found"))
		return
	}

	profile := profiles[0]
	instrument := instances[0]

	cardInstrumentStore := bankcard.CardInstrumentStore{
		CardRepository:     th.cardStore,
		CardMaskRepository: th.cardMaskStore,
	}

	err, instrumentApi := plugins.InstrumentApi(
		instrument,
		cardInstrumentStore,
		th.loggerFunc,
	)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	instrumentInstance, err := instrumentApi.FromRequest(c, req, validators.CardPreAuthorizationInstrumentRequester)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	card, ok := instrumentInstance.(*repository.CardMask)
	if !ok {
		httputil.NewError(c, http.StatusBadRequest, fmt.Errorf("instrumentInstance has wrong type"))
		return
	}

	err, route := th.route(c, profile, instrument, cardInstrumentStore, validators.CardPreAuthorizationInstrumentRequester, req)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	err, bankApi := plugins.BankApi(th.cfg, route.Account, route.Channel, route.Instrument, cardInstrumentStore, th.sessionStore, th.transactionStore, th.loggerFunc)
	if err != nil {
		httputil.NewError(c, http.StatusBadRequest, err)
		return
	}

	th.loggerFunc(c).Printf("using account: %v", route.Account)

	err, transaction := ecom.NewTransaction(c, instruments.NewInstrumentInstance(c, instrumentApi, card.Id),
		repository.PREAUTH,
		th.rateStore,
		&req.OrderId,
		req.Description,
		profile,
		route.Account,
		route.Channel,
		instrument,
		card.Id,
		&req.Amount,
		&req.Commission,
		&req.Customer,
		nil,
		&req.BrowserInfo,
	)
	if err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := th.transactionStore.Add(c, transaction); err != nil {
		httputil.NewError(c, http.StatusInternalServerError, err)
		return
	}

	if err := bankApi.PreAuthorize(c, transaction, req); err != nil {
		mess := err.Error()
		transaction.Declined(&mess)
	}

	if err, notfound := th.transactionStore.Update(c, transaction); err != nil {
		th.loggerFunc(c).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
	}

	c.JSON(http.StatusOK, transaction)
}

func NewTransactionHandler(
	routeStore ecom.RouteRepository,
	routerStore repository.RouterRepository,
	instrumentStore repository.InstrumentRepository,
	profileStore repository.ProfileRepository,
	accountStore ecom.AccountRepository,
	channelStore repository.ChannelRepository,
	currencyStore repository.CurrencyRepository,
	cardStore repository.CardRepository,
	cardMaskStore repository.CardMaskRepository,
	transactionStore ecom.TransactionRepository,
	sessionStore repository.SessionRepository,
	rateStore repository.RateRepository,
	cfg *config.Config,
	loggerFunc repository.LoggerFunc,
) *transactionHandler {
	return &transactionHandler{
		cfg:              cfg,
		loggerFunc:       loggerFunc,
		profileStore:     profileStore,
		accountStore:     accountStore,
		currencyStore:    currencyStore,
		channelStore:     channelStore,
		instrumentStore:  instrumentStore,
		routeStore:       routeStore,
		routerStore:      routerStore,
		cardStore:        cardStore,
		cardMaskStore:    cardMaskStore,
		transactionStore: transactionStore,
		sessionStore:     sessionStore,
		rateStore:        rateStore,
	}
}
