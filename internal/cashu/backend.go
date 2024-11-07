package cashu

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"ratasker/internal/database"
	"slices"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut01"
	"github.com/elnosh/gonuts/cashu/nuts/nut10"
	"github.com/elnosh/gonuts/cashu/nuts/nut11"
	"github.com/elnosh/gonuts/cashu/nuts/nut12"
	"github.com/elnosh/gonuts/wallet"
	"github.com/tyler-smith/go-bip39"
)

const DerivationForP2PK = 129372
const ExpirationOfPubkeyHours = 4

var (
	ErrNotTrustedMint         = errors.New("Not from trusted Mint")
	ErrKeysetIdNotFound       = errors.New("Keyset Id not found")
	ErrNotLockedToPubkey      = errors.New("Proof not locked to pubkey")
	ErrCouldNotFindMintPubkey = errors.New("Could not find mint pubkey")
	ErrCouldNotVerifyDLEQ     = errors.New("Could not verify proof comes from trusted mint")
	ErrProofIsNotP2PK         = errors.New("Proof is not P2PK")
	ErrProofAlreadySeen       = errors.New("Proof already seen")
)

type CashuWallet interface {
	RotatePubkey(tx *sql.Tx, db database.Database) error
	GetActivePubkey() string
	StoreEcash(proofs cashu.Proofs, tx *sql.Tx, db database.Database) error
	VerifyToken(token cashu.Token, tx *sql.Tx, db database.Database) (cashu.Proofs, error)
}

type DBNativeWallet struct {
	privKey       *hdkeychain.ExtendedKey
	CurrentPubkey *secp256k1.PublicKey
	PubkeyVersion database.CurrentPubkey
	activeKeys    map[string]nut01.Keyset
	filter        *bloom.BloomFilter
}

func NewDBLocalWallet(seedWords string, db database.Database) (DBNativeWallet, error) {
	var wallet DBNativeWallet
	wallet.activeKeys = make(map[string]nut01.Keyset)

	seed, err := bip39.MnemonicToByteArray(seedWords)
	if err != nil {
		return wallet, fmt.Errorf("bip39.MnemonicToByteArray(seedWords). %w", err)
	}

	privekey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return wallet, fmt.Errorf("hdkeychain.NewMaster(. %w", err)
	}

	tx, err := db.BeginTransaction()
	if err != nil {
		return wallet, fmt.Errorf("db.BeginTransaction() %w", err)

	}
	// Ensure that the transaction is rolled back in case of a panic or error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Printf("\n Failed to commit transaction: %v\n", err)
			}
			fmt.Println("Transaction committed successfully.")
		}
	}()

	mints, err := db.GetTrustedMints(tx)
	if err != nil {
		return wallet, fmt.Errorf("db.GetTrustedMints() %w", err)

	}

	// Get all active keys form mints
	err = wallet.getActiveKeysFromTrustedMints(mints)
	if err != nil {
		return wallet, fmt.Errorf("wallet.getActiveKeysFromTrustedMints() %w", err)
	}

	// Set bloom filter
	wallet.filter = bloom.NewWithEstimates(1_000_000, 0.01)

	// Get proofs that are not redeemed
	proofs, err := db.GetProofsByRedeemed(tx, false)
	if err != nil {
		return wallet, fmt.Errorf("db.GetProofsByRedeemed(tx, false) %w", err)
	}

	for i := 0; i < len(proofs); i++ {
		bytes, err := hex.DecodeString(proofs[i].C)
		if err != nil {
			return wallet, fmt.Errorf("db.GetProofsByRedeemed(tx, false) %w", err)
		}
		wallet.filter.Add(bytes)
	}

	wallet.privKey = privekey
	return wallet, nil
}

func (l *DBNativeWallet) getActiveKeysFromTrustedMints(mints []string) error {
	for _, mintUrl := range mints {
		keys, err := wallet.GetActiveKeysets(mintUrl)
		if err != nil {
			return fmt.Errorf("wallet.GetAllKeysets(mintUrl) %w", err)
		}

		for _, keyset := range keys.Keysets {
			if keyset.Unit == "sat" {
				l.activeKeys[mintUrl] = keyset
			}
		}
	}
	return nil
}

