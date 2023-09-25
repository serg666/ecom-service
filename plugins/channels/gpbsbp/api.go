package gpbsbp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	gpbclient "github.com/serg666/gpb-client"
	repository "github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/client"
	"github.com/serg666/ecom-service/config"
	"github.com/serg666/ecom-service/plugins"
	"github.com/serg666/ecom-service/plugins/channels"
	"github.com/serg666/ecom-service/plugins/instruments/sbp"
	"github.com/serg666/ecom-service/validators"
)

var (
	Id         = 7
	Key        = "gpbsbp"
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
			return fmt.Errorf("gpbspb channel not sutable for instrument <%d>", *instrument.Id), nil
		}

		jsonbody, err := json.Marshal(account.Settings)
		if err != nil {
			return fmt.Errorf("can not marshal gpb sbp account settings: %v", err), nil
		}

		d := json.NewDecoder(bytes.NewReader(jsonbody))
		d.DisallowUnknownFields()

		var gpbSbpCfg GpbSbpSettings

		if err := d.Decode(&gpbSbpCfg); err != nil {
			return fmt.Errorf("can not decode gpb sbp account settings: %v", err), nil
		}

		pkPEMData, err := os.ReadFile(cfg.Gpb.Sbp.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to read gpb sbp private key file: %v", err), nil
		}

		return nil, &GpbSbpChannel{
			SessionStore:     sessionStore,
			InstrumentStore:  instrumentStore,
			TransactionStore: transactionStore,
			Logger:           logger,
			QrTtl:            cfg.Gpb.Sbp.QrTtl,
			gpbClient: gpbclient.GPBSbpClient{
				BaseUrl:    cfg.Gpb.Sbp.Url,
				PrivateKey: pkPEMData,
				Client:     client.Client,
				LegalID:    gpbSbpCfg.LegalID,
				MerchantID: gpbSbpCfg.MerchantID,
				Password:   gpbSbpCfg.Password,
				Account:    gpbSbpCfg.Account,
			},
		}
	})
)

type GpbSbpSettings struct {
	LegalID    string `json:"legal_id"`
	Account    string `json:"account"`
	MerchantID string `json:"merchant_id"`
	Password   string `json:"password"`
}

type GpbSbpChannel struct {
	QrTtl            int
	gpbClient        gpbclient.GPBSbpClient
	SessionStore     repository.SessionRepository
	TransactionStore ecom.TransactionRepository
	InstrumentStore  interface{}
	Logger           repository.LoggerFunc
}

func (ch *GpbSbpChannel) Authorize(c interface{}, transaction *ecom.Transaction, request interface{}) error {
	logger := ch.Logger(c)

	req, ok := request.(validators.SbpAuthorizeRequest)
	if !ok {
		return errors.New("GPB SBP. Request has wrong type")
	}

	description := "Оплата услуги"
	if transaction.Description != nil {
		description = *transaction.Description
	}

	jsonBody := map[string]interface{}{
		"templateVersion":               "01",
		"qrcType":                       "02",
		"qrTtl":                         strconv.Itoa(ch.QrTtl),
		"amount":                        transaction.TotalAmountConverted(),
		"currency":                      *transaction.CurrencyConverted.CharCode,
		"paymentPurpose":                description,
		"callbackMerchantNotifications": req.CallbackUrl,
	}

	sbp_result, err := ch.gpbClient.QrcData(logger, jsonBody)

	if err != nil {
		logger.Errorf("GPB SBP. QrcData err=%v", err)
		return errors.New("GPB SBP. QrcData failed")
	}

	transaction.ResponseCode = &sbp_result.Code

	if sbp_result.Code != "000" {
		logger.Errorf("GPB SBP. QrcData error code:%v message: %v", sbp_result.Code, sbp_result.Message)
		return fmt.Errorf("GPB SBP. QrcData failed. Response code: %v Message: %v", sbp_result.Code, sbp_result.Message)
	}

	transaction.SbpQR = &ecom.SbpQR{
		QrcId:   sbp_result.Data.QrcId,
		Payload: sbp_result.Data.Payload,
	}
	transaction.RemoteId = sbp_result.TrxId
	transaction.WaitSBP()

	return nil
}

