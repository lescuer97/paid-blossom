package cashu

import (
	"encoding/hex"
	"fmt"
	"ratasker/internal/database"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/elnosh/gonuts/cashu"
	"github.com/tyler-smith/go-bip39"
)

type CashuWallet interface {
	RotatePubkey() error
	GetActivePubkey() string
	StoreEcash(proofs cashu.Proofs, db database.Database) error
	redeemEcash(db database.Database) error
	EcashIsLockedToWallet(proofs cashu.Proofs) (bool, error)
}

type DBNativeWallet struct {
	privKey       *hdkeychain.ExtendedKey
	CurrentPubkey secp256k1.PublicKey
}

func NewDBLocalWallet(seedWords string, db database.Database) (DBNativeWallet, error) {
	var wallet DBNativeWallet

	seed, err := bip39.MnemonicToByteArray(seedWords)
	if err != nil {
		return wallet, fmt.Errorf("bip39.MnemonicToByteArray(seedWords). %w", err)
	}

	privekey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return wallet, fmt.Errorf("hdkeychain.NewMaster(. %w", err)
	}

	wallet.privKey = privekey

	return wallet, nil
}

func (l *DBNativeWallet) RotatePubkey(db database.Database) error {
	// l.Wallet.GetReceivePubkey()
	return nil
}

func (l *DBNativeWallet) GetActivePubkey() string {
	return hex.EncodeToString(l.CurrentPubkey.SerializeCompressed())
}

func (l *DBNativeWallet) StoreEcash(proofs cashu.Proofs, db database.Database) error {
	return nil
}

func (l *DBNativeWallet) redeemEcash(db database.Database) error {
	// l.CurrentPubkey
	// l.R
	return nil

}

func (l *DBNativeWallet) EcashIsLockedToWallet(proofs cashu.Proofs) (bool, error) {
	// l.CurrentPubkey
	// l.R
	return false, nil

}
