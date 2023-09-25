package bpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	ua "github.com/mileusna/useragent"

	"github.com/serg666/service-repository"
	ecom "github.com/serg666/service-repository/ecom"

	"github.com/serg666/ecom-service/client"
	"github.com/serg666/ecom-service/plugins/instruments/card"
	"github.com/serg666/ecom-service/validators"
)

type ClientInfo struct {
	UserAgent             string `json:"userAgent"`
	OS                    string `json:"os"`
	OSVersion             string `json:"osversion"`
	Device                string `json:"device"`
	Mobile                bool   `json:"mobile"`
	ScreenPrint           string `json:"screenPrint"`
	ColorDepth            int    `json:"colorDepth"`
	ScreenHeight          int    `json:"screenHeight"`
	ScreenWidth           int    `json:"screenWidth"`
	JavaEnabled           bool   `json:"javaEnabled"`
	JavascriptEnabled     bool   `json:"javascriptEnabled"`
	BrowserLanguage       string `json:"browserLanguage"`
	BrowserTimeZone       string `json:"browserTimeZone"`
	BrowserTimeZoneOffset int    `json:"browserTimeZoneOffset"`
}

type BpcSettings struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type BpcEcomSettings struct {
	Url string
}

type BpcChannel struct {
	Cfg              *BpcEcomSettings
	Logger           repository.LoggerFunc
	InstrumentStore  interface{}
	SessionStore     repository.SessionRepository
	TransactionStore ecom.TransactionRepository
	Settings         *BpcSettings
}

func (bpc *BpcChannel) maskParams(data string) string {
	sampleRegexp := regexp.MustCompile(`PAN=[^&]+([^&]{4})`)
	result := sampleRegexp.ReplaceAllString(data, "PAN=******$1")
	sampleRegexp = regexp.MustCompile(`CVC=[^&]+`)
	result = sampleRegexp.ReplaceAllString(result, "CVC=***")
	sampleRegexp = regexp.MustCompile(`password=[^&]+`)
	result = sampleRegexp.ReplaceAllString(result, "password=******")
	return result
}

func (bpc *BpcChannel) makeRequest(
	c interface{},
	method string,
	url string,
	data string,
) (error, *map[string]interface{}) {
	uri := fmt.Sprintf("%s/%s", bpc.Cfg.Url, url)
	bpc.Logger(c).Printf("Requesting: %s", uri)
	bpc.Logger(c).Printf("Params: %s", bpc.maskParams(data))
	r, err := http.NewRequest(method, uri, strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("can not make new request: %v", err), nil
	}

	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data)))

	res, err := client.Client.Do(r)
	if err != nil {
		return fmt.Errorf("can not do request: %v", err), nil
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("can not read body: %v", err), nil
	}

	bpc.Logger(c).Printf("response body: %s", string(body))

	var jsonResp map[string]interface{}
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return fmt.Errorf("can not unmarshal body: %v", err), nil
	}

	return nil, &jsonResp
}