func (l *DBNativeWallet) derivePrivateKey(version uint) (*secp256k1.PrivateKey, error) {
	var derivedKey *secp256k1.PrivateKey
	p2pkPurpose, err := l.privKey.Derive(hdkeychain.HardenedKeyStart + DerivationForP2PK)
	if err != nil {
		return derivedKey, fmt.Errorf("l.privKey.Derive(hdkeychain.HardenedKeyStart + DerivationForP2PK). %w", err)
	}

	key, err := p2pkPurpose.Derive(hdkeychain.HardenedKeyStart + uint32(version))

	if err != nil {
		return derivedKey, fmt.Errorf("p2pkPurpose.Derive(hdkeychain.HardenedKeyStart + version). %w", err)
	}
	derivedKey, err = key.ECPrivKey()
	if err != nil {
		return derivedKey, fmt.Errorf("key.ECPubKey() %w", err)
	}
	return derivedKey, nil
}

func (l *DBNativeWallet) RotatePubkey(tx *sql.Tx, db database.Database) error {

	expiration := time.Now().Add(ExpirationOfPubkeyHours * time.Hour)
	version, err := db.RotateNewPubkey(tx, expiration.Unix())
	if err != nil {
		return fmt.Errorf("github.com/elnosh/gonuts.%w", err)
	}

	privKey, err := l.derivePrivateKey(l.PubkeyVersion.VersionNum)

	if err != nil {
		return fmt.Errorf("l.derivePrivateKey(version) %w", err)
	}

	l.CurrentPubkey = privKey.PubKey()
	l.PubkeyVersion = version
	return nil
}

func (l *DBNativeWallet) GetActivePubkey() string {
	return hex.EncodeToString(l.CurrentPubkey.SerializeCompressed())
}

func (l *DBNativeWallet) StoreEcash(proofs cashu.Proofs, tx *sql.Tx, db database.Database) error {
	now := time.Now().Unix()
	err := db.AddProofs(tx, proofs, l.PubkeyVersion.VersionNum, false, uint64(now))
	if err != nil {
		return fmt.Errorf("db.AddProofs(proofs, false, now) %w", err)
	}

	return nil
}

// func (l *DBNativeWallet) redeemEcash(db database.Database) error {
//     proofs, err := db.GetProofsByPubkeyVersion(l.pubkeyVersion)
//     if err != nil {
//         return fmt.Errorf("db.GetProofsByPubkeyVersion(l.pubkeyVersion) %w", err)
//     }
//
//     trustedMints, err := db.GetTrustedMints()
//     if err != nil {
//         return fmt.Errorf("db.GetTrustedMints() %w", err)
//     }
//
//
// 	return nil
//
// }

func checkMapOfPubkeys(keys nut01.KeysMap, amount uint64) (*secp256k1.PublicKey, error) {
	key, ok := keys[amount]

	if !ok {
		return nil, ErrCouldNotFindMintPubkey
	}

	bytes, err := hex.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString(key). %w ", err)
	}
	pubkey, err := secp256k1.ParsePubKey(bytes)
	if err != nil {
		return nil, fmt.Errorf("secp256k1.ParsePubKey(bytes) %w ", err)
	}
	return pubkey, nil

}

func FindKeysetPubkey(tx *sql.Tx, proof cashu.Proof, mintUrl string, activeKeyset nut01.Keyset, tmpKeys map[string]nut01.Keyset) (*secp256k1.PublicKey, error) {
	// See if keyset is available  in activeKeyset
	if activeKeyset.Id == proof.Id {
		pubkey, err := checkMapOfPubkeys(activeKeyset.Keys, proof.Amount)
		if err != nil {
			return nil, fmt.Errorf("checkMapOfPubkeys(activeKeyset.Keys, proof.Amount) %w ", err)
		}
		return pubkey, nil
	}

	// check tmpKeys
	keyset, ok := tmpKeys[proof.Id]
	if ok {
		pubkey, err := checkMapOfPubkeys(keyset.Keys, proof.Amount)
		if err != nil {
			return nil, fmt.Errorf("checkMapOfPubkeys(activeKeyset.Keys, proof.Amount) %w ", err)
		}
		return pubkey, nil
	}

	// Call the mint and ask for the keyset if found store it
	keys, err := wallet.GetKeysetById(mintUrl, proof.Id)
	if err != nil {
		return nil, fmt.Errorf("wallet.GetKeysetById(mintUrl, proof.Id). %w", err)
	}

	if len(keys.Keysets) > 0 {
		tmpKeys[proof.Id] = keys.Keysets[0]
		pubkey, err := checkMapOfPubkeys(tmpKeys[proof.Id].Keys, proof.Amount)
		if err != nil {
			return nil, fmt.Errorf("checkMapOfPubkeys(activeKeyset.Keys, proof.Amount) %w ", err)
		}
		return pubkey, nil

	}

	// if not found error out
	return nil, ErrCouldNotFindMintPubkey
}

