package testbank

import (
	"fmt"

	"github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/config"
	"github.com/serg666/ecom-service/plugins"
	"github.com/serg666/ecom-service/plugins/channels"
	"github.com/serg666/ecom-service/plugins/instruments/card"
)

var (
	Id         = 1
	Key        = "testbank"
	Registered = plugins.RegisterBankChannel(Id, Key, func(
		cfg *config.Config,
		account *ecom.Account,
		instrument *repository.Instrument,
		instrumentStore interface{},
		sessionStore repository.SessionRepository,
		transactionStore ecom.TransactionRepository,
		logger repository.LoggerFunc,
	) (error, channels.BankChannel) {
		if *instrument.Id != bankcard.Id {
			return fmt.Errorf("testbank channel not sutable for instrument <%d>", *instrument.Id), nil
		}

		return nil, &TestBankChannel{
			cfg:              cfg,
			sessionStore:     sessionStore,
			instrumentStore:  instrumentStore,
			transactionStore: transactionStore,
			logger:           logger,
		}
	})
)

type TestBankChannel struct {
	cfg              *config.Config
	sessionStore     repository.SessionRepository
	transactionStore ecom.TransactionRepository
	instrumentStore  interface{}
	logger           repository.LoggerFunc
}

func (kbc *TestBankChannel) Authorize(c interface{}, transaction *ecom.Transaction, request interface{}) error {
	kbc.logger(c).Print("authorize int")
	transaction.Success()
	return nil
}

func (kbc *TestBankChannel) PreAuthorize(c interface{}, transaction *ecom.Transaction, request interface{}) error {
	kbc.logger(c).Print("preauthorize int")
	transaction.Success()
	return nil
}

func (kbc *TestBankChannel) Confirm(c interface{}, transaction *ecom.Transaction) error {
	kbc.logger(c).Print("confirm int")
	transaction.Success()
	return nil
}

func (kbc *TestBankChannel) Reverse(c interface{}, transaction *ecom.Transaction) error {
	kbc.logger(c).Print("reverse int")
	transaction.Success()
	return nil
}

func (kbc *TestBankChannel) Refund(c interface{}, transaction *ecom.Transaction) error {
	kbc.logger(c).Print("refund int")
	transaction.Success()
	return nil
}

func (kbc *TestBankChannel) Rebill(c interface{}, transaction *ecom.Transaction) error {
	kbc.logger(c).Print("rebill int")
	transaction.Success()
	return nil
}

func (kbc *TestBankChannel) ProcessCres(c interface{}, transaction *ecom.Transaction, cres string) error {
	kbc.logger(c).Print("process cres")
	transaction.Success()
	return nil
}

func (kbc *TestBankChannel) ProcessPares(c interface{}, transaction *ecom.Transaction, pares string) error {
	kbc.logger(c).Print("process pares")
	transaction.Success()
	return nil
}

func (kbc *TestBankChannel) CompleteMethodUrl(c interface{}, transaction *ecom.Transaction, completed bool) error {
	kbc.logger(c).Print("complete method url")
	transaction.Success()
	return nil
}

func (kbc *TestBankChannel) Update(c interface{}, transaction *ecom.Transaction, force bool) {
	kbc.logger(c).Print("update int")
	transaction.Success()
	return
}