func (bpc *BpcChannel) putBrowserInfo(
	c interface{},
	browserInfo *ecom.BrowserInfo,
	serverUrl *string,
	transId *string,
) error {
	bpc.Logger(c).Debugf("putting browserInfo: %v", browserInfo)

	if serverUrl == nil {
		return errors.New("server URL is null")
	}

	bpc.Logger(c).Debugf("requesting: %s", *serverUrl)
	data := url.Values{}

	r, err := http.NewRequest("POST", *serverUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("can not make new request: %v", err)
	}

	r.Header.Add("User-Agent", browserInfo.UserAgent)
	r.Header.Add("Accept", browserInfo.AcceptHeader)
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	res, err := client.Client.Do(r)
	if err != nil {
		return fmt.Errorf("can not do request: %v", err)
	}
	defer res.Body.Close()

	bpc.Logger(c).Debugf("response code: %d", res.StatusCode)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("can not read response body: %v", err)
	}

	bpc.Logger(c).Debugf("response body: %s", string(body))
	re := regexp.MustCompile(`(https?://[^\"\s>]+)`)
	links := re.FindAll(body, -1)
	bpc.Logger(c).Printf("links: %q", links)
	if len(links) == 0 {
		return errors.New("no links has been found")
	}

	clientUrl := string(links[len(links)-1])
	bpc.Logger(c).Printf("link: %s", clientUrl)
	bpc.Logger(c).Debugf("requesting: %s", clientUrl)
	ua := ua.Parse(browserInfo.UserAgent)
	clientInfo := ClientInfo{
		UserAgent:             ua.String,
		OS:                    ua.OS,
		OSVersion:             ua.OSVersion,
		Device:                ua.Device,
		Mobile:                ua.Mobile,
		ScreenPrint:           browserInfo.ScreenPrint,
		ColorDepth:            *browserInfo.ColorDepth,
		ScreenHeight:          *browserInfo.ScreenHeight,
		ScreenWidth:           *browserInfo.ScreenWidth,
		JavaEnabled:           *browserInfo.JavaEnabled,
		JavascriptEnabled:     true,
		BrowserLanguage:       browserInfo.Language,
		BrowserTimeZone:       browserInfo.TimeZone,
		BrowserTimeZoneOffset: *browserInfo.TZ,
	}

	jsonbody, err := json.Marshal(clientInfo)
	if err != nil {
		return fmt.Errorf("can not marshal client info: %v", err)
	}

	bpc.Logger(c).Printf("json body: %s", string(jsonbody))
	postData := url.Values{}
	postData.Set("threeDSServerTransID", *transId)
	postData.Set("clientInfo", string(jsonbody))
	nr, err := http.NewRequest("POST", clientUrl, strings.NewReader(postData.Encode()))
	if err != nil {
		return fmt.Errorf("can not make client info request: %v", err)
	}

	nr.Header.Add("User-Agent", browserInfo.UserAgent)
	nr.Header.Add("Accept", browserInfo.AcceptHeader)
	nr.Header.Add("Referer", *serverUrl)
	nr.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	nr.Header.Add("Content-Length", strconv.Itoa(len(postData.Encode())))

	nres, err := client.Client.Do(nr)
	if err != nil {
		return fmt.Errorf("can not do client info request: %v", err)
	}
	defer nres.Body.Close()

	bpc.Logger(c).Debugf("client response code: %d", nres.StatusCode)
	nbody, err := ioutil.ReadAll(nres.Body)
	if err != nil {
		return fmt.Errorf("can not read client info response body: %v", err)
	}

	bpc.Logger(c).Debugf("client response body: %s", string(nbody))

	return nil
}

func (bpc *BpcChannel) isCREQ(c interface{}, response *map[string]interface{}) (bool, *string, *string) {
	var acsURL *string
	var cReq *string

	if acsUrl, ok := (*response)["acsUrl"]; ok {
		if acs, ok := acsUrl.(string); ok {
			acsURL = &acs
		} else {
			return false, nil, nil
		}
	} else {
		return false, nil, nil
	}

	if packedCReq, ok := (*response)["packedCReq"]; ok {
		if creq, ok := packedCReq.(string); ok {
			cReq = &creq
		} else {
			return false, nil, nil
		}
	} else {
		return false, nil, nil
	}

	return true, acsURL, cReq
}

func (bpc *BpcChannel) is3DS20(
	c interface{},
	response *map[string]interface{},
) (bool, *string, *string, *string, *string) {
	var threeDSServerTransId *string
	var threeDSMethodURLServer *string
	var threeDSMethodURL *string
	var threeDSMethodDataPacked *string
	var is3DS bool

	is3DS = false
	if is3DSVer2, ok := (*response)["is3DSVer2"]; ok {
		switch is3DS2 := is3DSVer2.(type) {
		case bool:
			is3DS = is3DS2
		case string:
			is3DS = is3DS2 == "true"
		default:
			bpc.Logger(c).Printf("is3DS2 type: %v (%T)", is3DS2, is3DS2)
		}
	}

	if serverTransId, ok := (*response)["threeDSServerTransId"]; ok {
		if transId, ok := serverTransId.(string); ok {
			threeDSServerTransId = &transId
		}
	}

	if methodURLServer, ok := (*response)["threeDSMethodURLServer"]; ok {
		if serverURL, ok := methodURLServer.(string); ok {
			threeDSMethodURLServer = &serverURL
		}
	}

	if methodURL, ok := (*response)["threeDSMethodURL"]; ok {
		if mURL, ok := methodURL.(string); ok {
			threeDSMethodURL = &mURL
		}
	}

	if methodDataPacked, ok := (*response)["threeDSMethodDataPacked"]; ok {
		if methodData, ok := methodDataPacked.(string); ok {
			threeDSMethodDataPacked = &methodData
		}
	}

	return is3DS, threeDSServerTransId, threeDSMethodURLServer, threeDSMethodURL, threeDSMethodDataPacked
}