func (l *DBNativeWallet) VerifyToken(token cashu.Token, tx *sql.Tx, db database.Database) (cashu.Proofs, error) {

	trustedMints, err := db.GetTrustedMints(tx)
	if err != nil {
		return token.Proofs(), fmt.Errorf("db.GetTrustedMints() %w", err)
	}
	if !slices.Contains(trustedMints, token.Mint()) {
		return token.Proofs(), ErrNotTrustedMint
	}

	// Get Keysets form mints
	// can make a good guess that the tokens with the currect active keyset because of the lock to p2pk
	if err != nil {
		return token.Proofs(), fmt.Errorf("wallet.GetAllKeysets(token.Mint()) %w", err)
	}

	lockedEcashPrivateKey, err := l.derivePrivateKey(l.PubkeyVersion.VersionNum)

	if err != nil {
		return token.Proofs(), fmt.Errorf("l.derivePrivateKey(version) %w", err)
	}
	now := time.Now()

	mintKeys, ok := l.activeKeys[token.Mint()]

	if !ok {
		return token.Proofs(), ErrNotTrustedMint
	}

	tmpKeys := make(map[string]nut01.Keyset)

	for _, p := range token.Proofs() {
		mintPubkey, err := FindKeysetPubkey(tx, p, token.Mint(), mintKeys, tmpKeys)

		spendCondition, err := nut10.DeserializeSecret(p.Secret)
		if err != nil {
			return token.Proofs(), fmt.Errorf("nut10.DeserializeSecret(p.Secret) %w. %w", err, ErrNotLockedToPubkey)
		}

		// Verify that it is a P2PK
		if spendCondition.Kind != nut10.P2PK {
			return token.Proofs(), ErrProofIsNotP2PK
		}

		// Verify that is lock to a private key that I control
		if !nut11.CanSign(spendCondition, lockedEcashPrivateKey) {
			return token.Proofs(), fmt.Errorf("nut10.DeserializeSecret(p.Secret) %w. %w", err, ErrNotLockedToPubkey)
		}
		// Verificar que tiene un bloqueo de al menos 4 horas
		p2pkTags, err := nut11.ParseP2PKTags(spendCondition.Data.Tags)
		if err != nil {
			return token.Proofs(), fmt.Errorf("nut11.ParseP2PKTags(spendCondition.Data.Tags) %w.", err)
		}

		locktime := time.Unix(p2pkTags.Locktime, 0)
		now = now.Add(ExpirationOfPubkeyHours * time.Hour)

		if locktime.Unix() < now.Unix() {
			return token.Proofs(), fmt.Errorf("Timestamp doesn't have a locktime of 4 hours")
		}

		// Verificar que esta unblinded correctamente
		if !nut12.VerifyProofDLEQ(p, mintPubkey) {
			return token.Proofs(), fmt.Errorf("nut12.VerifyProofDLEQ(p, mintPubkey). %w. %w", err, ErrCouldNotVerifyDLEQ)
		}

		// TODO - LATER Check Bloom filter if there is a collision and if there is check if it's a secret
		bytesC, err := hex.DecodeString(p.C)
		if err != nil {
			return token.Proofs(), fmt.Errorf("hex.DecodeString(p.C) %w.", err)
		}

		// if conflic check if C already Exists
		if l.filter.TestOrAdd(bytesC) {
			proofs, err := db.GetProofsByC(tx, []string{p.C})
			if len(proofs) > 0 {
				return token.Proofs(), ErrProofAlreadySeen
			}
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}

				return token.Proofs(), fmt.Errorf("db.GetProofsByC(tx, []string{p.C}) %w.", err)
			}
		}

	}
	return token.Proofs(), nil
}
