package core

import (
	"encoding/hex"
	"fmt"
	"log"
	"ratasker/internal/cashu"
	"ratasker/internal/database"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	c "github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut12"
	"github.com/elnosh/gonuts/crypto"
)

func StringToPubkey(pubkey string) (*secp256k1.PublicKey, error) {
	var pubkeyFromMint *secp256k1.PublicKey
	pubkeyFromMintByte, err := hex.DecodeString(pubkey)
	if err != nil {
		return pubkeyFromMint, fmt.Errorf("hex.DecodeString(pubkey). %w", err)
	}

	pubkeyFromMint, err = secp256k1.ParsePubKey(pubkeyFromMintByte)
	if err != nil {
		return pubkeyFromMint, fmt.Errorf("secp256k1.ParsePubKey(pubkeyFromMintByte). %w", err)
	}
	return pubkeyFromMint, nil

}

func WatchForPubkeyRotation(wallet cashu.CashuWallet, db database.Database) {

}

func RotateLockedProofs(wallet cashu.CashuWallet, db database.Database) error {

	tx, err := db.BeginTransaction()
	if err != nil {
		return fmt.Errorf("db.BeginTransaction(). %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			log.Fatalf("Panic occurred: %v\n", p)
		} else if err != nil {
			log.Println("Rolling back transaction due to error.")
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Fatalf("Failed to commit transaction: %v\n", err)
			}
		}
	}()

	proofsPerMint, err := db.GetLockedProofsByRedeemed(tx, false)
	if err != nil {
		return fmt.Errorf("db.GetProofsByRedeemed(tx, false ). %w", err)
	}

	for mint_url, proofsToSwap := range proofsPerMint {

		keyset, err := wallet.GetActiveKeyset(mint_url)
		if err != nil {
			return fmt.Errorf("wallet.GetActiveKeyset(mint_url). %w", err)
		}
		counter, err := db.GetKeysetCounter(tx, keyset.Id)
		if err != nil {
			return fmt.Errorf("db.GetKeysetCounter(tx,keyset.Id ). %w", err)
		}

		blindMessages, secrets, keys, err := wallet.MakeBlindMessages(proofsToSwap.Amount(), mint_url, &counter)
		if err != nil {
			return fmt.Errorf("wallet.MakeBlindMessages(proofs, mint_url). %w", err)
		}

		fmt.Printf("\n counter %+v \n", counter)

		blindSigs, err := wallet.SwapProofs(blindMessages, proofsToSwap, mint_url)
		if err != nil {
			return fmt.Errorf("wallet.SwapProofs(blindMessages, proofs, mint_url). %w", err)
		}

		err = db.SetKeysetCounter(tx, counter)
		if err != nil {
			return fmt.Errorf("db.SetKeysetCounter(tx, counter). %w", err)
		}

		var NewProofs c.Proofs

		for i, blindSig := range blindSigs {

			C_, err := StringToPubkey(blindSig.C_)
			if err != nil {
				return fmt.Errorf("StringToPubkey(blindSig.C_). %w", err)
			}

			mintPubkey, err := StringToPubkey(keyset.Keys[blindSig.Amount])
			if err != nil {
				return fmt.Errorf("StringToPubkey(blindSig.C_). %w", err)
			}

			C := crypto.UnblindSignature(C_, keys[i], mintPubkey)

			if blindSig.DLEQ != nil {
				dleqRes := nut12.VerifyBlindSignatureDLEQ(*blindSig.DLEQ, mintPubkey, blindMessages[i].B_, blindSig.C_)
				if !dleqRes {
					log.Printf("\n ERROR: DLEQ has not passed. %+v", blindSig)
				}
			}

			proof := c.Proof{
				Amount: blindSig.Amount,
				Id:     blindSig.Id,
				Secret: secrets[i],
				DLEQ:   blindSig.DLEQ,
				C:      hex.EncodeToString(C.SerializeCompressed()),
			}

			NewProofs = append(NewProofs, proof)
		}

		// crypto.Un

		// wallet.S
		// bytes, err := hex.DecodeString(proofs[i].C)
		// if err != nil {
		// 	return wallet, fmt.Errorf("db.GetProofsByRedeemed(tx, false) %w", err)
		// }
		// wallet.filter.Add(bytes)

	}
	return nil
}
