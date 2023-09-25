package gpb

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	gpbclient "github.com/serg666/gpb-client"
	repository "github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/client"
	"github.com/serg666/ecom-service/config"
	"github.com/serg666/ecom-service/plugins"
	"github.com/serg666/ecom-service/plugins/channels"
	bankcard "github.com/serg666/ecom-service/plugins/instruments/card"
	"github.com/serg666/ecom-service/validators"
)

type GPBEcomSettings struct {
	PortalID   string `json:"portal_id"`
	MerchantID string `json:"merchant_id"`
	Password   string `json:"password"`
}

type ThreeDSMethodData struct {
	NotificationUrl string `json:"threeDSMethodNotificationURL"`
	ServerTransID   string `json:"threeDSServerTransID"`
}

var (
	Id         = 4
	Key        = "gpb"
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
			return fmt.Errorf("gpb channel not sutable for instrument <%d>", *instrument.Id), nil
		}

		jsonbody, err := json.Marshal(account.Settings)
		if err != nil {
			return fmt.Errorf("can not marshal gpb account settings: %v", err), nil
		}

		d := json.NewDecoder(bytes.NewReader(jsonbody))
		d.DisallowUnknownFields()

		var gpbs GPBEcomSettings

		if err := d.Decode(&gpbs); err != nil {
			return fmt.Errorf("can not decode gpb account settings: %v", err), nil
		}

		return nil, &GPBChannel{
			Logger:           logger,
			TransactionStore: transactionStore,
			gpbClient: gpbclient.GPBClient{
				BaseUrl:     cfg.Gpb.Ecom.Url,
				PayApi:      cfg.Gpb.Ecom.PayApi,
				MerchantApi: cfg.Gpb.Ecom.MerchantApi,
				Client:      client.Client,
				PortalID:    gpbs.PortalID,
				MerchantID:  gpbs.MerchantID,
				Password:    gpbs.Password,
			},
		}
	})
)

type GPBChannel struct {
	Logger           repository.LoggerFunc
	TransactionStore ecom.TransactionRepository
	gpbClient        gpbclient.GPBClient
}

func (gpb *GPBChannel) Authorize(c interface{}, transaction *ecom.Transaction, request interface{}) error {
	logger := gpb.Logger(c)

	req, ok := request.(validators.CardAuthorizeRequest)
	if !ok {
		return errRequestWrongType
	}

	token, err := gpb.gpbClient.Token(logger)
	if err != nil {
		logger.Errorf("GPB. Create token err=%v", err)
		return errCreateToken
	}

	transaction.RemoteId = &token

	description := "Оплата услуги"
	if transaction.Description != nil {
		description = *transaction.Description
	}

	jsonBody := map[string]interface{}{
		"returnUrl":   req.ThreeDSVer2TermUrl,
		"description": description,
		"amount":      transaction.TotalAmountConverted(),
		"currency":    *transaction.CurrencyConverted.CharCode,
		"state": map[string]string{
			"redirect": "post_params",
		},
		"3ds2": map[string]bool{
			"supported": true,
		},
		"src": map[string]string{
			"type":   "card",
			"pan":    req.Card.Pan,
			"expiry": req.Card.ExpDate.AsTime().Format("0106"),
			"csc":    req.Card.Cvv,
		},
	}

	if gbpExtraData, ok := req.ExtraData["gpb"]; ok {
		appendExtraData(jsonBody, gbpExtraData)
	}

	init_payment_result, err := gpb.gpbClient.InitPayment(logger, token, jsonBody)
	if err != nil {
		logger.Errorf("GPB. Init payment err=%v", err)
		return errInitPayment
	}

	if init_payment_result.State == "3ds2_prepare" {
		jsonBody := map[string]interface{}{
			"version": "2.1.0",

			"challengeWindowSize": "02",

			"device": map[string]interface{}{
				"channel":             "BRW",
				"browserAcceptHeader": transaction.BrowserInfo.AcceptHeader,
				"browserIP":           transaction.BrowserInfo.IP,
				"browserJavaEnabled":  *transaction.BrowserInfo.JavaEnabled,
				"browserLanguage":     transaction.BrowserInfo.Language,
				"browserColorDepth":   *transaction.BrowserInfo.ColorDepth,
				"browserScreenWidth":  *transaction.BrowserInfo.ScreenWidth,
				"browserScreenHeight": *transaction.BrowserInfo.ScreenHeight,
				"browserTZ":           *transaction.BrowserInfo.TZ,
				"browserUserAgent":    transaction.BrowserInfo.UserAgent,
			},
		}

		_, err := gpb.gpbClient.Accept3ds2Prepare(logger, *transaction.RemoteId, jsonBody)
		if err != nil {
			logger.Errorf("GPB. 3DS2 prepare-accept err=%v", err)
			return err3ds2PrepareAccept
		}
	}

	return gpb.updateTransaction(c, transaction)
}

