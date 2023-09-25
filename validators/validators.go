package validators

import (
	"fmt"

	ecom "github.com/serg666/service-repository/ecom"

	bankcard "github.com/serg666/ecom-service/plugins/instruments/card"
)

type SbpAuthorizeRequest struct {
	OrderId     string  `json:"order_id" binding:"required" example:"1"`
	Amount      uint    `json:"amount" binding:"required,min=1"`
	Commission  uint    `json:"commission"`
	Customer    string  `json:"customer" binding:"required" example:"some@mail.ru"`
	CallbackUrl string  `json:"callbackurl" binding:"required" example:"http://some.domain.com/callback"`
	Description *string `json:"description"`
}

func SbpAuthorizationInstrumentRequester(request interface{}) (error, interface{}) {
	return nil, nil
}

type SbpPreAuthorizeRequest struct {
	SbpAuthorizeRequest
}

func SbpPreAuthorizationInstrumentRequester(request interface{}) (error, interface{}) {
	return nil, nil
}

type CardAuthorizeRequest struct {
	OrderId            string                 `json:"order_id" binding:"required" example:"1"`
	Amount             uint                   `json:"amount" binding:"required,min=1"`
	Commission         uint                   `json:"commission"`
	Customer           string                 `json:"customer" binding:"required" example:"some@mail.ru"`
	Card               bankcard.Card          `json:"card" binding:"required"`
	ThreeDSVer2TermUrl string                 `json:"threedsver2termurl" binding:"required" example:"http://some.domain.com/processcres"`
	BrowserInfo        ecom.BrowserInfo        `json:"browser_info" binding:"required"`
	Description        *string                `json:"description"`
	ExtraData          map[string]interface{} `json:"extra_data"`
}

func CardAuthorizationInstrumentRequester(request interface{}) (error, interface{}) {
	cardAuthorizeRequest, ok := request.(CardAuthorizeRequest)
	if !ok {
		return fmt.Errorf("card authorization request has wrong type"), nil
	}

	return nil, cardAuthorizeRequest.Card
}

type CardPreAuthorizeRequest struct {
	CardAuthorizeRequest
}

func CardPreAuthorizationInstrumentRequester(request interface{}) (error, interface{}) {
	cardPreAuthorizeRequest, ok := request.(CardPreAuthorizeRequest)
	if !ok {
		return fmt.Errorf("card preauthorization request has wrong type"), nil
	}

	return nil, cardPreAuthorizeRequest.Card
}

type CompleteMethodUrlRequest struct {
	Completed *bool `json:"completed" binding:"required"`
}

type ProcessCresRequest struct {
	Cres string `json:"cres" binding:"required,notempty"`
}

type ProcessParesRequest struct {
	Pares string `json:"pares" binding:"required,notempty"`
}

type ConfirmPreAuthRequest struct {
	Amount     uint `json:"amount" binding:"required,min=1"`
	Commission uint `json:"commission"`
}

func (req ConfirmPreAuthRequest) TotalAmount() uint {
	return req.Amount + req.Commission
}

type ReversalRequest struct {
	ConfirmPreAuthRequest
}

type RefundRequest struct {
	ConfirmPreAuthRequest
}

type RebillRequest struct {
	ConfirmPreAuthRequest
}
