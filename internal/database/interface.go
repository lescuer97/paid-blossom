package database

import (
	"database/sql"
	"ratasker/external/blossom"

	"github.com/elnosh/gonuts/cashu"
)

type Database interface {
	BeginTransaction() (*sql.Tx, error)
	GetBlob(hash []byte) (blossom.DBBlobData, error)
	GetBlobLength(hash []byte) (uint64, error)

	AddBlob(tx *sql.Tx, data blossom.DBBlobData) error
	// RemoveBlob(data blossom.StoredData) error

	// Database actions for proofs
	AddProofs(tx *sql.Tx, data cashu.Proofs, pubkey uint, redeemed bool, created_at uint64) error
	GetProofsByPubkeyVersion(tx *sql.Tx, pubkey uint) (cashu.Proofs, error)
	GetProofsByC(tx *sql.Tx, Cs []string) (cashu.Proofs, error)
	ChangeRedeemState(tx *sql.Tx, Cs []string, redeem bool) error

	AddTrustedMint(tx *sql.Tx, url string) error
	GetTrustedMints(tx *sql.Tx) ([]string, error)
	//
	// // take all pubkeys and turn active off and just make a new one
	RotateNewPubkey(tx *sql.Tx) (uint, error)
	GetActivePubkey(tx *sql.Tx) (uint, error)
}
