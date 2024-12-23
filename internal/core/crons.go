package core

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"ratasker/internal/cashu"
	"ratasker/internal/database"
	"ratasker/internal/utils"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	c "github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut12"
	"github.com/elnosh/gonuts/crypto"
	w "github.com/elnosh/gonuts/wallet"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip44"
)

const discoveryRelay = "wss://purplepag.es"

var (
	ErrNoRelayMetadataForMessaging = errors.New("No relay metadata for messaging")
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

func GetRelaysFromNIP65Pubkey(pubkey string, relayUrl string, pool *nostr.SimplePool) error {

	relay, err := nostr.RelayConnect(context.Background(), relayUrl)
	if err != nil {
		return fmt.Errorf("nostr.RelayConnect(context.Background(),discoveryRelay ). %w", err)
	}
	filter := nostr.Filter{
		Authors: []string{pubkey},
		Kinds:   []int{10002},
	}
	events, err := relay.QuerySync(context.Background(), filter)
	if err != nil {
		return fmt.Errorf("relay.QuerySync(context.Background(), filter). %w", err)
	}

	if len(events) == 0 {
		return ErrNoRelayMetadataForMessaging
	}

	for _, v := range events {
		for _, tag := range v.Tags {

			relay, err := nostr.RelayConnect(context.Background(), tag.Value())
			if err != nil {
				continue
			}
			pool.Relays.Store(tag.Value(), relay)
		}

	}
	return nil
}

func SendEncryptedProofsToPubkey(privKey string, encryptedToken string, pubkey string, pool *nostr.SimplePool) error {
	tag := nostr.Tag{"r", pubkey}
	// make event
	ev := nostr.Event{
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindEncryptedDirectMessage,
		Tags:      nostr.Tags{tag},
		Content:   encryptedToken,
	}

	err := ev.Sign(privKey)
	if err != nil {
		return fmt.Errorf("ev.Sign(privKey). %w", err)
	}
	// send event to relays
	pool.Relays.Range(func(key string, value *nostr.Relay) bool {
		if err := value.Publish(context.Background(), ev); err != nil {
			return true
		}

		return true
	})

	return nil
}

func GetUnspentProofsToTokens(wallet cashu.CashuWallet, db database.Database, tx *sql.Tx) ([]c.TokenV4, error) {
	var tokens []c.TokenV4
	mintsProofs, err := db.GetBySpentProofs(tx, false)
	if err != nil {
		return tokens, fmt.Errorf("db.GetBySpentProofs(tx, false ). %w", err)
	}

	for key, val := range mintsProofs {
		token, err := c.NewTokenV4(val, key, c.Sat, false)
		if err != nil {
			return tokens, fmt.Errorf("c.NewTokenV4(val, key, c.Sat, true). %w", err)
		}
		tokens = append(tokens, token)

	}

	return tokens, nil
}

func SpendSwappedProofs(wallet cashu.CashuWallet, db database.Database) error {

	// rotate keys up
	tx, err := db.BeginTransaction()
	if err != nil {
		log.Panicf("Could not get a lock on the db. %+v", err)
	}
	// Ensure that the transaction is rolled back in case of a panic or error
	defer func() {
		if p := recover(); p != nil {
			log.Printf("\n Rolling back  because of failure %+v\n", p)
			tx.Rollback()
		} else if err != nil {
			log.Println("Rolling back  because of error")
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Printf("\n Failed to write ecash token to file: %v\n", err)
			}
			fmt.Println("Ecash token written to file successfully")
		}
	}()

	tokens, err := GetUnspentProofsToTokens(wallet, db, tx)
	if err != nil {
		log.Printf("\n GetUnspentProofsToTokens(wallet, db, tx) %v\n", err)
	}

	if len(tokens) > 0 {
		err = WriteTokenToLocalFile(tokens)
		if err != nil {
			log.Printf("\n GetUnspentProofsToTokens(wallet, db, tx) %v\n", err)
		}

		var proofs c.Proofs
		for _, v := range tokens {

			proofs = append(proofs, v.Proofs()...)
		}

		err = db.ChangeSwappedProofsSpent(tx, proofs, true)
		if err != nil {
			log.Printf("\n db.ChangeSwappedProofsSpent() %v\n", err)
		}

	}
	return nil
}

const tokenFile = "tokens.txt"

func WriteTokenToLocalFile(tokens []c.TokenV4) error {
	// check if file exists
	homeDir, err := utils.GetRastaskerHomeDirectory()
	if err != nil {
		return fmt.Errorf("utils.GetRastaskerHomeDirectory(). %w", err)
	}

	err = utils.MakeSureFilePathExists(homeDir, tokenFile)
	if err != nil {
		return fmt.Errorf("utils.MakeSureFilePathExists(homeDir, tokenFile). %w", err)
	}

	completeFilePath := homeDir + "/" + tokenFile

	file, err := os.ReadFile(completeFilePath)
	if err != nil {
		return fmt.Errorf("os.ReadFile(completeFilePath). %w", err)
	}

	filestring := string(file)

	now := time.Now().Format(time.UnixDate)

	filestring = filestring + "\n" + now + ": \n"

	for _, token := range tokens {
		tokenString, err := token.Serialize()
		if err != nil {
			return fmt.Errorf("token.Serialize(). %w", err)
		}
		filestring = filestring + "\n" + tokenString + "\n"
	}

	err = os.WriteFile(completeFilePath, []byte(filestring), 0764)
	if err != nil {
		return fmt.Errorf("os.WriteFile(completeFilePath, file) %w", err)
	}

	return nil
}

