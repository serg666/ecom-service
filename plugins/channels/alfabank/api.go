package alfabank

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/config"
	"github.com/serg666/ecom-service/plugins"
	"github.com/serg666/ecom-service/plugins/channels"
	"github.com/serg666/ecom-service/plugins/channels/bpc"
	"github.com/serg666/ecom-service/plugins/instruments/card"
)

var (
	Id         = 2
	Key        = "alfabank"
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
			return fmt.Errorf("alfabank channel not sutable for instrument <%d>", *instrument.Id), nil
		}

		jsonbody, err := json.Marshal(account.Settings)
		if err != nil {
			return fmt.Errorf("can not marshal alfabank account settings: %v", err), nil
		}

		d := json.NewDecoder(bytes.NewReader(jsonbody))
		d.DisallowUnknownFields()

		var abs bpc.BpcSettings

		ecomCfg := &bpc.BpcEcomSettings{
			Url: cfg.Alfabank.Ecom.Url,
		}

		if err := d.Decode(&abs); err != nil {
			return fmt.Errorf("can not decode alfabank account settings: %v", err), nil
		}

		return nil, &AlfaBankChannel{bpc.BpcChannel{
			Cfg:              ecomCfg,
			Logger:           logger,
			InstrumentStore:  instrumentStore,
			SessionStore:     sessionStore,
			TransactionStore: transactionStore,
			Settings:         &abs,
		}}
	})
)

type AlfaBankChannel struct {
	bpc.BpcChannel
}
