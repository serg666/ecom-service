package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/config"
	"github.com/serg666/ecom-service/routers"
)

var (
	LoggerFunc = func(c interface{}) logrus.FieldLogger {
		logger := logrus.New()
		logger.SetLevel(logrus.DebugLevel)
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:          true,
			DisableLevelTruncation: true,
			ForceColors:            true,
		})
		var rid string
		if c != nil {
			rid = requestid.Get(c.(*gin.Context))
		}

		return logger.WithFields(logrus.Fields{
			"request_id": rid,
		})
	}
	Currency = &repository.Currency{
		Id:          intPtr(1),
		NumericCode: intPtr(643),
		Name:        strPtr("Российский рубль"),
		CharCode:    strPtr("RUB"),
		Exponent:    intPtr(100),
	}
	Profile = &repository.Profile{
		Id:          intPtr(1),
		Key:         strPtr("mock_profile"),
		Description: strPtr("Профиль для unit-теста"),
		Currency:    Currency,
	}
	Channel = &repository.Channel{
		Id:     intPtr(1),
		TypeId: intPtr(1),
		Key:    strPtr("testbank"),
	}
	Account = &ecom.Account{
		Id:       intPtr(1),
		Currency: Currency,
	}
	CardInstrument = &repository.Instrument{
		Id:  intPtr(1),
		Key: strPtr("card"),
	}
	CardMask = &repository.CardMask{
		Id:     intPtr(1),
		CardID: intPtr(1),
		Mask:   strPtr("411111**1111"),
		Token:  strPtr("ec62c7a4acbb035a881ea1ecbd7c18815f79f4c0c90b3f527be76eb22477cac4"),
	}
)

type mockRouteRepository struct {
	mock.Mock
}

func (m *mockRouteRepository) Add(ctx interface{}, route *ecom.Route) error {
	args := m.Called(ctx, route)
	return args.Error(0)
}

func (m *mockRouteRepository) Delete(ctx interface{}, route *ecom.Route) (error, bool) {
	args := m.Called(ctx, route)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockRouteRepository) Update(ctx interface{}, route *ecom.Route) (error, bool) {
	args := m.Called(ctx, route)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockRouteRepository) Query(ctx interface{}, specification ecom.RouteSpecification) (error, int, []*ecom.Route) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(int), args.Get(2).([]*ecom.Route)
}

type mockRouterRepository struct {
	mock.Mock
}

func (m *mockRouterRepository) Add(ctx interface{}, router *repository.Router) error {
	args := m.Called(ctx, router)
	return args.Error(0)
}

func (m *mockRouterRepository) Delete(ctx interface{}, router *repository.Router) (error, bool) {
	args := m.Called(ctx, router)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockRouterRepository) Update(ctx interface{}, router *repository.Router) (error, bool) {
	args := m.Called(ctx, router)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockRouterRepository) Query(ctx interface{}, specification repository.RouterSpecification) (error, int, []*repository.Router) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(int), args.Get(2).([]*repository.Router)
}

type mockInstrumentRepository struct {
	mock.Mock
}

func (m *mockInstrumentRepository) Add(ctx interface{}, instrument *repository.Instrument) error {
	args := m.Called(ctx, instrument)
	return args.Error(0)
}

func (m *mockInstrumentRepository) Delete(ctx interface{}, instrument *repository.Instrument) (error, bool) {
	args := m.Called(ctx, instrument)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockInstrumentRepository) Update(ctx interface{}, instrument *repository.Instrument) (error, bool) {
	args := m.Called(ctx, instrument)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockInstrumentRepository) Query(ctx interface{}, specification repository.InstrumentSpecification) (error, int, []*repository.Instrument) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(int), args.Get(2).([]*repository.Instrument)
}

type mockAccountRepository struct {
	mock.Mock
}

func (m *mockAccountRepository) Add(ctx interface{}, account *ecom.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *mockAccountRepository) Delete(ctx interface{}, account *ecom.Account) (error, bool) {
	args := m.Called(ctx, account)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockAccountRepository) Update(ctx interface{}, account *ecom.Account) (error, bool) {
	args := m.Called(ctx, account)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockAccountRepository) Query(ctx interface{}, specification ecom.AccountSpecification) (error, int, []*ecom.Account) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(int), args.Get(2).([]*ecom.Account)
}

type mockChannelRepository struct {
	mock.Mock
}

