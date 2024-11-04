package database

import (
	"ratasker/external/blossom"

	"github.com/elnosh/gonuts/cashu"
)

type Database interface {
	AddBlob(data blossom.DBBlobData) error
	GetBlob(hash []byte) (blossom.DBBlobData, error)
	GetBlobLength(hash []byte) (uint64, error)
	// RemoveBlob(data blossom.StoredData) error

	// Database actions for proofs
	AddProofs(data cashu.Proofs, pubkey uint, redeemed bool, created_at uint64) error
	GetProofsByPubkeyVersion(pubkey uint) (cashu.Proofs, error)
	GetProofsByC(Cs []string) (cashu.Proofs, error)
	ChangeRedeemState(Cs []string, redeem bool) error

	AddTrustedMint(url string) error
	GetTrustedMints() ([]string, error)
	//
	// // take all pubkeys and turn active off and just make a new one
	RotateNewPubkey() (uint, error)
	GetActivePubkey() (uint, error)
}
