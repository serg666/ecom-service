package httputil

import (
	"github.com/gin-gonic/gin"

	"github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"
)

type AccountsResponse struct {
	Overall  int            `json:"overall"`
	Accounts []*ecom.Account `json:"accounts"`
}

type ProfilesResponse struct {
	Overall  int                   `json:"overall"`
	Profiles []*repository.Profile `json:"profiles"`
}

type CurrenciesResponse struct {
	Overall    int                    `json:"overall"`
	Currencies []*repository.Currency `json:"currencies"`
}

type ChannelsResponse struct {
	Overall  int                   `json:"overall"`
	Channels []*repository.Channel `json:"channels"`
}

type InstrumentsResponse struct {
	Overall     int                      `json:"overall"`
	Instruments []*repository.Instrument `json:"instruments"`
}

type RoutesResponse struct {
	Overall int          `json:"overall"`
	Routes  []*ecom.Route `json:"routes"`
}

type RoutersResponse struct {
	Overall int                  `json:"overall"`
	Routers []*repository.Router `json:"routers"`
}

type TransactionsResponse struct {
	Overall      int                `json:"overall"`
	Transactions []*ecom.Transaction `json:"transactions"`
}

func AccountsList(ctx *gin.Context, status int, overall int, accounts []*ecom.Account) {
	resp := AccountsResponse{
		Overall:  overall,
		Accounts: accounts,
	}
	ctx.JSON(status, resp)
}

func ProfilesList(ctx *gin.Context, status int, overall int, profiles []*repository.Profile) {
	resp := ProfilesResponse{
		Overall:  overall,
		Profiles: profiles,
	}
	ctx.JSON(status, resp)
}

func CurrenciesList(ctx *gin.Context, status int, overall int, currencies []*repository.Currency) {
	resp := CurrenciesResponse{
		Overall:    overall,
		Currencies: currencies,
	}
	ctx.JSON(status, resp)
}

func ChannelsList(ctx *gin.Context, status int, overall int, channels []*repository.Channel) {
	resp := ChannelsResponse{
		Overall:  overall,
		Channels: channels,
	}
	ctx.JSON(status, resp)
}

func InstrumentsList(ctx *gin.Context, status int, overall int, instruments []*repository.Instrument) {
	resp := InstrumentsResponse{
		Overall:     overall,
		Instruments: instruments,
	}
	ctx.JSON(status, resp)
}

func RoutesList(ctx *gin.Context, status int, overall int, routes []*ecom.Route) {
	resp := RoutesResponse{
		Overall: overall,
		Routes:  routes,
	}
	ctx.JSON(status, resp)
}

func RoutersList(ctx *gin.Context, status int, overall int, routers []*repository.Router) {
	resp := RoutersResponse{
		Overall: overall,
		Routers: routers,
	}
	ctx.JSON(status, resp)
}

func TransactionsList(ctx *gin.Context, status int, overall int, transactions []*ecom.Transaction) {
	resp := TransactionsResponse{
		Overall:      overall,
		Transactions: transactions,
	}
	ctx.JSON(status, resp)
}