func (bpc *BpcChannel) is3DS10(c interface{}, response *map[string]interface{}) (bool, *string, *string) {
	var acsURL *string
	var paREQ *string

	if acsUrl, ok := (*response)["acsUrl"]; ok {
		if acs, ok := acsUrl.(string); ok {
			acsURL = &acs
		} else {
			return false, nil, nil
		}
	} else {
		return false, nil, nil
	}

	if paReq, ok := (*response)["paReq"]; ok {
		if pareq, ok := paReq.(string); ok {
			paREQ = &pareq
		} else {
			return false, nil, nil
		}
	} else {
		return false, nil, nil
	}

	return true, acsURL, paREQ
}

func (bpc *BpcChannel) parseError(c interface{}, response *map[string]interface{}) (*string, *string) {
	errCode := "-1"
	errMess := "unknown error"

	if errorCode, ok := (*response)["errorCode"]; ok {
		switch rc := errorCode.(type) {
		case float64:
			errCode = strconv.FormatFloat(rc, 'f', -1, 32)
		case int:
			errCode = strconv.Itoa(rc)
		case string:
			errCode = rc
		default:
			bpc.Logger(c).Printf("rc type: %v (%T)", rc, rc)
		}
	}

	if errorMessage, ok := (*response)["errorMessage"]; ok {
		if mess, ok := errorMessage.(string); ok {
			errMess = mess
		}
	}

	return &errCode, &errMess
}

func (bpc *BpcChannel) parseState(
	c interface{},
	response *map[string]interface{},
) (string, *string, *string, *string, *string, *string) {
	var actionCode *string
	var actionCodeDescription *string
	var rrn *string
	var authcode *string
	var bindingId *string

	state := "-1"

	if orderStatus, ok := (*response)["orderStatus"]; ok {
		switch status := orderStatus.(type) {
		case float64:
			state = strconv.FormatFloat(status, 'f', -1, 32)
		case int:
			state = strconv.Itoa(status)
		case string:
			state = status
		default:
			bpc.Logger(c).Warningf("orderStatus type: %v (%T)", status, status)
		}
	}

	if ac, ok := (*response)["actionCode"]; ok {
		switch rc := ac.(type) {
		case float64:
			errCode := strconv.FormatFloat(rc, 'f', -1, 32)
			actionCode = &errCode
		case int:
			errCode := strconv.Itoa(rc)
			actionCode = &errCode
		case string:
			actionCode = &rc
		default:
			bpc.Logger(c).Warningf("actionCode type: %v (%T)", rc, rc)
		}
	}

	if actionDescr, ok := (*response)["actionCodeDescription"]; ok {
		switch mess := actionDescr.(type) {
		case string:
			actionCodeDescription = &mess
		default:
			bpc.Logger(c).Warningf("actionCodeDescription type: %v (%T)", mess, mess)
		}
	}

	if authRefNum, ok := (*response)["authRefNum"]; ok {
		switch refNum := authRefNum.(type) {
		case float64:
			ref := strconv.FormatFloat(refNum, 'f', -1, 32)
			rrn = &ref
		case int:
			ref := strconv.Itoa(refNum)
			rrn = &ref
		case string:
			rrn = &refNum
		default:
			bpc.Logger(c).Warningf("authRefNum type: %v (%T)", refNum, refNum)
		}
	}

	if cardAuthInfo, ok := (*response)["cardAuthInfo"]; ok {
		if authInfo, ok := cardAuthInfo.(map[string]interface{}); ok {
			if approvalCode, ok := authInfo["approvalCode"]; ok {
				switch approval := approvalCode.(type) {
				case float64:
					authCode := strconv.FormatFloat(approval, 'f', -1, 32)
					authcode = &authCode
				case int:
					authCode := strconv.Itoa(approval)
					authcode = &authCode
				case string:
					authcode = &approval
				default:
					bpc.Logger(c).Warningf("approvalCode type: %v (%T)", approval, approval)
				}
			}
		}
	}

	if bindingInfo, ok := (*response)["bindingInfo"]; ok {
		if binding, ok := bindingInfo.(map[string]interface{}); ok {
			if linkId, ok := binding["bindingId"]; ok {
				switch bid := linkId.(type) {
				case float64:
					link := strconv.FormatFloat(bid, 'f', -1, 32)
					bindingId = &link
				case int:
					link := strconv.Itoa(bid)
					bindingId = &link
				case string:
					bindingId = &bid
				default:
					bpc.Logger(c).Warningf("bindingId type: %v (%T)", bid, bid)
				}
			}
		}
	}

	return state, actionCode, actionCodeDescription, rrn, authcode, bindingId
}

