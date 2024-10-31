package xcashu

import (
	"errors"
	"fmt"
	"slices"

	"github.com/elnosh/gonuts/cashu"
	w "github.com/elnosh/gonuts/wallet"
)

const Xcashu = "x-cashu"
const XContentLength = "x-content-length"
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
	ErrNotTrustedMint = errors.New("Not from trusted Mint")
)

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

func VerifyTokenIsValid(tokenHeader string, amountToPay uint64, wallet *w.Wallet) error {
	token, err := cashu.DecodeToken(tokenHeader)

	if err != nil {
		return fmt.Errorf("cashu.DecodeToken(tokenHeader) %w", err)
	}

	if token.Amount() < amountToPay {
		return ErrNotEnoughtSats
	}

	if !slices.Contains(wallet.TrustedMints(), token.Mint()) {
		return ErrNotTrustedMint
	}

	fmt.Printf("\n Token: %+v", token)
	// TODO - Check if it is locked to the pubkey of the wallet

	_, err = wallet.Receive(token, false)
	if err != nil {
		return fmt.Errorf("wallet.Receive(token, false) %w", err)
	}
	return nil

}