func (gpb *GPBChannel) PreAuthorize(c interface{}, transaction *ecom.Transaction, request interface{}) error {
	return errUnimplemented
}

func (gpb *GPBChannel) Confirm(c interface{}, transaction *ecom.Transaction) error {
	return errUnimplemented
}

func (gpb *GPBChannel) Reverse(c interface{}, transaction *ecom.Transaction) error {
	return errUnimplemented
}

func (gpb *GPBChannel) Refund(c interface{}, transaction *ecom.Transaction) error {
	logger := gpb.Logger(c)

	err, successRefsTotal := gpb.TransactionStore.TypeTurnOver(c, ecom.NewTransactionSpecificationByReferenceIdAndStatus(
		*transaction.Reference.Id,
		repository.SUCCESS,
	))
	logger.Printf("successRefsTotal: %v (%v)", successRefsTotal, err)

	if err != nil {
		return fmt.Errorf("can not get gpb success references total: %v", err)
	}

	var refunds uint

	if refundTurnOver, ok := (*successRefsTotal)[repository.REFUND]; ok {
		refunds = refundTurnOver.Sum
	}

	availableAmount := transaction.Reference.TotalAmount() - refunds
	if availableAmount < transaction.TotalAmount() {
		return fmt.Errorf("incorrect amount: %d, availabe to refund: %d", transaction.TotalAmount(), availableAmount)
	}

	result, err := gpb.gpbClient.Refund(logger, transaction.TotalAmountConverted(), *transaction.Reference.RemoteId, *transaction.Reference.RRN)
	if err != nil {
		return fmt.Errorf("can not refund gbp transaction: %v", err)
	}

	transaction.ResponseCode = &result.Message.RefundResponse.Result.Code
	if result.Message.RefundResponse.Result.Code != "1" {
		return errors.New(result.Message.RefundResponse.Result.Desc)
	}

	transaction.AuthCode = &result.Message.RefundResponse.Result.Refund.AuthCode
	transaction.RRN = &result.Message.RefundResponse.Result.Refund.RRN
	transaction.Success()

	return nil
}
func (gpb *GPBChannel) Rebill(c interface{}, transaction *ecom.Transaction) error {
	return errUnimplemented
}

func (gpb *GPBChannel) ProcessCres(c interface{}, transaction *ecom.Transaction, cres string) error {
	// Cres не получилось направить на бек. Проверяется на стороне банка
	return gpb.updateTransaction(c, transaction)
}

func (gpb *GPBChannel) ProcessPares(c interface{}, transaction *ecom.Transaction, pares string) error {
	logger := gpb.Logger(c)

	md := fmt.Sprintf("%s_%d", *transaction.OrderId, *transaction.Id)

	_, err := gpb.gpbClient.MerchantReturn(logger, *transaction.RemoteId, pares, md)
	if err != nil {
		logger.Errorf("GPB. Merchant return err=%v", err)
		return errMerchantReturn
	}

	return gpb.updateTransaction(c, transaction)
}