func (bpc *BpcChannel) Update(c interface{}, transaction *ecom.Transaction, force bool) {
	if !transaction.InFinalState() || force {
		data := url.Values{}
		data.Set("userName", bpc.Settings.Login)
		data.Set("password", bpc.Settings.Password)
		data.Set("orderId", transaction.SafeRemoteId())
		if err, jsonResp := bpc.makeRequest(c, "POST", "rest/getOrderStatusExtended.do", data.Encode()); err == nil {
			state, actionCode, actionCodeDescr, rrn, authCode, bindingId := bpc.parseState(c, jsonResp)

			transaction.ResponseCode = actionCode
			transaction.AuthCode = authCode
			transaction.RRN = rrn
			if bindingId != nil {
				transaction.AdditionalData = &ecom.AdditionalData{
					"bindingId": *bindingId,
				}
			}

			switch state {
			case "0":
				if *actionCode == "0" {
					transaction.Success()
				} else {
					transaction.Declined(actionCodeDescr)
				}
			case
				"1",
				"2",
				"3",
				"4":
				transaction.Success()
			case "6":
				transaction.Declined(actionCodeDescr)
			}
		} else {
			bpc.Logger(c).Warningf("can not make get order status request: %v", err)
		}
	} else {
		bpc.Logger(c).Warningf("transaction <%d> has already got final state: %s", *transaction.Id, *transaction.Status)
	}
}

func (bpc *BpcChannel) processCard(
	c interface{},
	transaction *ecom.Transaction,
	card bankcard.Card,
	termUrl string,
	registerMethod string,
) error {
	data := url.Values{}
	data.Set("userName", bpc.Settings.Login)
	data.Set("password", bpc.Settings.Password)
	data.Set("orderNumber", strconv.Itoa(*transaction.Id))
	data.Set("currency", strconv.Itoa(*transaction.CurrencyConverted.NumericCode))
	data.Set("amount", strconv.Itoa(int(transaction.TotalAmountConverted())))
	data.Set("clientId", *transaction.Customer)
	// @note: we do not use return url at all
	data.Set("returnUrl", "1")

	err, jsonResp := bpc.makeRequest(c, "POST", fmt.Sprintf("rest/%s", registerMethod), data.Encode())
	if err != nil {
		return fmt.Errorf("can not make register order request: %v", err)
	}

	if orderId, ok := (*jsonResp)["orderId"]; ok {
		if remoteId, ok := orderId.(string); ok {
			transaction.RemoteId = &remoteId

			data := url.Values{}
			data.Set("userName", bpc.Settings.Login)
			data.Set("password", bpc.Settings.Password)
			data.Set("MDORDER", remoteId)
			data.Set("$PAN", card.Pan)
			data.Set("$CVC", card.Cvv)
			data.Set("YYYY", strconv.Itoa(card.ExpDate.AsTime().Year()))
			data.Set("MM", fmt.Sprintf("%02d", int(card.ExpDate.AsTime().Month())))
			data.Set("TEXT", card.Holder)
			data.Set("threeDSVer2FinishUrl", termUrl)

			if err, jsonResp := bpc.makeRequest(c, "POST", "rest/paymentorder.do", data.Encode()); err == nil {
				if is3ds20, transId, serverUrl, methodUrl, methodData := bpc.is3DS20(c, jsonResp); is3ds20 {
					//3ds20
					if err := bpc.putBrowserInfo(c, transaction.BrowserInfo, serverUrl, transId); err != nil {
						bpc.Logger(c).Warningf("can not put browser info: %v", err)
					}
					data.Set("threeDSServerTransId", *transId)
					if err := bpc.SessionStore.Add(c, repository.NewSession(
						fmt.Sprintf("3ds20session_%d", *transaction.Id),
						repository.SessionData{
							"query":                data.Encode(),
							"threeDSServerTransId": *transId,
						},
					)); err != nil {
						return fmt.Errorf("can not add 3ds20 session: %v", err)
					}
					if methodUrl != nil {
						transaction.ThreeDSMethodUrl = &ecom.ThreeDSMethodUrl{
							MethodUrl:         methodUrl,
							ThreeDSMethodData: methodData,
						}
						transaction.WaitMethodUrl()
					} else {
						if err, jsonResp := bpc.makeRequest(c, "POST", "rest/paymentorder.do", data.Encode()); err == nil {
							if iscreq, acs, creq := bpc.isCREQ(c, jsonResp); iscreq {
								transaction.ThreeDSecure20 = &ecom.ThreeDSecure20{
									AcsUrl: acs,
									Creq:   creq,
								}
								transaction.Wait3DS()
							} else {
								bpc.Update(c, transaction, false)
							}
						} else {
							bpc.Update(c, transaction, false)
						}
					}
				} else {
					if is3ds10, acs, pareq := bpc.is3DS10(c, jsonResp); is3ds10 {
						transaction.ThreeDSecure10 = &ecom.ThreeDSecure10{
							AcsUrl: acs,
							PaReq:  pareq,
						}
						transaction.Wait3DS()
					} else {
						bpc.Update(c, transaction, false)
					}
				}
			} else {
				bpc.Logger(c).Warningf("can not make payment order request: %v", err)
				bpc.Update(c, transaction, false)
			}
		} else {
			return errors.New("orderId has wrong type")
		}
	} else {
		rc, mess := bpc.parseError(c, jsonResp)

		transaction.ResponseCode = rc
		return errors.New(*mess)
	}

	return nil
}

