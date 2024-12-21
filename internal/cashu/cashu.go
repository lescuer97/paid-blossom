package cashu

import (
	"errors"
	"os"
)

const TRUSTED_MINT = "TRUSTED_MINT"

var (
	ErrNoTrustedMint = errors.New("No trusted mint")
)

func GetTrustedMintFromOsEnv() (string, error) {
	trustedMint := os.Getenv(TRUSTED_MINT)

	if trustedMint == "" {
		return "", ErrNoTrustedMint
	}

	return trustedMint, nil
}
