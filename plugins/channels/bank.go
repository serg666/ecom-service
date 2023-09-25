package channels

import (
	ecom "github.com/serg666/service-repository/ecom"
)

type BankChannel interface {
	Authorize(c interface{}, transaction *ecom.Transaction, request interface{}) error
	PreAuthorize(c interface{}, transaction *ecom.Transaction, request interface{}) error
	Confirm(c interface{}, transaction *ecom.Transaction) error
	Reverse(c interface{}, transaction *ecom.Transaction) error
	Refund(c interface{}, transaction *ecom.Transaction) error
	Rebill(c interface{}, transaction *ecom.Transaction) error
	ProcessCres(c interface{}, transaction *ecom.Transaction, cres string) error
	ProcessPares(c interface{}, transaction *ecom.Transaction, pares string) error
	CompleteMethodUrl(c interface{}, transaction *ecom.Transaction, completed bool) error
	Update(c interface{}, transaction *ecom.Transaction, force bool)
}

var BankChannelType int = 1
