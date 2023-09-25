package main

import (
	"flag"
	"log"
	"os"
	"syscall"

	"github.com/sirupsen/logrus"

	repository "github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/client"
	"github.com/serg666/ecom-service/config"
	"github.com/serg666/ecom-service/gateway"
	"github.com/serg666/ecom-service/plugins"
	"github.com/serg666/ecom-service/plugins/instruments"
	bankcard "github.com/serg666/ecom-service/plugins/instruments/card"
)

func create_lock_file(filename string) (*os.File, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func main() {
	cfgPath := flag.String("config", "./config.yml", "path to config file")
	batchSize := flag.Int("batch_size", 1000, "Size for the maximum chunk of data in memory")
	pidFile := flag.String("pid", "./updatetransactions.pid", "full path to pid file")

	flag.Parse()

	if err := config.ValidateConfigPath(cfgPath); err != nil {
		log.Fatalf("can not parse flags due to: %v", err)
	}

	cfg, err := config.NewConfig(cfgPath)
	if err != nil {
		log.Fatalf("can not get new config due to: %v", err)
	}

	pgPool, err := repository.MakePgPoolFromDSN(cfg.Databases.Default.Dsn)
	if err != nil {
		log.Fatalf("can not make pg pool: %v", err)
	}

	f, err := create_lock_file(*pidFile)
	if err != nil {
		log.Fatalf("can not create lock file: %v", err)
	}

	defer f.Close()

	loggerFunc := func(c interface{}) logrus.FieldLogger {
		return cfg.LogRusLogger(c)
	}

	currencyStore := repository.NewPGPoolCurrencyStore(pgPool, loggerFunc)
	profileStore := repository.NewPGPoolProfileStore(pgPool, currencyStore, loggerFunc)
	instrumentStore := repository.NewPGPoolInstrumentStore(pgPool, loggerFunc)
	accountStore := ecom.NewPGPoolAccountStore(pgPool, currencyStore, loggerFunc)
	channelStore := repository.NewPGPoolChannelStore(pgPool, loggerFunc)
	cardStore := repository.NewHttpClientCardStore(cfg.CardStore.Url, client.Client, loggerFunc)
	cardMaskStore := repository.NewPGPoolCardMaskStore(pgPool, loggerFunc)
	sessionStore := repository.NewHttpClientSessionStore(cfg.CardStore.Url, client.Client, loggerFunc)
	routerStore := repository.NewPGPoolRouterStore(pgPool, loggerFunc)
	transactionStore := ecom.NewPGPoolTransactionStore(
		pgPool,
		profileStore,
		instrumentStore,
		accountStore,
		channelStore,
		currencyStore,
		loggerFunc,
	)

	client.Client = cfg.HttpClient()

	gateway.RegisterPlugins(channelStore, instrumentStore, routerStore)

	loggerFunc(nil).Printf("start update transactions")
	lastid := 0
	for {
		err, _, transactions := transactionStore.Query(nil,
			ecom.NewTransactionSpecificationByNotFinalStatus(*batchSize, lastid),
		)

		if err != nil {
			log.Fatalf("faild to query transaction store: %v", err)
		}

		for _, transaction := range transactions {
			loggerFunc(nil).Printf("process transaction: %d", *transaction.Id)

			var iStore interface{}

			// @note: depends on instrument type
			switch *transaction.Instrument.Key {
			case "card":
				iStore = bankcard.CardInstrumentStore{
					CardRepository:     cardStore,
					CardMaskRepository: cardMaskStore,
				}
			}

			err, instrumentApi := plugins.InstrumentApi(
				transaction.Instrument,
				iStore,
				loggerFunc,
			)

			if err != nil {
				log.Fatalf("failed to get instrument api: %v", err)
			}

			transaction.InstrumentInstance = instruments.NewInstrumentInstance(nil, instrumentApi, transaction.InstrumentId)

			err, bankApi := plugins.BankApi(
				cfg,
				transaction.Account,
				transaction.Channel,
				transaction.Instrument,
				iStore,
				sessionStore,
				transactionStore,
				loggerFunc,
			)
			if err != nil {
				log.Fatalf("faild to get bank api: %v", err)
			}

			loggerFunc(nil).Printf("using account: %v", transaction.Account)
			bankApi.Update(nil, transaction, true)

			if err, notfound := transactionStore.Update(nil, transaction); err != nil {
				loggerFunc(nil).Warningf("failed to update transaction: %v (notfound: %v)", err, notfound)
			}

			lastid = *transaction.Id
		}

		if len(transactions) == 0 {
			break
		}
	}

	loggerFunc(nil).Printf("stop update transactions")
}
