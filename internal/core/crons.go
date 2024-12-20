package core

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"ratasker/internal/cashu"
	"ratasker/internal/database"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	c "github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/cashu/nuts/nut12"
	"github.com/elnosh/gonuts/crypto"
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
		log.Println("\n trying to publish event to relay: ", key)
		if err := value.Publish(context.Background(), ev); err != nil {
			log.Printf("\ncould not publish event. relay: %+v. Err: %+v\n\n", key, err)
			return true
		}

		return true
	})

	return nil
}

// take the redeem proofs and send them to a nostr user
func SendProofsToOwner(wallet cashu.CashuWallet, db database.Database, tx *sql.Tx, pubkey string) error {

	mintsProofs, err := db.GetBySpentProofs(tx, false)
	if err != nil {
		return fmt.Errorf("db.GetBySpentProofs(tx, false ). %w", err)
	}

	ctx := context.Background()

	privKey := nostr.GeneratePrivateKey()
	pool := nostr.NewSimplePool(ctx)

	// get relays of the nostr user
	err = GetRelaysFromNIP65Pubkey(pubkey, discoveryRelay, pool)
	if err != nil {
		return fmt.Errorf("GetRelaysFromNIP65Pubkey(pubkey, pool). %w", err)
	}

	conversationKey, err := nip44.GenerateConversationKey(pubkey, privKey)
	if err != nil {
		return fmt.Errorf("nip44.GenerateConversationKey(pubkey, privKey). %w", err)
	}

	for key, val := range mintsProofs {
		token, err := c.NewTokenV4(val, key, c.Sat, false)
		if err != nil {
			return fmt.Errorf("c.NewTokenV4(val, key, c.Sat, true). %w", err)
		}
		tokenString, err := token.Serialize()
		if err != nil {
			return fmt.Errorf("token.Serialize(). %w", err)
		}
		log.Printf("\n token to redeem: %+v \n", tokenString)

		encryptedString, err := nip44.Encrypt(tokenString, conversationKey)
		if err != nil {
			return fmt.Errorf("nip44.Encrypt(tokenString, conversationKey). %w", err)
		}
		err = SendEncryptedProofsToPubkey(privKey, encryptedString, pubkey, pool) // send to user
		if err != nil {
			return fmt.Errorf("SendEncryptedProofsToPubkey(privKey, encryptedString, pubkey,pool). %w", err)
		}

		err = db.ChangeSwappedProofsSpent(tx, val, true)
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
				log.Println("Setting counter to 0 for keysetid: ", keyset.Id)
				counter.Counter = 0
				counter.KeysetId = keyset.Id
			} else {
				return fmt.Errorf("db.GetKeysetCounter(tx,keyset.Id ). %w", err)
			}
		}

		blindMessages, secrets, keys, err := wallet.MakeBlindMessages(proofsToSwap.Amount(), mint_url, &counter)
		if err != nil {
			return fmt.Errorf("wallet.MakeBlindMessages(proofs, mint_url). %w", err)
		}

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

		// Cs from used Proofs
		Cs := []string{}
		for i := 0; i < len(proofsToSwap); i++ {
			Cs = append(Cs, proofsToSwap[i].C)
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