// take the redeem proofs and send them to a nostr user
func SendProofsToOwner(db database.Database, tx *sql.Tx, tokens []c.Token, pubkey string) error {

	ctx := context.Background()

	// generate key to send proofs
	privKey := nostr.GeneratePrivateKey()
	pool := nostr.NewSimplePool(ctx)

	// get relays of the nostr user
	err := GetRelaysFromNIP65Pubkey(pubkey, discoveryRelay, pool)
	if err != nil {
		return fmt.Errorf("GetRelaysFromNIP65Pubkey(pubkey, pool). %w", err)
	}

	_, err = nip44.GenerateConversationKey(pubkey, privKey)
	if err != nil {
		return fmt.Errorf("nip44.GenerateConversationKey(pubkey, privKey). %w", err)
	}

	for _, token := range tokens {
		tokenString, err := token.Serialize()
		if err != nil {
			return fmt.Errorf("token.Serialize(). %w", err)
		}

		log.Println("tokenString: ", tokenString)
		// encryptedString, err := nip44.Encrypt(tokenString, conversationKey, nil)
		// if err != nil {
		// 	return fmt.Errorf("nip44.Encrypt(tokenString, conversationKey). %w", err)
		// }
		// err = SendEncryptedProofsToPubkey(privKey, encryptedString, pubkey, pool) // send to user
		// if err != nil {
		// 	return fmt.Errorf("SendEncryptedProofsToPubkey(privKey, encryptedString, pubkey,pool). %w", err)
		// }
		//
		err = db.ChangeSwappedProofsSpent(tx, token.Proofs(), true)
		if err != nil {
			return fmt.Errorf("db.ChangeSwappedProofsSpent(tx, val, true). %w", err)
		}
	}

	return nil
}

func RotateLockedProofs(wallet cashu.CashuWallet, db database.Database, tx *sql.Tx) error {

	proofsPerMint, err := db.GetLockedProofsByRedeemed(tx, false)
	if err != nil {
		return fmt.Errorf("db.GetLockedProofsByRedeemed(tx, false). %w", err)
	}

	for mint_url, proofsToSwap := range proofsPerMint {
		keyset, err := wallet.GetActiveKeyset(mint_url)
		if err != nil {
			return fmt.Errorf("wallet.GetActiveKeyset(mint_url). %w", err)
		}

		counter, err := db.GetKeysetCounter(tx, keyset.Id)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				counter.Counter = 0
				counter.KeysetId = keyset.Id
			} else {
				return fmt.Errorf("db.GetKeysetCounter(tx,keyset.Id ). %w", err)
			}
		}

		// TODO query fees of mint and keysets
		keysets, err := w.GetAllKeysets(mint_url)
		if err != nil {
			return fmt.Errorf("w.GetAllKeysets(mint_url). %w", err)

		}

		fees, err := wallet.CalculateFeesFromProofs(proofsToSwap, keysets)
		if err != nil {
			return fmt.Errorf("wallet.CalculateFeesFromProofs(proofsToSwap,keysets ).  %w", err)
		}

		var valueOfProofs uint64

		for _, v := range proofsToSwap {
			valueOfProofs += v.Proof.Amount
		}

		amountToAsk := valueOfProofs - uint64(fees)

		if amountToAsk == 0 {
			log.Println("Amount to swap after fees is 0 not making a swap")
			return nil
		}

		if counter.Counter == 0 {
			err = db.SetKeysetCounter(tx, counter)
			if err != nil {
				return fmt.Errorf("db.SetKeysetCounter(tx, counter). %w", err)

			}
		}

		blindMessages, secrets, keys, err := wallet.MakeBlindMessages(amountToAsk, mint_url, &counter)
		if err != nil {
			return fmt.Errorf("wallet.MakeBlindMessages(proofs, mint_url). %w", err)
		}

		blindSigs, err := wallet.SwapProofs(blindMessages, proofsToSwap, mint_url)
		if err != nil {
			return fmt.Errorf("wallet.SwapProofs(blindMessages, proofs, mint_url). %w", err)
		}

		err = db.ModifyKeysetCounter(tx, counter)
		if err != nil {
			return fmt.Errorf("db.ModifyKeysetCounter(tx, counter). %w", err)
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

		// Cs from used Proofs
		Cs := []string{}
		for i := 0; i < len(proofsToSwap); i++ {
			Cs = append(Cs, proofsToSwap[i].Proof.C)
		}

		err = db.ChangeLockedProofsRedeem(tx, Cs, true)
		if err != nil {
			return fmt.Errorf("db.ChangeLockedProofsRedeem(tx, Cs, true) %w", err)
		}

		err = db.AddProofs(tx, NewProofs, mint_url)
		if err != nil {
			return fmt.Errorf("db.AddProofs(tx, NewProofs, mint_url ) %w", err)
		}

	}

	return nil
}