func (ch *GpbSbpChannel) PreAuthorize(c interface{}, transaction *ecom.Transaction, request interface{}) error {
	return errors.New("Method not implemented yet")
}

func (ch *GpbSbpChannel) Confirm(c interface{}, transaction *ecom.Transaction) error {
	return errors.New("Method not implemented yet")
}

func (ch *GpbSbpChannel) Reverse(c interface{}, transaction *ecom.Transaction) error {
	return errors.New("Method not implemented yet")
}

func (ch *GpbSbpChannel) Refund(c interface{}, transaction *ecom.Transaction) error {
	logger := ch.Logger(c)

	err, successRefsTotal := ch.TransactionStore.TypeTurnOver(c, ecom.NewTransactionSpecificationByReferenceIdAndStatus(
		*transaction.Reference.Id,
		repository.SUCCESS,
	))
	logger.Printf("successRefsTotal: %v (%v)", successRefsTotal, err)

	if err != nil {
		return fmt.Errorf("can not get gpb sbp success references total: %v", err)
	}

	var refunds uint

	if refundTurnOver, ok := (*successRefsTotal)[repository.REFUND]; ok {
		refunds = refundTurnOver.Sum
	}

	availableAmount := transaction.Reference.TotalAmount() - refunds
	if availableAmount < transaction.TotalAmount() {
		return fmt.Errorf("incorrect amount: %d, availabe to refund: %d", transaction.TotalAmount(), availableAmount)
	}

	result, err := ch.gpbClient.Refund(
		logger,
		transaction.TotalAmountConverted(),
		*transaction.Reference.CurrencyConverted.CharCode,
		*transaction.Reference.RemoteId,
	)
	if err != nil {
		return fmt.Errorf("can not refund gbp sbp transaction: %v", err)
	}

	transaction.ResponseCode = &result.Code
	if result.Code != "000" {
		return errors.New(result.Message)
	}

	transaction.RemoteId = result.TrxId
	transaction.Success()

	return nil
}

func (ch *GpbSbpChannel) Rebill(c interface{}, transaction *ecom.Transaction) error {
	return errors.New("Method not implemented yet")
}

func (ch *GpbSbpChannel) ProcessCres(c interface{}, transaction *ecom.Transaction, cres string) error {
	return errors.New("Method not implemented yet")
}

func (ch *GpbSbpChannel) ProcessPares(c interface{}, transaction *ecom.Transaction, pares string) error {
	return errors.New("Method not implemented yet")
}

func (ch *GpbSbpChannel) CompleteMethodUrl(c interface{}, transaction *ecom.Transaction, completed bool) error {
	return errors.New("Method not implemented yet")
}

func (ch *GpbSbpChannel) Update(c interface{}, transaction *ecom.Transaction, force bool) {
	logger := ch.Logger(c)

	if transaction.InFinalState() && !force {
		logger.Printf("gpb sbp transaction %v has already got final state: %v", *transaction.Id, *transaction.Status)
		return
	}

	sbp_result, err := ch.gpbClient.QrcStatus(logger, transaction.SafeQrcId())

	if err != nil {
		logger.Errorf("GPB SBP. QrcStatus err=%v", err)
		return
	}

	transaction.ResponseCode = &sbp_result.Code

	if sbp_result.Code != "000" {
		logger.Errorf("GPB SBP. QrcStatus error code:%v message: %v", sbp_result.Code, sbp_result.Message)
		return
	}

	data := *sbp_result.Data
	switch *data[0].Status {
	case "ACCEPTED":
		if data[0].TrxId != nil {
			// @note: only trxid with ACCEPTED state give us success
			transaction.RemoteId = data[0].TrxId
			transaction.Success()
		}
	case "REJECTED":
		transaction.ResponseCode = &data[0].Code
		transaction.Declined(&data[0].Message)
	}
}
