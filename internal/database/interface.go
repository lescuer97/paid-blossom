package database

import (
	"database/sql"
	"ratasker/external/blossom"

	"github.com/elnosh/gonuts/cashu"
)

type CurrentPubkey struct {
	VersionNum uint
	Expiration uint64
}

type KeysetCounter struct {
	KeysetId string `db:"keyset_id"`
	Counter  uint32 `db:"counter"`
}

type Database interface {
	BeginTransaction() (*sql.Tx, error)
	GetBlob(hash []byte) (blossom.DBBlobData, error)
	GetBlobLength(hash []byte) (uint64, error)

	AddBlob(tx *sql.Tx, data blossom.DBBlobData) error
	// RemoveBlob(data blossom.StoredData) error

	// Database actions for proofs
	AddLockedProofs(tx *sql.Tx, token cashu.Token, pubkey uint, redeemed bool, created_at uint64) error
	GetLockedProofsByPubkeyVersion(tx *sql.Tx, pubkey uint) (cashu.Proofs, error)
	GetLockedProofsByC(tx *sql.Tx, Cs []string) (cashu.Proofs, error)
	// should return proofs separated by the mint that they come from
	GetLockedProofsByRedeemed(tx *sql.Tx, redeemed bool) (map[string]cashu.Proofs, error)
	ChangeLockedProofsRedeem(tx *sql.Tx, Cs []string, redeem bool) error

	//For proofs that have already been swapped
	AddProofs(tx *sql.Tx, proofs cashu.Proofs, mint string) error
	GetBySpentProofs(tx *sql.Tx, spent bool) (map[string]cashu.Proofs, error)
	ChangeSwappedProofsSpent(tx *sql.Tx, proofs cashu.Proofs, spent bool) error

	AddTrustedMint(tx *sql.Tx, url string) error
	GetTrustedMints(tx *sql.Tx) ([]string, error)

	// take all pubkeys and turn active off and just make a new one
	RotateNewPubkey(tx *sql.Tx, expiration int64) (CurrentPubkey, error)
	GetActivePubkey(tx *sql.Tx) (CurrentPubkey, error)

	GetKeysetCounter(tx *sql.Tx, id string) (KeysetCounter, error)
	SetKeysetCounter(tx *sql.Tx, counter KeysetCounter) error
	ModifyKeysetCounter(tx *sql.Tx, counter KeysetCounter) error
}
