package testsbp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"

	"github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/client"
	"github.com/serg666/ecom-service/config"
	"github.com/serg666/ecom-service/plugins"
	"github.com/serg666/ecom-service/plugins/channels"
	"github.com/serg666/ecom-service/plugins/instruments/sbp"

	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

var (
	Id         = 6
	Key        = "testsbp"
	Registered = plugins.RegisterBankChannel(Id, Key, func(
		cfg *config.Config,
		account *ecom.Account,
		instrument *repository.Instrument,
		instrumentStore interface{},
		sessionStore repository.SessionRepository,
		transactionStore ecom.TransactionRepository,
		logger repository.LoggerFunc,
	) (error, channels.BankChannel) {
		if *instrument.Id != sbp.Id {
			return fmt.Errorf("testspb channel not sutable for instrument <%d>", *instrument.Id), nil
		}

		testSbpCfg := &TestSbpSettings{
			Url: cfg.Testbank.Sbp.Url,
		}

		return nil, &TestSbpChannel{
			Cfg:              testSbpCfg,
			SessionStore:     sessionStore,
			InstrumentStore:  instrumentStore,
			TransactionStore: transactionStore,
			Logger:           logger,
		}
	})
)

type TestSbpQrHTTP200Response struct {
	QrcId   *string `json:"qrId"`
	Payload *string `json:"qrUrl"`
}

type TestSbpSettings struct {
	Url string
}

type TestSbpChannel struct {
	Cfg              *TestSbpSettings
	SessionStore     repository.SessionRepository
	TransactionStore ecom.TransactionRepository
	InstrumentStore  interface{}
	Logger           repository.LoggerFunc
}

func (ch *TestSbpChannel) makeRequest(
	c interface{},
	method string,
	uri string,
	contentType string,
	data string,
) (*[]byte, *int, error) {
	url := fmt.Sprintf("%s/%s", ch.Cfg.Url, uri)
	ch.Logger(c).Printf("Requesting: %s", url)
	ch.Logger(c).Printf("Params: %s", data)
	r, err := http.NewRequest(method, url, strings.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("can not make new request: %v", err)
	}

	r.Header.Add("Content-Type", contentType)
	r.Header.Add("Content-Length", strconv.Itoa(len(data)))

	if c != nil {
		if ctx, ok := c.(*gin.Context); ok {
			if rid := requestid.Get(ctx); len(rid) > 0 {
				r.Header.Add("X-Request-ID", rid)
			}
		}
	}

	res, err := client.Client.Do(r)
	if err != nil {
		return nil, nil, fmt.Errorf("can not do request: %v", err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("can not read body: %v", err)
	}

	ch.Logger(c).Printf("response body: %s", string(body))

	return &body, &res.StatusCode, nil
}

func (ch *TestSbpChannel) Authorize(c interface{}, transaction *ecom.Transaction, request interface{}) error {
	body := map[string]interface{}{
		"amount": transaction.TotalAmountConverted(),
	}

	jsonbody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("can not marshal testbank sbp authorization request body: %v", err)
	}

	response, code, err := ch.makeRequest(c, "POST", "api/qr/create", "application/json; charset=utf-8", string(jsonbody))
	if err != nil {
		return fmt.Errorf("can not make testbank sbp authorization request: %v", err)
	}

	ch.Logger(c).Printf("testbank sbp authorization response code: %v", *code)

	if *code != 200 {
		return fmt.Errorf("wrong testbank sbp responce code: %v", *code)
	}

	d := json.NewDecoder(bytes.NewReader(*response))
	d.DisallowUnknownFields()

	var testSbpQrHTTP200Response TestSbpQrHTTP200Response

	if err := d.Decode(&testSbpQrHTTP200Response); err != nil {
		return fmt.Errorf("can not decode testbank sbp authorization response: %v", err)
	}

	transaction.SbpQR = &ecom.SbpQR{
		QrcId:   testSbpQrHTTP200Response.QrcId,
		Payload: testSbpQrHTTP200Response.Payload,
	}
	transaction.WaitSBP()

	return nil
}

func (ch *TestSbpChannel) PreAuthorize(c interface{}, transaction *ecom.Transaction, request interface{}) error {
	return ch.Authorize(c, transaction, request)
}

func (ch *TestSbpChannel) Confirm(c interface{}, transaction *ecom.Transaction) error {
	ch.Logger(c).Print("confirm int")
	transaction.Success()
	return nil
}

func (ch *TestSbpChannel) Reverse(c interface{}, transaction *ecom.Transaction) error {
	ch.Logger(c).Print("reverse int")
	transaction.Success()
	return nil
}

func (ch *TestSbpChannel) Refund(c interface{}, transaction *ecom.Transaction) error {
	ch.Logger(c).Print("refund int")
	transaction.Success()
	return nil
}

func (ch *TestSbpChannel) Rebill(c interface{}, transaction *ecom.Transaction) error {
	ch.Logger(c).Print("rebill int")
	transaction.Success()
	return nil
}

func (ch *TestSbpChannel) ProcessCres(c interface{}, transaction *ecom.Transaction, cres string) error {
	return errors.New("Method not implemented yet")
}

func (ch *TestSbpChannel) ProcessPares(c interface{}, transaction *ecom.Transaction, pares string) error {
	return errors.New("Method not implemented yet")
}

func (ch *TestSbpChannel) CompleteMethodUrl(c interface{}, transaction *ecom.Transaction, completed bool) error {
	return errors.New("Method not implemented yet")
}

func (ch *TestSbpChannel) Update(c interface{}, transaction *ecom.Transaction, force bool) {
	ch.Logger(c).Print("update int")
	transaction.Success()
	return
}
