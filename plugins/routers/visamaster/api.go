package visamaster

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/plugins"
	"github.com/serg666/ecom-service/plugins/instruments"
	"github.com/serg666/ecom-service/plugins/instruments/card"
	"github.com/serg666/ecom-service/plugins/routers"
)

var (
	Id         = 1
	Key        = "visamaster"
	Registered = plugins.RegisterRouter(Id, Key, func(
		route *ecom.Route,
		accountStore ecom.AccountRepository,
		channelStore repository.ChannelRepository,
		instrumentStore interface{},
		instrumentRequester instruments.InstrumentRequesterFunc,
		logger repository.LoggerFunc,
	) (error, routers.Router) {
		if *route.Instrument.Id != bankcard.Id {
			return fmt.Errorf("visamaster router not sutable for instrument <%d>", *route.Instrument.Id), nil
		}

		jsonbody, err := json.Marshal(route.Settings)
		if err != nil {
			return fmt.Errorf("can not marshal route settings: %v", err), nil
		}

		d := json.NewDecoder(bytes.NewReader(jsonbody))
		d.DisallowUnknownFields()

		var vms VisaMasterSettings

		if err := d.Decode(&vms); err != nil {
			return fmt.Errorf("can not decode route settings: %v", err), nil
		}

		return nil, &VisaMasterRouter{
			logger:              logger,
			accountStore:        accountStore,
			channelStore:        channelStore,
			instrumentStore:     instrumentStore,
			instrumentRequester: instrumentRequester,
			settings:            &vms,
		}
	})
)

type VisaMasterSettings struct {
	VisaAcc   int `json:"visa_acc"`
	MasterAcc int `json:"master_acc"`
}

type VisaMasterRouter struct {
	logger              repository.LoggerFunc
	accountStore        ecom.AccountRepository
	channelStore        repository.ChannelRepository
	instrumentStore     interface{}
	instrumentRequester instruments.InstrumentRequesterFunc
	settings            *VisaMasterSettings
}

func (vmr *VisaMasterRouter) Route(c *gin.Context, route *ecom.Route, request interface{}) error {
	err, instrumentApi := plugins.InstrumentApi(
		route.Instrument,
		vmr.instrumentStore,
		vmr.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to get instrument api: %v", err)
	}

	instrumentInstance, err := instrumentApi.FromRequest(c, request, vmr.instrumentRequester)
	if err != nil {
		return fmt.Errorf("failed to get instrumentInstance: %v", err)
	}

	cardInstrumentStore, ok := vmr.instrumentStore.(bankcard.CardInstrumentStore)
	if !ok {
		return fmt.Errorf("instrumentStore has wrong type")
	}

	cardMask, ok := instrumentInstance.(*repository.CardMask)
	if !ok {
		return fmt.Errorf("instrumentInstance has wrong type")
	}

	err, _, cards := cardInstrumentStore.CardRepository.Query(c, repository.NewCardSpecificationByToken(*cardMask.Token))
	if err != nil {
		return fmt.Errorf("failed to query card store: %v", err)
	}

	if len(cards) == 0 {
		return fmt.Errorf("card by token %s not found", *cardMask.Token)
	}
	card := cards[0]

	err, _, maccs := vmr.accountStore.Query(c, ecom.NewAccountSpecificationByID(vmr.settings.MasterAcc))
	if err != nil {
		return fmt.Errorf("failed to query acc store: %v", err)
	}

	err, _, vaccs := vmr.accountStore.Query(c, ecom.NewAccountSpecificationByID(vmr.settings.VisaAcc))
	if err != nil {
		return fmt.Errorf("failed to query acc store: %v", err)
	}

	if len(maccs) == 0 {
		return fmt.Errorf("mastercard account %d not found", vmr.settings.MasterAcc)
	}

	if len(vaccs) == 0 {
		return fmt.Errorf("visa account %d not found", vmr.settings.MasterAcc)
	}

	vmr.logger(c).Printf("visamaster routing card: %v", card)

	macc := maccs[0]
	vacc := vaccs[0]

	switch ctype := card.Type(); ctype {
	case "visa":
		route.Account = vacc
	case "mastercard":
		route.Account = macc
	}

	return nil
}
