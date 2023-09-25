package sbp

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	"github.com/serg666/service-repository"

	"github.com/serg666/ecom-service/plugins"
	"github.com/serg666/ecom-service/plugins/instruments"
)

var (
	Id         = 2
	Key        = "sbp"
	Registered = plugins.RegisterPaymentInstrument(Id, Key, func(
		instrumentStore interface{},
		logger repository.LoggerFunc,
	) instruments.PaymentInstrument {
		return &Sbp{
			instrumentStore: instrumentStore,
			logger:          logger,
		}
	})
)

type Sbp struct {
	instrumentStore interface{}
	logger          repository.LoggerFunc
}

func (sbp *Sbp) MarshalInstrumentJSON(c *gin.Context, instrumentId *int) ([]byte, error) {
	return json.Marshal(&struct{}{})
}

func (sbp *Sbp) FromRequest(c *gin.Context, request interface{}, requesterFunc instruments.InstrumentRequesterFunc) (interface{}, error) {
	return nil, nil
}
