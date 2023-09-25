package routers

import (
	creditcard "github.com/durango/go-credit-card"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	repository "github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/config"
	docs "github.com/serg666/ecom-service/docs"
	"github.com/serg666/ecom-service/handlers"
	"github.com/serg666/ecom-service/middlewares"
)

const basePath = "/api/v1"

func MakeHandler(
	routeStore ecom.RouteRepository,
	routerStore repository.RouterRepository,
	instrumentStore repository.InstrumentRepository,
	accountStore ecom.AccountRepository,
	channelStore repository.ChannelRepository,
	profileStore repository.ProfileRepository,
	currencyStore repository.CurrencyRepository,
	cardStore repository.CardRepository,
	transactionStore ecom.TransactionRepository,
	sessionStore repository.SessionRepository,
	rateStore repository.RateRepository,
	cardMaskStore repository.CardMaskRepository,
	cfg *config.Config,
	loggerFunc repository.LoggerFunc,
) *gin.Engine {
	routeHandler := handlers.NewRouteHandler(
		routeStore,
		profileStore,
		instrumentStore,
		accountStore,
		routerStore,
		currencyStore,
		channelStore,
		loggerFunc,
	)
	accountHandler := handlers.NewAccountHandler(accountStore, currencyStore, loggerFunc)
	profileHandler := handlers.NewProfileHandler(profileStore, currencyStore, loggerFunc)
	currencyHandler := handlers.NewCurrencyHandler(currencyStore, loggerFunc)
	channelHandler := handlers.NewChannelHandler(channelStore, loggerFunc)
	instrumentHandler := handlers.NewInstrumentHandler(instrumentStore, loggerFunc)
	transactionHandler := handlers.NewTransactionHandler(
		routeStore,
		routerStore,
		instrumentStore,
		profileStore,
		accountStore,
		channelStore,
		currencyStore,
		cardStore,
		cardMaskStore,
		transactionStore,
		sessionStore,
		rateStore,
		cfg,
		loggerFunc,
	)
	routerHandler := handlers.NewRouterHandler(routerStore, loggerFunc)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("iscvv", func(fl validator.FieldLevel) bool {
			card := creditcard.Card{Cvv: fl.Field().String()}
			err := card.ValidateCVV()
			return err == nil
		})
		v.RegisterValidation("luhncheck", func(fl validator.FieldLevel) bool {
			card := creditcard.Card{Number: fl.Field().String()}
			return card.ValidateNumber()
		})
		v.RegisterValidation("notempty", func(fl validator.FieldLevel) bool {
			return fl.Field().Len() != 0
		})
	}

	gin.EnableJsonDecoderDisallowUnknownFields()

	handler := gin.New()

	handler.Use(
		requestid.New(),
		middlewares.Logger(loggerFunc),
		gin.Recovery(),
	)

	docs.SwaggerInfo.BasePath = basePath
	docs.SwaggerInfo.Title = "Payment Gateway HTTP API"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Description = `Web service for merchants, that allow make online payments by card and through sbp`
	docs.SwaggerInfo.Host = cfg.Server.ExtHost + ":" + cfg.Server.ExtPort
	v1 := handler.Group(basePath)
	{
		v1.GET("/health", handlers.NewHealthCheckHandler().Health)

		// @note: payment interface
		v1.POST("/profiles/:pid/transactions/authorize/card", transactionHandler.CardAuthorizeHandler)
		v1.POST("/profiles/:pid/transactions/preauthorize/card", transactionHandler.CardPreAuthorizeHandler)
		v1.POST("/profiles/:pid/transactions/authorize/sbp", transactionHandler.SbpAuthorizeHandler)
		v1.POST("/profiles/:pid/transactions/preauthorize/sbp", transactionHandler.SbpPreAuthorizeHandler)
		v1.POST("/profiles/:pid/transactions/:tid/confirm", transactionHandler.ConfirmPreAuthHandler)
		v1.POST("/profiles/:pid/transactions/:tid/reverse", transactionHandler.ReverseHandler)
		v1.POST("/profiles/:pid/transactions/:tid/refund", transactionHandler.RefundHandler)
		v1.POST("/profiles/:pid/transactions/:tid/rebill", transactionHandler.RebillHandler)
		v1.POST("/profiles/:pid/transactions/:tid/update", transactionHandler.UpdateTransactionHandler)
		v1.POST("/profiles/:pid/transactions/:tid/completemethodurl", transactionHandler.CompleteMethodUrlHandler)
		v1.POST("/profiles/:pid/transactions/:tid/processcres", transactionHandler.ProcessCresHandler)
		v1.POST("/profiles/:pid/transactions/:tid/processpares", transactionHandler.ProcessParesHandler)
		v1.GET("/profiles/:pid/transactions/:tid", transactionHandler.GetTransactionHandler)
		v1.GET("/transactions", transactionHandler.GetTransactionsHandler)

		// @note: admin interface (should be moved to another web service)
		v1.POST("/routes", routeHandler.CreateRouteHandler)
		v1.GET("/routes", routeHandler.GetRoutesHandler)
		v1.GET("/routes/:id", routeHandler.GetRouteHandler)
		v1.DELETE("/routes/:id", routeHandler.DeleteRouteHandler)
		v1.PATCH("/routes/:id", routeHandler.PatchRouteHandler)

		v1.GET("/routers", routerHandler.GetRoutersHandler)

		v1.POST("/accounts", accountHandler.CreateAccountHandler)
		v1.GET("/accounts", accountHandler.GetAccountsHandler)
		v1.GET("/accounts/:id", accountHandler.GetAccountHandler)
		v1.DELETE("/accounts/:id", accountHandler.DeleteAccountHandler)
		v1.PATCH("/accounts/:id", accountHandler.PatchAccountHandler)

		v1.POST("/profiles", profileHandler.CreateProfileHandler)
		v1.GET("/profiles", profileHandler.GetProfilesHandler)
		v1.DELETE("/profiles/:pid", profileHandler.DeleteProfileHandler)
		v1.GET("/profiles/:pid", profileHandler.GetProfileHandler)
		v1.PATCH("/profiles/:pid", profileHandler.PatchProfileHandler)

		v1.POST("/currencies", currencyHandler.CreateCurrencyHandler)
		v1.GET("/currencies", currencyHandler.GetCurrenciesHandler)
		v1.DELETE("/currencies/:id", currencyHandler.DeleteCurrencyHandler)
		v1.GET("/currencies/:id", currencyHandler.GetCurrencyHandler)
		v1.PATCH("/currencies/:id", currencyHandler.PatchCurrencyHandler)

		v1.GET("/channels", channelHandler.GetChannelsHandler)

		v1.GET("/instruments", instrumentHandler.GetInstrumentsHandler)
	}
	handler.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	return handler
}
