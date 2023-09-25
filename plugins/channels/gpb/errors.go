package gpb

import "errors"

var (
	errUnimplemented     = errors.New("Method not implemented yet")
	errRequestWrongType  = errors.New("GPB. Request has wrong type")
	errCreateToken       = errors.New("GPB. Create token failed")
	errInitPayment       = errors.New("GPB. Init payment failed")
	err3ds2PrepareAccept = errors.New("GPB. 3DS2 prepare accept failed")
	errMerchantReturn    = errors.New("GPB. Merchant return failed")
)
