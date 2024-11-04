package database

import (
	"context"
	"github.com/elnosh/gonuts/cashu"
	"testing"
	"time"
)

func TestRotatePubkey(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := DatabaseSetup(ctx, dir, "../../migrations")
	if err != nil {
		t.Fatalf("Could not setup db")
	}
	version, err := sqlite.RotateNewPubkey()
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if version != 1 {
		t.Errorf("should be version 0. got: %v", version)
	}

	version, err = sqlite.RotateNewPubkey()
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if version != 2 {
		t.Errorf("should be version 1 got: %v", version)
	}
}

func TestAddProofsAndGetForC(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := DatabaseSetup(ctx, dir, "../../migrations")
	if err != nil {
		t.Fatalf("Could not setup db")
	}
	version, err := sqlite.RotateNewPubkey()
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if version != 1 {
		t.Errorf("should be version 0. got: %v", version)
	}

	proofs := cashu.Proofs{
		{
			Id:      "test",
			Amount:  2,
			Secret:  "secret tedst1",
			C:       "Ctest",
			Witness: "",
		}, {
			Id:      "test",
			Amount:  2,
			Secret:  "secret tedst2",
			C:       "Ctest2",
			Witness: "",
		},
	}

	now := time.Now().Unix()
	err = sqlite.AddProofs(proofs, version, false, uint64(now))
	if err != nil {
		t.Fatalf("sqlite.AddProofs(proofs,version, false, uint64(now) %+v", err)
	}

	newProofs, err := sqlite.GetProofsByC([]string{"Ctest"})
	if err != nil {
		t.Fatalf(`sqlite.GetProofsByC([]string{"Ctest"}) %+v`, err)
	}
	if newProofs[0].C != "Ctest" {
		t.Errorf(`Proof is wrong C %+v`, err)
	}
	if len(newProofs) != 1 {
		t.Errorf(`Wrong length of proofs %+v`, err)

	}

}
func TestAddProofsAndGetViaPubkey(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := DatabaseSetup(ctx, dir, "../../migrations")

	if err != nil {
		t.Fatalf("Could not setup db")
	}
	version, err := sqlite.RotateNewPubkey()
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if version != 1 {
		t.Errorf("should be version 0. got: %v", version)
	}

	proofs := cashu.Proofs{
		{
			Id:      "test",
			Amount:  2,
			Secret:  "secret tedst1`",
			C:       "Ctest",
			Witness: "",
		}, {
			Id:      "test",
			Amount:  2,
			Secret:  "secret tedstswd",
			C:       "Ctest2",
			Witness: "",
		},
	}

	now := time.Now().Unix()
	err = sqlite.AddProofs(proofs, version, false, uint64(now))
	if err != nil {
		t.Fatalf("sqlite.AddProofs(proofs,version, false, uint64(now) %+v", err)
	}
	// rotate pubkey and add proofs with new pubkey
	version, err = sqlite.RotateNewPubkey()
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if version != 2 {
		t.Errorf("should be version 2. got: %v", version)
	}

	proofs2 := cashu.Proofs{
		{
			Id:      "test",
			Amount:  2,
			Secret:  "secret tedst3",
			C:       "Ctest3",
			Witness: "",
		}, {
			Id:     "test",
			Amount: 2,
			Secret: "secret tedst4",
			C:      "Ctest24",
		},
	}

	now = time.Now().Unix()
	err = sqlite.AddProofs(proofs2, version, false, uint64(now))
	if err != nil {
		t.Fatalf("sqlite.AddProofs(proofs,version, false, uint64(now) %+v", err)
	}

	newProofs, err := sqlite.GetProofsByPubkeyVersion(version)
	if err != nil {
		t.Fatalf(`sqlite.GetProofsByC([]string{"Ctest"}) %+v`, err)
	}
	if newProofs[0].C != "Ctest3" {
		t.Errorf(`Proof is wrong C %+v`, err)
	}
	if len(newProofs) != 2 {
		t.Errorf(`Wrong length of proofs %+v`, err)
	}

}

func TestCheckPubkeyVersion(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := DatabaseSetup(ctx, dir, "../../migrations")

	if err != nil {
		t.Fatalf("Could not setup db")
	}
	version, err := sqlite.RotateNewPubkey()
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if version != 1 {
		t.Errorf("should be version 1. got: %v", version)
	}
	version, err = sqlite.RotateNewPubkey()
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if version != 2 {
		t.Errorf("should be version 2. got: %v", version)
	}
	_, err = sqlite.RotateNewPubkey()
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	version, err = sqlite.GetActivePubkey()
	if err != nil {
		t.Fatalf("sqlite.RotateNewPubkey() %+v", err)
	}
	if version != 3 {
		t.Errorf("should be version 3. got: %v", version)
	}

}
func TestAddTrustedMints(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := DatabaseSetup(ctx, dir, "../../migrations")

	if err != nil {
		t.Fatalf("Could not setup db")
	}
	err = sqlite.AddTrustedMint("https://localhost.com")
	if err != nil {
		t.Fatalf(`sqlite.AddTrustedMint("https://localhost.com") %+v`, err)
	}

	err = sqlite.AddTrustedMint("https://localhost2.com")
	if err != nil {
		t.Fatalf(`sqlite.AddTrustedMint("https://localhost2.com") %+v`, err)
	}

	trustedMint, err := sqlite.GetTrustedMints()
	if err != nil {
		t.Fatalf(`sqlite.GetTrustedMints() %+v`, err)
	}

	if len(trustedMint) != 2 {
		t.Error("There should be 2 trusted mints")
	}
	if trustedMint[0] != "https://localhost.com" {
		t.Error("There should be 2 trusted mints")
	}
}
