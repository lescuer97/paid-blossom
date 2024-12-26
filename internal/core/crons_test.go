package core

import (
	"context"
	"testing"

	"github.com/nbd-wtf/go-nostr"
	n "github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

const npubTest = "npub1d7exvqfvxqyrq0j54e23gz6xj4lfj7qfssqamg60fkfp5f6mlzaskklrf3"

func TestCheckForRelayExistence(t *testing.T) {
	_, pubkey, err := nip19.Decode(npubTest)

	if err != nil {
		t.Fatalf("npub is incorrect, %+v", err)
	}

	pool := n.NewSimplePool(context.Background())
	err = GetRelaysFromNIP65Pubkey(pubkey.(string), discoveryRelay, pool)
	if err != nil {
		t.Errorf("GetRelaysFromNIP65Pubkey(pubkeyTest, discoveryRelay, pool). %+v", err)
	}

	if pool.Relays.Size() < 1 {

		t.Errorf("function did not get relays. %+v", pool.Relays)
	}

}
func TestCheckSendingMessage(t *testing.T) {
	_, pubkey, err := nip19.Decode(npubTest)

	if err != nil {
		t.Fatalf("npub is incorrect, %+v", err)
	}

	pool := n.NewSimplePool(context.Background())
	err = GetRelaysFromNIP65Pubkey(pubkey.(string), discoveryRelay, pool)
	if err != nil {
		t.Errorf("GetRelaysFromNIP65Pubkey(pubkeyTest, discoveryRelay, pool). %+v", err)
	}

	if pool.Relays.Size() < 1 {

		t.Errorf("function did not get relays. %+v", pool.Relays)
	}

	privkey := nostr.GeneratePrivateKey()
	err = SendEncryptedProofsToPubkey(privkey, "test from sending to encrypted proofs", pubkey.(string), pool)
	if err != nil {
		t.Errorf(`SendEncryptedProofsToPubkey(privkey, "test", pubkey.(string), pool) %+v`, err)
	}
}
