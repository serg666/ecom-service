package bankcard

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/serg666/service-repository"

	"github.com/serg666/ecom-service/plugins"
	"github.com/serg666/ecom-service/plugins/instruments"
)

var (
	Id         = 1
	Key        = "card"
	Registered = plugins.RegisterPaymentInstrument(Id, Key, func(
		instrumentStore interface{},
		logger repository.LoggerFunc,
	) instruments.PaymentInstrument {
		return &BankCard{
			instrumentStore: instrumentStore,
			logger:          logger,
		}
	})
)

type ExpDate time.Time

func (t ExpDate) AsTime() time.Time {
	return time.Time(t)
}

func (t *ExpDate) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	parsed, err := time.Parse("2006-01-02", s)
	if err == nil {
		*t = ExpDate(parsed)
	}
	return
}

type Card struct {
	Pan     string  `json:"pan" binding:"required,luhncheck" example:"2201382000000021"`
	Cvv     string  `json:"cvv" binding:"required,iscvv" example:"123"`
	ExpDate ExpDate `json:"expire" binding:"required" example:"2024-12-31"`
	Holder  string  `json:"holder" binding:"required" example:"Card Holder"`
}

type CardInstrumentStore struct {
	CardRepository     repository.CardRepository
	CardMaskRepository repository.CardMaskRepository
}

type BankCard struct {
	instrumentStore interface{}
	logger          repository.LoggerFunc
}

func (bc *BankCard) MarshalInstrumentJSON(c *gin.Context, instrumentId *int) ([]byte, error) {
	cardInstrumentStore, ok := bc.instrumentStore.(CardInstrumentStore)
	if !ok {
		return nil, fmt.Errorf("cardInstrumentStore has wrong type")
	}

	if instrumentId == nil {
		return nil, fmt.Errorf("cardInstrumentId can not be null")
	}

	instId := *instrumentId

	err, _, cardMasks := cardInstrumentStore.CardMaskRepository.Query(c, repository.NewCardMaskSpecificationByID(instId))
	if err != nil {
		return nil, fmt.Errorf("card mask store quering failed: %v", err)
	}

	if len(cardMasks) == 0 {
		bc.logger(c).Warningf("card mask by ID <%d> not found", instId)
		return json.Marshal(nil)
	}

	cardMask := cardMasks[0]

	return json.Marshal(&struct {
		Mask    *string                 `json:"mask"`
		Token   *string                 `json:"token"`
		CardID  *int                    `json:"card_id"`
		Holder  *string                 `json:"holder"`
		ExpDate *repository.CardExpDate `json:"exp_date"`
	}{
		Mask:    cardMask.Mask,
		Token:   cardMask.Token,
		CardID:  cardMask.CardID,
		Holder:  cardMask.Holder,
		ExpDate: cardMask.ExpDate,
	})
}

func (bc *BankCard) FromRequest(c *gin.Context, request interface{}, requesterFunc instruments.InstrumentRequesterFunc) (interface{}, error) {
	err, bankCardRequest := requesterFunc(request)
	if err != nil {
		return nil, fmt.Errorf("can not get bank card request: %v")
	}

	bankCardReq, ok := bankCardRequest.(Card)
	if !ok {
		return nil, fmt.Errorf("bank card request has wrong type")
	}

	cardInstrumentStore, ok := bc.instrumentStore.(CardInstrumentStore)
	if !ok {
		return nil, fmt.Errorf("instrumentStore has wrong type")
	}

	pan := repository.PAN(bankCardReq.Pan)
	exp_dte := repository.ExpDate{bankCardReq.ExpDate.AsTime()}

	err, _, cards := cardInstrumentStore.CardRepository.Query(c, repository.NewCardSpecificationByPANEXPDTEHOLDER(pan, exp_dte, bankCardReq.Holder))
	if err != nil {
		return nil, fmt.Errorf("card store quering failed: %v", err)
	}

	if len(cards) > 0 {
		card := cards[0]

		err, _, cardMasks := cardInstrumentStore.CardMaskRepository.Query(c, repository.NewCardMaskSpecificationByCardID(*card.Id))
		if err != nil {
			return nil, fmt.Errorf("card mask store quering failed: %v", err)
		}

		if len(cardMasks) > 0 {
			return cardMasks[0], nil
		}

		mask := card.PAN.Mask()

		expire := *card.ExpDate
		cardMaskExpDate := repository.CardExpDate(expire.Time)
		cardMask := &repository.CardMask{
			CardID:  card.Id,
			Mask:    &mask,
			Token:   card.Token,
			Holder:  card.Holder,
			ExpDate: &cardMaskExpDate,
		}

		if err := cardInstrumentStore.CardMaskRepository.Add(c, cardMask); err != nil {
			return nil, fmt.Errorf("card mask store adding failed: %v", err)
		}

		return cardMask, nil
	}

	card := &repository.Card{
		PAN:     &pan,
		ExpDate: &exp_dte,
		Holder:  &bankCardReq.Holder,
	}

	if err := cardInstrumentStore.CardRepository.Add(c, card); err != nil {
		return nil, fmt.Errorf("card store adding failed: %v", err)
	}

	mask := card.PAN.Mask()

	cardMaskExpDate := repository.CardExpDate(exp_dte.Time)
	cardMask := &repository.CardMask{
		CardID:  card.Id,
		Mask:    &mask,
		Token:   card.Token,
		Holder:  card.Holder,
		ExpDate: &cardMaskExpDate,
	}

	if err := cardInstrumentStore.CardMaskRepository.Add(c, cardMask); err != nil {
		return nil, fmt.Errorf("card mask store adding failed: %v", err)
	}

	return cardMask, nil
}
