package alfaaft

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
	"github.com/serg666/ecom-service/plugins/instruments/card"
	"github.com/serg666/ecom-service/validators"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const EXPIRE_DATE_FORMAT = "2006-01-02"

var (
	Id         = 5
	Key        = "alfaaft"
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
			return fmt.Errorf("alfabank aft channel not sutable for instrument <%d>", *instrument.Id), nil
		}

		jsonbody, err := json.Marshal(account.Settings)
		if err != nil {
			return fmt.Errorf("can not marshal alfabank aft account settings: %v", err), nil
		}

		d := json.NewDecoder(bytes.NewReader(jsonbody))
		d.DisallowUnknownFields()

		var alfaAftAccountSettings AlfaAftAccountSettings

		if err := d.Decode(&alfaAftAccountSettings); err != nil {
			return fmt.Errorf("can not decode alfabank aft account settings: %v", err), nil
		}

		alfaAftCfg := &AlfaAftSettings{
			Url: cfg.Alfabank.Aft.Url,
		}

		return nil, &AlfaAftChannel{
			Cfg:              alfaAftCfg,
			Logger:           logger,
			InstrumentStore:  instrumentStore,
			SessionStore:     sessionStore,
			TransactionStore: transactionStore,
			Settings:         &alfaAftAccountSettings,
		}
	})
)

type AlfaAftHTTP200Response struct {
	State        *string `json:"state"`
	AuthCode     *string `json:"authcode"`
	AcsUrl       *string `json:"acsurl"`
	MD           *string `json:"md"`
	PaReq        *string `json:"pareq"`
	ErrorCode    *string `json:"error_code"`
	ErrorMessage *string `json:"error_message"`
}

type AlfaAftAccountSettings struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type AlfaAftSettings struct {
	Url string
}

type AlfaAftChannel struct {
	Cfg              *AlfaAftSettings
	Logger           repository.LoggerFunc
	InstrumentStore  interface{}
	SessionStore     repository.SessionRepository
	TransactionStore ecom.TransactionRepository
	Settings         *AlfaAftAccountSettings
}

func (channel *AlfaAftChannel) maskParams(data string) string {
	sampleRegexp := regexp.MustCompile(`"pan":"[^"]+([^"]{4})"`)
	result := sampleRegexp.ReplaceAllString(data, "\"pan\":\"******$1\"")
	sampleRegexp = regexp.MustCompile(`"cvv":"[^"]+"`)
	result = sampleRegexp.ReplaceAllString(result, "\"cvv\":\"***\"")
	sampleRegexp = regexp.MustCompile(`"recipient_pan":"[^"]+([^"]{4})"`)
	result = sampleRegexp.ReplaceAllString(result, "\"recipient_pan\":\"******$1\"")
	sampleRegexp = regexp.MustCompile(`"password":"[^"]+"`)
	result = sampleRegexp.ReplaceAllString(result, "\"password\":\"******\"")
	return result
}

func (channel *AlfaAftChannel) makeRequest(
	c interface{},
	method string,
	uri string,
	contentType string,
	data string,
) (*[]byte, *int, error) {
	url := fmt.Sprintf("%s/%s", channel.Cfg.Url, uri)
	channel.Logger(c).Printf("Requesting: %s", channel.maskParams(url))
	channel.Logger(c).Printf("Params: %s", channel.maskParams(data))
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

	channel.Logger(c).Printf("response body: %s", channel.maskParams(string(body)))

	return &body, &res.StatusCode, nil
}

func (channel *AlfaAftChannel) Update(c interface{}, transaction *ecom.Transaction, force bool) {
	if transaction.InFinalState() && !force {
		channel.Logger(c).Warningf("transaction <%d> has already got final state: %s", *transaction.Id, *transaction.Status)
		return
	}

	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"login":    channel.Settings.Login,
			"password": channel.Settings.Password,
		},
		"payment_id": strconv.Itoa(*transaction.Id),
	}

	jsonbody, err := json.Marshal(body)
	if err != nil {
		channel.Logger(c).Warningf("can not marshal alfabank aft info request body: %v", err)
		return
	}

	response, code, err := channel.makeRequest(c, "POST", "alfabank/transfer-status", "application/json; charset=utf-8", string(jsonbody))
	if err != nil {
		channel.Logger(c).Warningf("can not make alfabank aft info request: %v", err)
		return
	}

	channel.Logger(c).Printf("alfabank aft info response code: %v", *code)

	if *code != 200 {
		channel.Logger(c).Warningf("alfabank aft info response HTTP-code: %d", *code)
		return
	}

	d := json.NewDecoder(bytes.NewReader(*response))
	d.DisallowUnknownFields()

	var alfaAftHTTP200Response AlfaAftHTTP200Response

	if err := d.Decode(&alfaAftHTTP200Response); err != nil {
		channel.Logger(c).Warningf("can not decode alfabank aft info response: %v", err)
		return
	}

	if alfaAftHTTP200Response.State == nil {
		channel.Logger(c).Warningf("alfabank aft info response state is null")
		return
	}

	transaction.ResponseCode = alfaAftHTTP200Response.ErrorCode

	switch *alfaAftHTTP200Response.State {
	case "declined":
		transaction.Declined(alfaAftHTTP200Response.ErrorMessage)
	case "deposited":
		transaction.Success()
	}
}

