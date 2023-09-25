package vbrrecom

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
	Id         = 3
	Key        = "vbrrecom"
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
			return fmt.Errorf("vbrr ecom channel not sutable for instrument <%d>", *instrument.Id), nil
		}

		jsonbody, err := json.Marshal(account.Settings)
		if err != nil {
			return fmt.Errorf("can not marshal vbrr ecom account settings: %v", err), nil
		}

		d := json.NewDecoder(bytes.NewReader(jsonbody))
		d.DisallowUnknownFields()

		var vbrrSettings bpc.BpcSettings

		ecomCfg := &bpc.BpcEcomSettings{
			Url: cfg.Vbrr.Ecom.Url,
		}

		if err := d.Decode(&vbrrSettings); err != nil {
			return fmt.Errorf("can not decode vbrr ecom account settings: %v", err), nil
		}

		return nil, &VbrrEcomChannel{bpc.BpcChannel{
			Cfg:              ecomCfg,
			Logger:           logger,
			InstrumentStore:  instrumentStore,
			SessionStore:     sessionStore,
			TransactionStore: transactionStore,
			Settings:         &vbrrSettings,
		}}
	})
)

type VbrrEcomChannel struct {
	bpc.BpcChannel
}