func (m *mockChannelRepository) Add(ctx interface{}, channel *repository.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *mockChannelRepository) Delete(ctx interface{}, channel *repository.Channel) (error, bool) {
	args := m.Called(ctx, channel)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockChannelRepository) Update(ctx interface{}, channel *repository.Channel) (error, bool) {
	args := m.Called(ctx, channel)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockChannelRepository) Query(ctx interface{}, specification repository.ChannelSpecification) (error, int, []*repository.Channel) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(int), args.Get(2).([]*repository.Channel)
}

type mockProfileRepository struct {
	mock.Mock
}

func (m *mockProfileRepository) Add(ctx interface{}, profile *repository.Profile) error {
	args := m.Called(ctx, profile)
	return args.Error(0)
}

func (m *mockProfileRepository) Delete(ctx interface{}, profile *repository.Profile) (error, bool) {
	args := m.Called(ctx, profile)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockProfileRepository) Update(ctx interface{}, profile *repository.Profile) (error, bool) {
	args := m.Called(ctx, profile)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockProfileRepository) Query(ctx interface{}, specification repository.ProfileSpecification) (error, int, []*repository.Profile) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(int), args.Get(2).([]*repository.Profile)
}

type mockCurrencyRepository struct {
	mock.Mock
}

func (m *mockCurrencyRepository) Add(ctx interface{}, currency *repository.Currency) error {
	args := m.Called(ctx, currency)
	return args.Error(0)
}

func (m *mockCurrencyRepository) Delete(ctx interface{}, currency *repository.Currency) (error, bool) {
	args := m.Called(ctx, currency)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockCurrencyRepository) Update(ctx interface{}, currency *repository.Currency) (error, bool) {
	args := m.Called(ctx, currency)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockCurrencyRepository) Query(ctx interface{}, specification repository.CurrencySpecification) (error, int, []*repository.Currency) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(int), args.Get(2).([]*repository.Currency)
}

type mockCardRepository struct {
	mock.Mock
}

func (m *mockCardRepository) Add(ctx interface{}, card *repository.Card) error {
	args := m.Called(ctx, card)
	return args.Error(0)
}

func (m *mockCardRepository) Delete(ctx interface{}, card *repository.Card) (error, bool) {
	args := m.Called(ctx, card)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockCardRepository) Query(ctx interface{}, specification repository.CardSpecification) (error, int, []*repository.Card) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(int), args.Get(2).([]*repository.Card)
}

type mockCardMaskRepository struct {
	mock.Mock
}

func (m *mockCardMaskRepository) Add(ctx interface{}, cardMask *repository.CardMask) error {
	args := m.Called(ctx, cardMask)
	return args.Error(0)
}

func (m *mockCardMaskRepository) Delete(ctx interface{}, cardMask *repository.CardMask) (error, bool) {
	args := m.Called(ctx, cardMask)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockCardMaskRepository) Update(ctx interface{}, cardMask *repository.CardMask) (error, bool) {
	args := m.Called(ctx, cardMask)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockCardMaskRepository) Query(ctx interface{}, specification repository.CardMaskSpecification) (error, int, []*repository.CardMask) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(int), args.Get(2).([]*repository.CardMask)
}

type mockTransactionRepository struct {
	mock.Mock
}

func (m *mockTransactionRepository) Add(ctx interface{}, transaction *ecom.Transaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *mockTransactionRepository) Update(ctx interface{}, transaction *ecom.Transaction) (error, bool) {
	args := m.Called(ctx, transaction)
	return args.Error(0), args.Get(1).(bool)
}

func (m *mockTransactionRepository) Query(ctx interface{}, specification ecom.TransactionSpecification) (error, int, []*ecom.Transaction) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(int), args.Get(2).([]*ecom.Transaction)
}

func (m *mockTransactionRepository) TypeTurnOver(ctx interface{}, specification ecom.TransactionSpecification) (error, *map[string]ecom.TurnOverResult) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(*map[string]ecom.TurnOverResult)
}

type mockSessionRepository struct {
	mock.Mock
}

func (m *mockSessionRepository) Add(ctx interface{}, session *repository.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockSessionRepository) Query(ctx interface{}, specification repository.SessionSpecification) (error, int, []*repository.Session) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(int), args.Get(2).([]*repository.Session)
}

type mockRateRepository struct {
	mock.Mock
}

func (m *mockRateRepository) Add(ctx interface{}, rate *repository.Rate) error {
	args := m.Called(ctx, rate)
	return args.Error(0)
}

func (m *mockRateRepository) GetRate(ctx interface{}, date *time.Time, src *repository.Currency, dst *repository.Currency) (error, bool, *float64) {
	args := m.Called(ctx, date, src, dst)
	return args.Error(0), args.Get(1).(bool), args.Get(2).(*float64)
}

