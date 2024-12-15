package xcashu

import (
	"errors"
	"fmt"
	"github.com/elnosh/gonuts/cashu"
)

const Xcashu = "x-cashu"

const BPerMB = 1024 * 1024

type Unit string

const Sat Unit = "sat"
const Msat Unit = "milisat"

type PaymentQuoteResponse struct {
	Amount uint64   `json:"amount"`
	Unit   Unit     `json:"unit"`
	Mints  []string `json:"mints"`
	Pubkey string   `json:"pubkey"`
}

var (
	ErrNotEnoughtSats = errors.New("Not enough sats")
)

// charges per
func QuoteAmountToPay(Blength uint64, satPerMB uint64) uint64 {
	if Blength < 1024 {
		return 1
	}

	lengthInMb := Blength / BPerMB

	res := lengthInMb / satPerMB

	if res == 0 {

		return 1
	}
	return res
}

func ParseTokenHeader(tokenHeader string, amountToPay uint64) (cashu.Token, error) {
	token, err := cashu.DecodeToken(tokenHeader)

	if err != nil {
		return token, fmt.Errorf("cashu.DecodeToken(tokenHeader) %w", err)
	}

	if token.Amount() < amountToPay {
		return token, ErrNotEnoughtSats
	}

	return token, nil

}
