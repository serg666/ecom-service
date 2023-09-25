package main

import (
	"log"

	"github.com/sirupsen/logrus"

	//"github.com/wk8/go-ordered-map"
	repository "github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/client"
	"github.com/serg666/ecom-service/config"
	"github.com/serg666/ecom-service/gateway"
	"github.com/serg666/ecom-service/routers"
)

func main() {
	cfgPath, err := config.ParseFlags()
	if err != nil {
		log.Fatalf("can not parse flags due to: %v", err)
	}

	cfg, err := config.NewConfig(cfgPath)
	if err != nil {
		log.Fatalf("can not get new config due to: %v", err)
	}

	pgPool, err := repository.MakePgPoolFromDSN(cfg.Databases.Default.Dsn)
	if err != nil {
		log.Fatalf("Can not make pg pool: %v", err)
	}

	loggerFunc := func(c interface{}) logrus.FieldLogger {
		return cfg.LogRusLogger(c)
	}

	client.Client = cfg.HttpClient()

	//currencyStore := repository.NewOrderedMapCurrencyStore(orderedmap.New(), loggerFunc)
	currencyStore := repository.NewPGPoolCurrencyStore(pgPool, loggerFunc)
	//profileStore := repository.NewOrderedMapProfileStore(orderedmap.New(), currencyStore, loggerFunc)
	profileStore := repository.NewPGPoolProfileStore(pgPool, currencyStore, loggerFunc)
	//sessionStore := repository.NewOrderedMapSessionStore(orderedmap.New(), loggerFunc)
	sessionStore := repository.NewHttpClientSessionStore(cfg.CardStore.Url, client.Client, loggerFunc)
	//cardStore := repository.NewOrderedMapCardStore(orderedmap.New(), loggerFunc)
	cardStore := repository.NewHttpClientCardStore(cfg.CardStore.Url, client.Client, loggerFunc)
	cardMaskStore := repository.NewPGPoolCardMaskStore(pgPool, loggerFunc)
	channelStore := repository.NewPGPoolChannelStore(pgPool, loggerFunc)
	accountStore := ecom.NewPGPoolAccountStore(pgPool, currencyStore, loggerFunc)
	rateStore := repository.NewPGPoolRateStore(pgPool, currencyStore, loggerFunc)
	instrumentStore := repository.NewPGPoolInstrumentStore(pgPool, loggerFunc)
	routerStore := repository.NewPGPoolRouterStore(pgPool, loggerFunc)
	routeStore := ecom.NewPGPoolRouteStore(
		pgPool,
		profileStore,
		instrumentStore,
		accountStore,
		channelStore,
		routerStore,
		loggerFunc,
	)
	transactionStore := ecom.NewPGPoolTransactionStore(
		pgPool,
		profileStore,
		instrumentStore,
		accountStore,
		channelStore,
		currencyStore,
		loggerFunc,
	)

	gateway.RegisterPlugins(channelStore, instrumentStore, routerStore)

	handler := routers.MakeHandler(
		routeStore,
		routerStore,
		instrumentStore,
		accountStore,
		channelStore,
		profileStore,
		currencyStore,
		cardStore,
		transactionStore,
		sessionStore,
		rateStore,
		cardMaskStore,
		cfg,
		loggerFunc,
	)

	// Run the server
	cfg.RunServer(handler, loggerFunc)
}