func (m *mockRateRepository) Query(ctx interface{}, specification repository.RateSpecification) (error, int, []*repository.Rate) {
	args := m.Called(ctx, specification)
	return args.Error(0), args.Get(1).(int), args.Get(2).([]*repository.Rate)
}

func NewMockRouteRepository() *mockRouteRepository {
	return &mockRouteRepository{}
}

func NewMockRouterRepository() *mockRouterRepository {
	return &mockRouterRepository{}
}

func NewMockInstrumentRepository() *mockInstrumentRepository {
	return &mockInstrumentRepository{}
}

func NewMockAccountRepository() *mockAccountRepository {
	return &mockAccountRepository{}
}

func NewMockChannelRepository() *mockChannelRepository {
	return &mockChannelRepository{}
}

func NewMockProfileRepository() *mockProfileRepository {
	return &mockProfileRepository{}
}

func NewMockCurrencyRepository() *mockCurrencyRepository {
	return &mockCurrencyRepository{}
}

func NewMockCardRepository() *mockCardRepository {
	return &mockCardRepository{}
}

func NewMockCardMaskRepository() *mockCardMaskRepository {
	return &mockCardMaskRepository{}
}

func NewMockTransactionRepository() *mockTransactionRepository {
	return &mockTransactionRepository{}
}

func NewMockSessionRepository() *mockSessionRepository {
	return &mockSessionRepository{}
}

func NewMockRateRepository() *mockRateRepository {
	return &mockRateRepository{}
}

func boolPtr(val bool) *bool {
	return &val
}

func strPtr(val string) *string {
	return &val
}

func intPtr(val int) *int {
	return &val
}