func (gpb *GPBChannel) CompleteMethodUrl(c interface{}, transaction *ecom.Transaction, completed bool) error {
	return gpb.updateTransaction(c, transaction)
}

func (gpb *GPBChannel) Update(c interface{}, transaction *ecom.Transaction, force bool) {
	logger := gpb.Logger(c)

	if transaction.InFinalState() && !force {
		logger.Printf("gpb transaction %v has already got final state: %v", *transaction.Id, *transaction.Status)
		return
	}

	if err := gpb.updateTransaction(c, transaction); err != nil {
		logger.Errorf("failed to update gpb transaction: %v", err)
	}
}

func (gpb *GPBChannel) updateTransaction(c interface{}, transaction *ecom.Transaction) error {
	logger := gpb.Logger(c)

	result, err := gpb.gpbClient.GetState(logger, transaction.SafeRemoteId())
	if err != nil {
		return err
	}

	if result.State == "iframe" {
		if result.Post.CReq != "" {
			transaction.ThreeDSecure20 = &ecom.ThreeDSecure20{
				AcsUrl: &result.Url,
				Creq:   &result.Post.CReq,
			}
			transaction.Wait3DS()
		} else if result.Post.ThreeDSMethodData != "" {
			methodData, err := gpb.overrideNotificationUrl(result.Post.ThreeDSMethodData, transaction)
			if err != nil {
				gpb.Logger(c).Error(err)
				return err
			}

			transaction.ThreeDSMethodUrl = &ecom.ThreeDSMethodUrl{
				MethodUrl:         &result.Url,
				ThreeDSMethodData: &methodData,
			}
			transaction.WaitMethodUrl()
		}
	} else if result.State == "redirect" && result.Post.PaReq != "" {
		transaction.ThreeDSecure10 = &ecom.ThreeDSecure10{
			AcsUrl: &result.Url,
			PaReq:  &result.Post.PaReq,
		}
		transaction.Wait3DS()
	} else if result.State == "result" {
		transaction.ResponseCode = result.Result.ResponceCode
		transaction.AuthCode = result.Result.ApprovalCode
		transaction.RRN = result.Result.RRN
		if result.Result.Status == "FAILED" {
			transaction.Declined(&result.Result.ExtendedCode)
		} else if result.Result.Status == "DECLINED" {
			transaction.Declined(&result.Result.ExtendedCode)
		} else if result.Result.Status == "SUCCESS" {
			transaction.Success()
		}
	}

	return nil
}

func (gpb *GPBChannel) overrideNotificationUrl(threeDSMethodData string, transaction *ecom.Transaction) (string, error) {
	data, err := base64.StdEncoding.DecodeString(validBase64(threeDSMethodData))
	if err != nil {
		return "", err
	}

	var mData ThreeDSMethodData
	if err := json.Unmarshal(data, &mData); err != nil {
		return "", err
	}

	mData.NotificationUrl = fmt.Sprintf(
		"%s%s/%s/payment/%s/merchant-return",
		gpb.gpbClient.BaseUrl,
		gpb.gpbClient.PayApi,
		gpb.gpbClient.PortalID,
		*transaction.RemoteId,
	)

	b, err := json.Marshal(mData)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func appendExtraData(body map[string]interface{}, extraData interface{}) {
	extraDataAsStrMap, ok := extraData.(map[string]interface{})
	if !ok {
		return
	}
	for key, value := range extraDataAsStrMap {
		keyParts := strings.Split(key, ".")
		firstKey := keyParts[0]
		if _, ok := body[firstKey]; !ok {
			body[firstKey] = make(map[string]interface{}, 0)
		}
		field := body[firstKey].(map[string]interface{})
		field[strings.Join(keyParts[1:], ".")] = value
	}
}

func validBase64(jsonStr string) string {
	b64 := fmt.Sprintf("%s%s", jsonStr, strings.Repeat("=", len(jsonStr)%4))
	b64 = strings.ReplaceAll(b64, "_", "/")
	b64 = strings.ReplaceAll(b64, "-", "+")
	return b64
}