func (bpc *BpcChannel) Authorize(c interface{}, transaction *ecom.Transaction, request interface{}) error {
	req, ok := request.(validators.CardAuthorizeRequest)
	if !ok {
		return fmt.Errorf("request has wrong type")
	}

	return bpc.processCard(c, transaction, req.Card, req.ThreeDSVer2TermUrl, "register.do")
}

func (bpc *BpcChannel) PreAuthorize(c interface{}, transaction *ecom.Transaction, request interface{}) error {
	req, ok := request.(validators.CardPreAuthorizeRequest)
	if !ok {
		return fmt.Errorf("request has wrong type")
	}

	return bpc.processCard(c, transaction, req.Card, req.ThreeDSVer2TermUrl, "registerPreAuth.do")
}

func (bpc *BpcChannel) Confirm(c interface{}, transaction *ecom.Transaction) error {
	err, successRefsTotal := bpc.TransactionStore.TypeTurnOver(c, ecom.NewTransactionSpecificationByReferenceIdAndStatus(
		*transaction.Reference.Id,
		repository.SUCCESS,
	))
	bpc.Logger(c).Printf("successRefsTotal: %v (%v)", successRefsTotal, err)

	if err != nil {
		return fmt.Errorf("can not get success references total: %v", err)
	}

	var reversals uint
	var confirms uint

	if confirmTurnOver, ok := (*successRefsTotal)[repository.CONFIRMAUTH]; ok {
		confirms = confirmTurnOver.Sum
	}

	if confirms > 0 {
		// @note: alfabank allow only one partial confirm
		return fmt.Errorf("transaction has already confirmed: %d", confirms)
	}

	if reversalTurnOver, ok := (*successRefsTotal)[repository.REVERSAL]; ok {
		reversals = reversalTurnOver.Sum
	}

	availableAmount := *transaction.Reference.Amount - reversals - confirms
	if availableAmount < *transaction.Amount {
		return fmt.Errorf("incorrect amount: %d, availabe to confirm: %d", *transaction.Amount, availableAmount)
	}

	data := url.Values{}
	data.Set("userName", bpc.Settings.Login)
	data.Set("password", bpc.Settings.Password)
	data.Set("orderId", *transaction.Reference.RemoteId)
	data.Set("amount", strconv.Itoa(int(transaction.TotalAmountConverted())))
	transaction.RemoteId = transaction.Reference.RemoteId

	err, jsonResp := bpc.makeRequest(c, "POST", "rest/deposit.do", data.Encode())
	if err != nil {
		return fmt.Errorf("can not make deposit request: %v", err)
	}

	rc, mess := bpc.parseError(c, jsonResp)
	if *rc != "0" {
		transaction.ResponseCode = rc
		return errors.New(*mess)
	}

	bpc.Update(c, transaction, false)

	return nil
}

