package instruments

import (
	"github.com/gin-gonic/gin"
)

type InstrumentRequesterFunc func(interface{}) (error, interface{})

type PaymentInstrument interface {
	MarshalInstrumentJSON(c *gin.Context, instrumentId *int) ([]byte, error)
	FromRequest(c *gin.Context, request interface{}, requesterFunc InstrumentRequesterFunc) (interface{}, error)
}

type InstrumentInstance struct {
	ctx *gin.Context
	api PaymentInstrument
	id  *int
}

func (i *InstrumentInstance) MarshalJSON() ([]byte, error) {
	return i.api.MarshalInstrumentJSON(i.ctx, i.id)
}

func NewInstrumentInstance(ctx *gin.Context, api PaymentInstrument, id *int) *InstrumentInstance {
	return &InstrumentInstance{
		ctx: ctx,
		api: api,
		id:  id,
	}
}
