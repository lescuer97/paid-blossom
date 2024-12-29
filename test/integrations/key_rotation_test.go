package integrations

import (
	"context"
	"ratasker/internal/cashu"
	"ratasker/internal/core"
	"ratasker/internal/database"
	"testing"

	c "github.com/elnosh/gonuts/cashu"
	"github.com/elnosh/gonuts/wallet"
)

const URL_FOR_MINT = ""
const TEST_SEED = "speed grid safe equal monkey maple submit finish elite potato gather coffee"

func TestKeyRotation(t *testing.T) {

	testDir := t.TempDir()
	ctx := context.Background()

	t.Setenv(cashu.TRUSTED_MINT, "http://127.0.0.1:8080")

	// tmpl, err := tmpl.ParseFS(embedMigrations, "templates/layout.gohtml")

	// Setup DAtabase
	sqlite, err := database.DatabaseSetup(ctx, testDir, database.EmbedMigrations)
	if err != nil {
		t.Fatalf("Could not setup db")
	}

	nativeWallet, err := cashu.NewDBLocalWallet(TEST_SEED, sqlite)
	if err != nil {
		t.Fatalf("cashu.NewDBLocalWallet(TEST_SEED, sqlite). %+v", err)
	}

	// setup wallet to get proofs for storing into locked proofs
	config := wallet.Config{
		WalletPath:     testDir,
		CurrentMintURL: "http://127.0.0.1:8080",
	}

	senderWallet, err := wallet.LoadWallet(config)
	if err != nil {
		t.Fatalf("Could not setup sender Wallet. \n wallet.LoadWallet(config). %+v", err)
	}

	// get proofs
	mintQuote, err := senderWallet.RequestMint(1000)
	if err != nil {
		t.Fatalf(" \n senderWallet.RequestMint(1000). %+v", err)
	}

	_, err = senderWallet.MintTokens(mintQuote.Quote)
	if err != nil {
		t.Fatalf(" \n senderWallet.MintTokens(mintQuote.Quote). %+v", err)
	}

	proofs, err := senderWallet.SendToPubkey(600, senderWallet.CurrentMint(), nativeWallet.CurrentPubkey, nil, false)
	if err != nil {
		t.Fatalf(" \n senderWallet.SendToPubkey(600,senderWallet.CurrentMint(),. %+v", err)
	}

	tokenForProofs, err := c.NewTokenV4(proofs, senderWallet.CurrentMint(), c.Sat, true)
	if err != nil {
		t.Fatalf(" \n c.NewTokenV4(proofs, senderWallet.CurrentMint(), c.Sat, true ) %+v", err)
	}

	tx, err := sqlite.BeginTransaction()
	if err != nil {
		t.Fatalf(" \n sqlite.BeginTransaction() %+v", err)
	}

	// Store locked Proofs
	err = nativeWallet.StoreEcash(tokenForProofs, tx, sqlite)
	if err != nil {
		t.Fatalf(" \n nativeWallet.StoreEcash(tokenForProofs, tx, sqlite) %+v", err)
	}

	err = core.RotateLockedProofs(&nativeWallet, sqlite, tx)
	if err != nil {
		t.Fatalf(" \n core.RotateLockedProofs(&nativeWallet, sqlite) %+v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf(" \n tx.Commit() %+v", err)
	}

	// check if I don't have any proofs unredeemed and check if the other proofs are stored

}