func (bpc *BpcChannel) Reverse(c interface{}, transaction *ecom.Transaction) error {
	if *transaction.Amount != *transaction.Reference.Amount {
		// @note: alfabank not allowed partial reversal
		return fmt.Errorf("partial reverse not allowed: %d", *transaction.Amount)
	}

	err, successRefsTotal := bpc.TransactionStore.TypeTurnOver(c, ecom.NewTransactionSpecificationByReferenceIdAndStatus(
		*transaction.Reference.Id,
		repository.SUCCESS,
	))
	bpc.Logger(c).Printf("successRefsTotal: %v (%v)", successRefsTotal, err)

	if err != nil {
		return fmt.Errorf("can not get success references total: %v", err)
	}

	var reversals uint

	if reversalTurnOver, ok := (*successRefsTotal)[repository.REVERSAL]; ok {
		reversals = reversalTurnOver.Sum
	}

	if reversals > 0 {
		// @note: alfabank allow only one reversal
		return fmt.Errorf("transaction has already reversed: %d", reversals)
	}

	data := url.Values{}
	data.Set("userName", bpc.Settings.Login)
	data.Set("password", bpc.Settings.Password)
	data.Set("orderId", *transaction.Reference.RemoteId)
	transaction.RemoteId = transaction.Reference.RemoteId

	err, jsonResp := bpc.makeRequest(c, "POST", "rest/reverse.do", data.Encode())
	if err != nil {
		return fmt.Errorf("can not make reverse request: %v", err)
	}

	rc, mess := bpc.parseError(c, jsonResp)
	if *rc != "0" {
		transaction.ResponseCode = rc
		return errors.New(*mess)
	}

	bpc.Update(c, transaction, false)

	return nil
}

func (bpc *BpcChannel) Refund(c interface{}, transaction *ecom.Transaction) error {
	err, successRefsTotal := bpc.TransactionStore.TypeTurnOver(c, ecom.NewTransactionSpecificationByReferenceIdAndStatus(
		*transaction.Reference.Id,
		repository.SUCCESS,
	))
	bpc.Logger(c).Printf("successRefsTotal: %v (%v)", successRefsTotal, err)

	if err != nil {
		return fmt.Errorf("can not get success references total: %v", err)
	}

	var refunds uint

	if refundTurnOver, ok := (*successRefsTotal)[repository.REFUND]; ok {
		refunds = refundTurnOver.Sum
	}

	availableAmount := transaction.Reference.TotalAmount() - refunds
	if availableAmount < transaction.TotalAmount() {
		return fmt.Errorf("incorrect amount: %d, availabe to refund: %d", transaction.TotalAmount(), availableAmount)
	}

	data := url.Values{}
	data.Set("userName", bpc.Settings.Login)
	data.Set("password", bpc.Settings.Password)
	data.Set("orderId", *transaction.Reference.RemoteId)
	data.Set("amount", strconv.Itoa(int(transaction.TotalAmountConverted())))
	transaction.RemoteId = transaction.Reference.RemoteId

	err, jsonResp := bpc.makeRequest(c, "POST", "rest/refund.do", data.Encode())
	if err != nil {
		return fmt.Errorf("can not make refund request: %v", err)
	}

	rc, mess := bpc.parseError(c, jsonResp)
	if *rc != "0" {
		transaction.ResponseCode = rc
		return errors.New(*mess)
	}

	bpc.Update(c, transaction, false)

	return nil
}