func (channel *AlfaAftChannel) Authorize(c interface{}, transaction *ecom.Transaction, request interface{}) error {
	req, ok := request.(validators.CardAuthorizeRequest)
	if !ok {
		return errors.New("alfabank aft card authorization request has wrong type")
	}

	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"login":    channel.Settings.Login,
			"password": channel.Settings.Password,
		},
		"payment_id": strconv.Itoa(*transaction.Id),
		"ip":         transaction.BrowserInfo.IP,
		"amount":     transaction.CurrencyConverted.MinorToNominal(transaction.TotalAmountConverted()),
		"currency":   strconv.Itoa(*transaction.CurrencyConverted.NumericCode),
		"pan":        req.Card.Pan,
		"cvv":        req.Card.Cvv,
		"expire":     req.Card.ExpDate.AsTime().Format(EXPIRE_DATE_FORMAT),
	}

	jsonbody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("can not marshal alfabank aft card authorization request body: %v", err)
	}

	response, code, err := channel.makeRequest(c, "POST", "alfabank/attempt-transfer", "application/json; charset=utf-8", string(jsonbody))
	if err != nil {
		return fmt.Errorf("can not make alfabank aft card authorization request: %v", err)
	}

	channel.Logger(c).Printf("alfabank aft card authorization response code: %v", *code)

	if *code == 200 {
		d := json.NewDecoder(bytes.NewReader(*response))
		d.DisallowUnknownFields()

		var alfaAftHTTP200Response AlfaAftHTTP200Response

		if err := d.Decode(&alfaAftHTTP200Response); err != nil {
			return fmt.Errorf("can not decode alfabank aft card authorization response: %v", err)
		}

		if alfaAftHTTP200Response.State != nil {
			transaction.ResponseCode = alfaAftHTTP200Response.ErrorCode
			transaction.AuthCode = alfaAftHTTP200Response.AuthCode

			switch *alfaAftHTTP200Response.State {
			case "wait3ds":
				if err := channel.SessionStore.Add(c, repository.NewSession(
					fmt.Sprintf("3ds_session_%d", *transaction.Id),
					repository.SessionData{
						"MD": *alfaAftHTTP200Response.MD,
					},
				)); err != nil {
					return fmt.Errorf("can not add alfabank aft 3ds session: %v", err)
				}

				transaction.ThreeDSecure10 = &ecom.ThreeDSecure10{
					AcsUrl: alfaAftHTTP200Response.AcsUrl,
					PaReq:  alfaAftHTTP200Response.PaReq,
				}
				transaction.Wait3DS()
			case "deposited":
				transaction.Success()
			case "declined":
				transaction.Declined(alfaAftHTTP200Response.ErrorMessage)
			default:
				channel.Update(c, transaction, false)
			}

			return nil
		}
	}

	channel.Update(c, transaction, false)

	return nil
}

func (channel *AlfaAftChannel) PreAuthorize(c interface{}, transaction *ecom.Transaction, request interface{}) error {
	return errors.New("Alfabank aft preauthorize not implemented")
}

func (channel *AlfaAftChannel) Confirm(c interface{}, transaction *ecom.Transaction) error {
	return errors.New("Alfabank aft confirm not implemented")
}

func (channel *AlfaAftChannel) Reverse(c interface{}, transaction *ecom.Transaction) error {
	return errors.New("Alfabank aft reverse not implemented")
}

func (channel *AlfaAftChannel) Refund(c interface{}, transaction *ecom.Transaction) error {
	return errors.New("Alfabank aft refund not implemented")
}

func (channel *AlfaAftChannel) Rebill(c interface{}, transaction *ecom.Transaction) error {
	return errors.New("Alfabank aft rebill not implemented")
}

func (channel *AlfaAftChannel) ProcessCres(c interface{}, transaction *ecom.Transaction, cres string) error {
	return errors.New("Alfabank aft process cres not implemented")
}

func (channel *AlfaAftChannel) ProcessPares(c interface{}, transaction *ecom.Transaction, pares string) error {
	sessionKey := fmt.Sprintf("3ds_session_%d", *transaction.Id)
	err, _, sessions := channel.SessionStore.Query(c, repository.NewSessionSpecificationByKey(sessionKey))

	if err != nil {
		return fmt.Errorf("alfabank aft failed to query session store: %v", err)
	}

	if len(sessions) == 0 {
		return fmt.Errorf("alfabank aft session with key %s not found", sessionKey)
	}

	session := sessions[0]
	sessionData := session.Data

	md, ok := (*sessionData)["MD"]
	if !ok {
		return errors.New("alfabank aft session data has not MD")
	}

	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"login":    channel.Settings.Login,
			"password": channel.Settings.Password,
		},
		"ip":    transaction.BrowserInfo.IP,
		"md":    md,
		"pares": pares,
	}

	jsonbody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("can not marshal alfabank aft process pares request body: %v", err)
	}

	response, code, err := channel.makeRequest(c, "POST", "alfabank/complete-transfer", "application/json; charset=utf-8", string(jsonbody))
	if err != nil {
		return fmt.Errorf("can not make alfabank aft process pares request: %v", err)
	}

	channel.Logger(c).Printf("alfabank aft process pares response code: %v", *code)

	if *code == 200 {
		d := json.NewDecoder(bytes.NewReader(*response))
		d.DisallowUnknownFields()

		var alfaAftHTTP200Response AlfaAftHTTP200Response

		if err := d.Decode(&alfaAftHTTP200Response); err != nil {
			return fmt.Errorf("can not decode alfabank aft process pares response: %v", err)
		}

		if alfaAftHTTP200Response.State != nil {
			transaction.ResponseCode = alfaAftHTTP200Response.ErrorCode
			transaction.AuthCode = alfaAftHTTP200Response.AuthCode

			switch *alfaAftHTTP200Response.State {
			case "deposited":
				transaction.Success()
			default:
				channel.Update(c, transaction, false)
			}

			return nil
		}
	}

	channel.Update(c, transaction, false)

	return nil
}

func (channel *AlfaAftChannel) CompleteMethodUrl(c interface{}, transaction *ecom.Transaction, completed bool) error {
	return errors.New("Alfabank aft complete method url not implemented")
}
