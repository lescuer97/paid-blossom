package database

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/elnosh/gonuts/cashu"
)

const TEST_MINT = "http://localhost:8080"

func TestRotatePubkey(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := DatabaseSetup(ctx, dir, "../../migrations")
	if err != nil {
		t.Fatalf("Could not setup db")
	}
	tx, err := sqlite.BeginTransaction()
	if err != nil {
		t.Fatalf("sqlite.BeginTransaction() %+v", err)
	}
	expirations := time.Now().Add(4 * time.Hour)
	current, err := sqlite.RotateNewPubkey(tx, expirations.Unix())
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if current.VersionNum != 1 {
		t.Errorf("should be version 0. got: %v", current.VersionNum)
	}
	current, err = sqlite.RotateNewPubkey(tx, expirations.Unix())
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if current.VersionNum != 2 {
		t.Errorf("should be version 1 got: %v", current.VersionNum)
	}
	err = tx.Commit()
	if err != nil {
		t.Fatalf("tx.Commit() %+v", err)
	}
}

func TestAddProofsAndGetForC(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := DatabaseSetup(ctx, dir, "../../migrations")
	if err != nil {
		t.Fatalf("Could not setup db")
	}
	tx, err := sqlite.BeginTransaction()
	if err != nil {
		t.Fatalf("sqlite.BeginTransaction() %+v", err)
	}
	expirations := time.Now().Add(4 * time.Hour)
	current, err := sqlite.RotateNewPubkey(tx, expirations.Unix())
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if current.VersionNum != 1 {
		t.Errorf("should be version 0. got: %v", current.VersionNum)
	}

	proofs := cashu.Proofs{
		{
			Id:      hex.EncodeToString([]byte("test")),
			Amount:  2,
			Secret:  "secret tedst1",
			C:       hex.EncodeToString([]byte("Ctest")),
			Witness: "",
		}, {
			Id:      hex.EncodeToString([]byte("test")),
			Amount:  2,
			Secret:  "secret tedst2",
			C:       hex.EncodeToString([]byte("Ctest2")),
			Witness: "",
		},
	}

	token1, err := cashu.NewTokenV4(proofs, TEST_MINT, cashu.Sat, false)
	if err != nil {
		t.Fatalf("cashu.NewTokenV4(proofs,TEST_MINT, cashu.Sat, false) %+v", err)
	}

	now := time.Now().Unix()
	err = sqlite.AddLockedProofs(tx, token1, current.VersionNum, false, uint64(now))
	if err != nil {
		t.Fatalf("sqlite.AddProofs(proofs,version, false, uint64(now) %+v", err)
	}

	newProofs, err := sqlite.GetLockedProofsByC(tx, []string{hex.EncodeToString([]byte("Ctest2"))})
	if err != nil {
		t.Fatalf(`sqlite.GetProofsByC([]string{"Ctest"}) %+v`, err)
	}
	if newProofs[0].C != hex.EncodeToString([]byte("Ctest2")) {
		t.Errorf(`Proof is wrong C %+v`, err)
	}
	if len(newProofs) != 1 {
		t.Errorf(`Wrong length of proofs %+v`, err)
	}
	err = tx.Commit()
	if err != nil {
		t.Fatalf("tx.Commit() %+v", err)
	}

}
func TestAddProofsAndGetViaPubkey(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := DatabaseSetup(ctx, dir, "../../migrations")

	if err != nil {
		t.Fatalf("Could not setup db")
	}

	tx, err := sqlite.BeginTransaction()
	if err != nil {
		t.Fatalf("sqlite.BeginTransaction() %+v", err)
	}
	expirations := time.Now().Add(4 * time.Hour)
	current, err := sqlite.RotateNewPubkey(tx, expirations.Unix())
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if current.VersionNum != 1 {
		t.Errorf("should be version 0. got: %v", current.VersionNum)
	}

	proofs := cashu.Proofs{
		{
			Id:      hex.EncodeToString([]byte("test")),
			Amount:  2,
			Secret:  "secret tedst1`",
			C:       hex.EncodeToString([]byte("Ctest")),
			Witness: "",
		}, {
			Id:      hex.EncodeToString([]byte("test")),
			Amount:  2,
			Secret:  "secret tedstswd",
			C:       hex.EncodeToString([]byte("Ctest2")),
			Witness: "",
		},
	}

	token1, err := cashu.NewTokenV4(proofs, TEST_MINT, cashu.Sat, false)
	if err != nil {
		t.Fatalf("cashu.NewTokenV4(proofs,TEST_MINT, cashu.Sat, false) %+v", err)
	}

	now := time.Now().Unix()
	err = sqlite.AddLockedProofs(tx, token1, current.VersionNum, false, uint64(now))
	if err != nil {
		t.Fatalf("sqlite.AddProofs(proofs,version, false, uint64(now) %+v", err)
	}

	// rotate pubkey and add proofs with new pubkey
	current, err = sqlite.RotateNewPubkey(tx, expirations.Unix())
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if current.VersionNum != 2 {
		t.Errorf("should be version 2. got: %v", current.VersionNum)
	}

	proofs2 := cashu.Proofs{
		{
			Id:      hex.EncodeToString([]byte("test")),
			Amount:  2,
			Secret:  "secret tedst3",
			C:       hex.EncodeToString([]byte("ctest34")),
			Witness: "",
		}, {
			Id:     hex.EncodeToString([]byte("test")),
			Amount: 2,
			Secret: "secret tedst4",
			C:      hex.EncodeToString([]byte("ctest23")),
		},
	}
	token2, err := cashu.NewTokenV4(proofs2, TEST_MINT, cashu.Sat, false)
	if err != nil {
		t.Fatalf("cashu.NewTokenV4(proofs,TEST_MINT, cashu.Sat, false) %+v", err)
	}

	now = time.Now().Unix()
	err = sqlite.AddLockedProofs(tx, token2, current.VersionNum, false, uint64(now))
	if err != nil {
		t.Fatalf("sqlite.AddProofs(proofs,version, false, uint64(now) %+v", err)
	}

	newProofs, err := sqlite.GetLockedProofsByPubkeyVersion(tx, current.VersionNum)
	if err != nil {
		t.Fatalf(`sqlite.GetProofsByC([]string{"Ctest"}) %+v`, err)
	}
	if newProofs[0].C != hex.EncodeToString([]byte("ctest34")) {
		t.Errorf(`Proof is wrong C %+v`, err)
	}
	if len(newProofs) != 2 {
		t.Errorf(`Wrong length of proofs %+v`, err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("tx.Commit() %+v", err)
	}
}

func TestCheckPubkeyVersion(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := DatabaseSetup(ctx, dir, "../../migrations")

	if err != nil {
		t.Fatalf("Could not setup db")
	}
	tx, err := sqlite.BeginTransaction()
	if err != nil {
		t.Fatalf("sqlite.BeginTransaction() %+v", err)
	}
	expirations := time.Now().Add(4 * time.Hour)

	current, err := sqlite.RotateNewPubkey(tx, expirations.Unix())
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if current.VersionNum != 1 {
		t.Errorf("should be version 1. got: %v", current.VersionNum)
	}
	current, err = sqlite.RotateNewPubkey(tx, expirations.Unix())
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if current.VersionNum != 2 {
		t.Errorf("should be version 2. got: %v", current.VersionNum)
	}
	_, err = sqlite.RotateNewPubkey(tx, expirations.Unix())
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	current, err = sqlite.GetActivePubkey(tx)
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if current.VersionNum != 3 {
		t.Errorf("should be version 3. got: %v", current.VersionNum)
	}
	err = tx.Commit()
	if err != nil {
		t.Fatalf("tx.Commit() %+v", err)
	}

}
func TestAddTrustedMints(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := DatabaseSetup(ctx, dir, "../../migrations")

	if err != nil {
		t.Fatalf("Could not setup db")
	}

	tx, err := sqlite.BeginTransaction()
	if err != nil {
		t.Fatalf("sqlite.BeginTransaction() %+v", err)
	}

	err = sqlite.AddTrustedMint(tx, "https://localhost.com")
	if err != nil {
		t.Fatalf(`sqlite.AddTrustedMint("https://localhost.com") %+v`, err)
	}

	err = sqlite.AddTrustedMint(tx, "https://localhost2.com")
	if err != nil {
		t.Fatalf(`sqlite.AddTrustedMint("https://localhost2.com") %+v`, err)
	}

	trustedMint, err := sqlite.GetTrustedMints(tx)
	if err != nil {
		t.Fatalf(`sqlite.GetTrustedMints() %+v`, err)
	}

	if len(trustedMint) != 2 {
		t.Error("There should be 2 trusted mints")
	}
	if trustedMint[0] != "https://localhost.com" {
		t.Error("There should be 2 trusted mints")
	}
	err = tx.Commit()
	if err != nil {
		t.Fatalf("tx.Commit() %+v", err)
	}
}

func TestAddTrustedMintsRollBack(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := DatabaseSetup(ctx, dir, "../../migrations")

	if err != nil {
		t.Fatalf("Could not setup db")
	}

	tx, err := sqlite.BeginTransaction()
	if err != nil {
		t.Fatalf("sqlite.BeginTransaction() %+v", err)
	}

	err = sqlite.AddTrustedMint(tx, "https://localhost.com")
	if err != nil {
		t.Fatalf(`sqlite.AddTrustedMint("https://localhost.com") %+v`, err)
	}

	trustedMint, err := sqlite.GetTrustedMints(tx)
	if err != nil {
		t.Fatalf(`sqlite.GetTrustedMints() %+v`, err)
	}

	if len(trustedMint) != 1 {
		t.Error("There should be 2 trusted mints")
	}
	if trustedMint[0] != "https://localhost.com" {
		t.Error("There should be 2 trusted mints")
	}
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("tx.Rollback() %+v", err)
	}

	tx, err = sqlite.BeginTransaction()
	if err != nil {
		t.Fatalf("sqlite.BeginTransaction() %+v", err)
	}

	trustedMint, err = sqlite.GetTrustedMints(tx)
	if err != nil {
		t.Fatalf(`sqlite.GetTrustedMints() %+v`, err)
	}
	if len(trustedMint) != 0 {
		t.Error("There should be 0 trusted mints")
	}
	tx.Commit()
}