func (bpc *BpcChannel) Rebill(c interface{}, transaction *ecom.Transaction) error {
	data := url.Values{}
	data.Set("userName", bpc.Settings.Login)
	data.Set("password", bpc.Settings.Password)
	data.Set("orderNumber", strconv.Itoa(*transaction.Id))
	data.Set("currency", strconv.Itoa(*transaction.CurrencyConverted.NumericCode))
	data.Set("features", "AUTO_PAYMENT")
	data.Set("amount", strconv.Itoa(int(transaction.TotalAmountConverted())))
	data.Set("clientId", *transaction.Customer)
	// @note: we do not use return url at all
	data.Set("returnUrl", "1")

	err, jsonResp := bpc.makeRequest(c, "POST", "rest/register.do", data.Encode())
	if err != nil {
		return fmt.Errorf("can not make register order request: %v", err)
	}

	if orderId, ok := (*jsonResp)["orderId"]; ok {
		if remoteId, ok := orderId.(string); ok {
			transaction.RemoteId = &remoteId

			additionalData := transaction.Reference.AdditionalData
			if additionalData == nil {
				return errors.New("reference has not additional data")
			}

			if bindingId, ok := (*additionalData)["bindingId"]; ok {
				if bid, ok := bindingId.(string); ok {
					data := url.Values{}
					data.Set("userName", bpc.Settings.Login)
					data.Set("password", bpc.Settings.Password)
					data.Set("mdOrder", remoteId)
					data.Set("ip", transaction.Reference.BrowserInfo.IP)
					data.Set("bindingId", bid)
					bpc.makeRequest(c, "POST", "rest/paymentOrderBinding.do", data.Encode())
					bpc.Update(c, transaction, false)
				} else {
					return errors.New("bindingId has wrong type")
				}
			} else {
				return errors.New("additional data has not bindingId")
			}

		} else {
			return errors.New("orderId has wrong type")
		}
	} else {
		rc, mess := bpc.parseError(c, jsonResp)

		transaction.ResponseCode = rc
		return errors.New(*mess)
	}

	return nil
}

func (bpc *BpcChannel) ProcessPares(c interface{}, transaction *ecom.Transaction, pares string) error {
	bpc.Logger(c).Printf("Pares: %v", pares)

	data := url.Values{}
	data.Set("PaRes", pares)
	data.Set("MD", *transaction.RemoteId)

	bpc.makeRequest(c, "POST", "rest/finish3ds.do", data.Encode())
	bpc.Update(c, transaction, false)

	return nil
}

func (bpc *BpcChannel) ProcessCres(c interface{}, transaction *ecom.Transaction, cres string) error {
	sessionKey := fmt.Sprintf("3ds20session_%d", *transaction.Id)
	err, _, sessions := bpc.SessionStore.Query(c, repository.NewSessionSpecificationByKey(sessionKey))

	if err != nil {
		return fmt.Errorf("failed to query session store: %v", err)
	}

	if len(sessions) == 0 {
		return fmt.Errorf("session with key %s not found", sessionKey)
	}

	session := sessions[0]
	sessionData := session.Data

	threeDSServerTransId, ok := (*sessionData)["threeDSServerTransId"]
	if !ok {
		return errors.New("session data has not threeDSServerTransId")
	}

	tDsTransId, ok := threeDSServerTransId.(string)
	if !ok {
		return errors.New("session data threeDSServerTransId has wrong type")
	}

	data := url.Values{}
	data.Set("userName", bpc.Settings.Login)
	data.Set("password", bpc.Settings.Password)
	data.Set("tDsTransId", tDsTransId)

	bpc.makeRequest(c, "POST", "rest/finish3dsVer2.do", data.Encode())
	bpc.Update(c, transaction, false)

	return nil
}

func (bpc *BpcChannel) CompleteMethodUrl(c interface{}, transaction *ecom.Transaction, completed bool) error {
	bpc.Logger(c).Printf("completed: %v", completed)

	sessionKey := fmt.Sprintf("3ds20session_%d", *transaction.Id)
	err, _, sessions := bpc.SessionStore.Query(c, repository.NewSessionSpecificationByKey(sessionKey))

	if err != nil {
		return fmt.Errorf("failed to query session store: %v", err)
	}

	if len(sessions) == 0 {
		return fmt.Errorf("session with key %s not found", sessionKey)
	}

	session := sessions[0]
	sessionData := session.Data

	query, ok := (*sessionData)["query"]
	if !ok {
		return errors.New("session data has not query")
	}

	qwr, ok := query.(string)
	if !ok {
		return errors.New("session data query has wrong type")
	}

	if err, jsonResp := bpc.makeRequest(c, "POST", "rest/paymentorder.do", qwr); err == nil {
		if iscreq, acs, creq := bpc.isCREQ(c, jsonResp); iscreq {
			transaction.ThreeDSecure20 = &ecom.ThreeDSecure20{
				AcsUrl: acs,
				Creq:   creq,
			}
			transaction.Wait3DS()
		} else {
			bpc.Update(c, transaction, false)
		}
	} else {
		bpc.Update(c, transaction, false)
	}

	return nil
}
