package gateway

import (
	"log"

	repository "github.com/serg666/service-repository"

	"github.com/serg666/ecom-service/plugins"
	"github.com/serg666/ecom-service/plugins/channels/alfaaft"
	"github.com/serg666/ecom-service/plugins/channels/alfabank"
	"github.com/serg666/ecom-service/plugins/channels/gpb"
	"github.com/serg666/ecom-service/plugins/channels/gpbsbp"
	"github.com/serg666/ecom-service/plugins/channels/testbank"
	"github.com/serg666/ecom-service/plugins/channels/testsbp"
	vbrrecom "github.com/serg666/ecom-service/plugins/channels/vbrr_ecom"
	bankcard "github.com/serg666/ecom-service/plugins/instruments/card"
	sbp "github.com/serg666/ecom-service/plugins/instruments/sbp"
	"github.com/serg666/ecom-service/plugins/routers/visamaster"
)

func RegisterPlugins(
	channelStore repository.ChannelRepository,
	instrumentStore repository.InstrumentRepository,
	routerStore repository.RouterRepository,
) {
	if visamaster.Registered != nil {
		log.Fatalf("Can not register visamaster router: %v", visamaster.Registered)
	}

	if bankcard.Registered != nil {
		log.Fatalf("Can not register bank card instrument type: %v", bankcard.Registered)
	}

	if sbp.Registered != nil {
		log.Fatalf("Can not register sbp instrument type: %v", sbp.Registered)
	}

	if testbank.Registered != nil {
		log.Fatalf("Can not register testbank channel: %v", testbank.Registered)
	}

	if testsbp.Registered != nil {
		log.Fatalf("Can not register testsbp channel: %v", testsbp.Registered)
	}

	if alfaaft.Registered != nil {
		log.Fatalf("Can not register alfaaft channel: %v", alfaaft.Registered)
	}

	if alfabank.Registered != nil {
		log.Fatalf("Can not register alfabank channel: %v", alfabank.Registered)
	}

	if vbrrecom.Registered != nil {
		log.Fatalf("Can not register vbrr ecom channel: %v", vbrrecom.Registered)
	}

	if gpb.Registered != nil {
		log.Fatalf("Can not register gpb channel: %v", gpb.Registered)
	}

	if gpbsbp.Registered != nil {
		log.Fatalf("Can not register gpbsbp channel: %v", gpbsbp.Registered)
	}

	if err := plugins.RegisterBankChannels(channelStore); err != nil {
		log.Fatalf("Failed to register bank channels: %v", err)
	}

	if err := plugins.CheckBankChannels(channelStore); err != nil {
		log.Fatalf("Failed to check bank channels: %v", err)
	}

	if err := plugins.RegisterPaymentInstruments(instrumentStore); err != nil {
		log.Fatalf("Failed to register payment instruments: %v", err)
	}

	if err := plugins.CheckPaymentInstruments(instrumentStore); err != nil {
		log.Fatalf("Failed to check payment instruments: %v", err)
	}

	if err := plugins.RegisterRouters(routerStore); err != nil {
		log.Fatalf("Failed to register routers: %v", err)
	}

	if err := plugins.CheckRouters(routerStore); err != nil {
		log.Fatalf("Failed to check routers: %v", err)
	}
}
