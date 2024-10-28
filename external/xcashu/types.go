package xcashu

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