func TestGetTransactionOk(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/profiles/1/transactions/1", nil)

	routeRepo := NewMockRouteRepository()
	routerRepo := NewMockRouterRepository()
	instrumentRepo := NewMockInstrumentRepository()
	accountRepo := NewMockAccountRepository()
	channelRepo := NewMockChannelRepository()
	profileRepo := NewMockProfileRepository()
	currencyRepo := NewMockCurrencyRepository()
	cardRepo := NewMockCardRepository()
	cardMaskRepo := NewMockCardMaskRepository()
	transactionRepo := NewMockTransactionRepository()
	sessionRepo := NewMockSessionRepository()
	rateRepo := NewMockRateRepository()

	mList := make([]*repository.CardMask, 0)
	mList = append(mList, CardMask)
	pList := make([]*repository.Profile, 0)
	pList = append(pList, Profile)
	tList := make([]*ecom.Transaction, 0)
	tList = append(tList, &ecom.Transaction{
		Id:                intPtr(1),
		Profile:           Profile,
		Account:           Account,
		Channel:           Channel,
		Currency:          Currency,
		CurrencyConverted: Currency,
		Instrument:        CardInstrument,
		InstrumentId:      intPtr(1),
	})

	profileRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, pList)
	transactionRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, tList)
	cardMaskRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, mList)

	router := routers.MakeHandler(
		routeRepo,
		routerRepo,
		instrumentRepo,
		accountRepo,
		channelRepo,
		profileRepo,
		currencyRepo,
		cardRepo,
		transactionRepo,
		sessionRepo,
		rateRepo,
		cardMaskRepo,
		&config.Config{},
		LoggerFunc,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetTransactionFail(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/profiles/1/transactions/1", nil)

	routeRepo := NewMockRouteRepository()
	routerRepo := NewMockRouterRepository()
	instrumentRepo := NewMockInstrumentRepository()
	accountRepo := NewMockAccountRepository()
	channelRepo := NewMockChannelRepository()
	profileRepo := NewMockProfileRepository()
	currencyRepo := NewMockCurrencyRepository()
	cardRepo := NewMockCardRepository()
	cardMaskRepo := NewMockCardMaskRepository()
	transactionRepo := NewMockTransactionRepository()
	sessionRepo := NewMockSessionRepository()
	rateRepo := NewMockRateRepository()

	mList := make([]*repository.CardMask, 0)
	mList = append(mList, CardMask)
	pList := make([]*repository.Profile, 0)
	pList = append(pList, Profile)
	tList := make([]*ecom.Transaction, 0)

	tList = append(tList, &ecom.Transaction{
		Profile: &repository.Profile{
			Id:          intPtr(2),
			Key:         strPtr("mock_profile"),
			Description: strPtr("Профиль для unit-теста"),
			Currency:    Currency,
		},
		Id:                intPtr(1),
		Account:           Account,
		Channel:           Channel,
		Currency:          Currency,
		CurrencyConverted: Currency,
		Instrument:        CardInstrument,
		InstrumentId:      intPtr(1),
	})

	profileRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, pList)
	transactionRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, tList)
	cardMaskRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, mList)

	router := routers.MakeHandler(
		routeRepo,
		routerRepo,
		instrumentRepo,
		accountRepo,
		channelRepo,
		profileRepo,
		currencyRepo,
		cardRepo,
		transactionRepo,
		sessionRepo,
		rateRepo,
		cardMaskRepo,
		&config.Config{},
		LoggerFunc,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetTransactionsListFailWithWrongParams(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/transactions?order_id=1&profile_id=wrongvalue", nil)

	routeRepo := NewMockRouteRepository()
	routerRepo := NewMockRouterRepository()
	instrumentRepo := NewMockInstrumentRepository()
	accountRepo := NewMockAccountRepository()
	channelRepo := NewMockChannelRepository()
	profileRepo := NewMockProfileRepository()
	currencyRepo := NewMockCurrencyRepository()
	cardRepo := NewMockCardRepository()
	cardMaskRepo := NewMockCardMaskRepository()
	transactionRepo := NewMockTransactionRepository()
	sessionRepo := NewMockSessionRepository()
	rateRepo := NewMockRateRepository()

	router := routers.MakeHandler(
		routeRepo,
		routerRepo,
		instrumentRepo,
		accountRepo,
		channelRepo,
		profileRepo,
		currencyRepo,
		cardRepo,
		transactionRepo,
		sessionRepo,
		rateRepo,
		cardMaskRepo,
		&config.Config{},
		LoggerFunc,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetTransactionsListQueringFail(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/transactions?order_id=1&profile_id=1", nil)

	routeRepo := NewMockRouteRepository()
	routerRepo := NewMockRouterRepository()
	instrumentRepo := NewMockInstrumentRepository()
	accountRepo := NewMockAccountRepository()
	channelRepo := NewMockChannelRepository()
	profileRepo := NewMockProfileRepository()
	currencyRepo := NewMockCurrencyRepository()
	cardRepo := NewMockCardRepository()
	cardMaskRepo := NewMockCardMaskRepository()
	transactionRepo := NewMockTransactionRepository()
	sessionRepo := NewMockSessionRepository()
	rateRepo := NewMockRateRepository()

	var l []*ecom.Transaction
	var c int

	transactionRepo.On("Query", mock.Anything, mock.Anything).Return(errors.New("failed to query transaction repo"), c, l)

	router := routers.MakeHandler(
		routeRepo,
		routerRepo,
		instrumentRepo,
		accountRepo,
		channelRepo,
		profileRepo,
		currencyRepo,
		cardRepo,
		transactionRepo,
		sessionRepo,
		rateRepo,
		cardMaskRepo,
		&config.Config{},
		LoggerFunc,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetTransactionsListWrongInstrumentType(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/transactions?order_id=1&profile_id=1", nil)

	routeRepo := NewMockRouteRepository()
	routerRepo := NewMockRouterRepository()
	instrumentRepo := NewMockInstrumentRepository()
	accountRepo := NewMockAccountRepository()
	channelRepo := NewMockChannelRepository()
	profileRepo := NewMockProfileRepository()
	currencyRepo := NewMockCurrencyRepository()
	cardRepo := NewMockCardRepository()
	cardMaskRepo := NewMockCardMaskRepository()
	transactionRepo := NewMockTransactionRepository()
	sessionRepo := NewMockSessionRepository()
	rateRepo := NewMockRateRepository()

	tList := make([]*ecom.Transaction, 0)

	tList = append(tList, &ecom.Transaction{
		Instrument: &repository.Instrument{
			Id:  intPtr(-1),
			Key: strPtr("unknown"),
		},
		Profile:           Profile,
		Id:                intPtr(1),
		Account:           Account,
		Channel:           Channel,
		Currency:          Currency,
		CurrencyConverted: Currency,
		InstrumentId:      intPtr(1),
	})

	transactionRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, tList)

	router := routers.MakeHandler(
		routeRepo,
		routerRepo,
		instrumentRepo,
		accountRepo,
		channelRepo,
		profileRepo,
		currencyRepo,
		cardRepo,
		transactionRepo,
		sessionRepo,
		rateRepo,
		cardMaskRepo,
		&config.Config{},
		LoggerFunc,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetTransactionsListOk(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/transactions?order_id=1&profile_id=1", nil)

	routeRepo := NewMockRouteRepository()
	routerRepo := NewMockRouterRepository()
	instrumentRepo := NewMockInstrumentRepository()
	accountRepo := NewMockAccountRepository()
	channelRepo := NewMockChannelRepository()
	profileRepo := NewMockProfileRepository()
	currencyRepo := NewMockCurrencyRepository()
	cardRepo := NewMockCardRepository()
	cardMaskRepo := NewMockCardMaskRepository()
	transactionRepo := NewMockTransactionRepository()
	sessionRepo := NewMockSessionRepository()
	rateRepo := NewMockRateRepository()

	mList := make([]*repository.CardMask, 0)
	mList = append(mList, CardMask)
	pList := make([]*repository.Profile, 0)
	pList = append(pList, Profile)
	tList := make([]*ecom.Transaction, 0)

	tList = append(tList, &ecom.Transaction{
		Profile:           Profile,
		Id:                intPtr(1),
		Account:           Account,
		Channel:           Channel,
		Currency:          Currency,
		CurrencyConverted: Currency,
		Instrument:        CardInstrument,
		InstrumentId:      intPtr(1),
	})

	profileRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, pList)
	transactionRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, tList)
	cardMaskRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, mList)

	router := routers.MakeHandler(
		routeRepo,
		routerRepo,
		instrumentRepo,
		accountRepo,
		channelRepo,
		profileRepo,
		currencyRepo,
		cardRepo,
		transactionRepo,
		sessionRepo,
		rateRepo,
		cardMaskRepo,
		&config.Config{},
		LoggerFunc,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProcessParesWithUnknownParams(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/profiles/1/transactions/1/processpares", strings.NewReader(
		`{
		  "pares":   "somevalue",
		  "unknown": "somevalue"
		 }`,
	))

	routeRepo := NewMockRouteRepository()
	routerRepo := NewMockRouterRepository()
	instrumentRepo := NewMockInstrumentRepository()
	accountRepo := NewMockAccountRepository()
	channelRepo := NewMockChannelRepository()
	profileRepo := NewMockProfileRepository()
	currencyRepo := NewMockCurrencyRepository()
	cardRepo := NewMockCardRepository()
	cardMaskRepo := NewMockCardMaskRepository()
	transactionRepo := NewMockTransactionRepository()
	sessionRepo := NewMockSessionRepository()
	rateRepo := NewMockRateRepository()

	router := routers.MakeHandler(
		routeRepo,
		routerRepo,
		instrumentRepo,
		accountRepo,
		channelRepo,
		profileRepo,
		currencyRepo,
		cardRepo,
		transactionRepo,
		sessionRepo,
		rateRepo,
		cardMaskRepo,
		&config.Config{},
		LoggerFunc,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProcessParesIncorrectTransaction(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/profiles/1/transactions/1/processpares", strings.NewReader(`{"pares": "somevalue"}`))

	routeRepo := NewMockRouteRepository()
	routerRepo := NewMockRouterRepository()
	instrumentRepo := NewMockInstrumentRepository()
	accountRepo := NewMockAccountRepository()
	channelRepo := NewMockChannelRepository()
	profileRepo := NewMockProfileRepository()
	currencyRepo := NewMockCurrencyRepository()
	cardRepo := NewMockCardRepository()
	cardMaskRepo := NewMockCardMaskRepository()
	transactionRepo := NewMockTransactionRepository()
	sessionRepo := NewMockSessionRepository()
	rateRepo := NewMockRateRepository()

	mList := make([]*repository.CardMask, 0)
	mList = append(mList, CardMask)
	pList := make([]*repository.Profile, 0)
	pList = append(pList, Profile)
	tList := make([]*ecom.Transaction, 0)

	tList = append(tList, &ecom.Transaction{
		Profile: &repository.Profile{
			Id:          intPtr(2),
			Key:         strPtr("mock_profile"),
			Description: strPtr("Профиль для unit-теста"),
			Currency:    Currency,
		},
		Id:                intPtr(1),
		Account:           Account,
		Channel:           Channel,
		Currency:          Currency,
		CurrencyConverted: Currency,
		Instrument:        CardInstrument,
		InstrumentId:      intPtr(1),
	})

	profileRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, pList)
	transactionRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, tList)
	cardMaskRepo.On("Query", mock.Anything, mock.Anything).Return(nil, 1, mList)

	router := routers.MakeHandler(
		routeRepo,
		routerRepo,
		instrumentRepo,
		accountRepo,
		channelRepo,
		profileRepo,
		currencyRepo,
		cardRepo,
		transactionRepo,
		sessionRepo,
		rateRepo,
		cardMaskRepo,
		&config.Config{},
		LoggerFunc,
	)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
