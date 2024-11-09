package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"ratasker/external/blossom"
	"strings"
	"time"

	"github.com/elnosh/gonuts/cashu"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

type SqliteDB struct {
	Db *sql.DB
}

func (sq SqliteDB) BeginTransaction() (*sql.Tx, error) {
	// sq.Db.BeginTx
	tx, err := sq.Db.Begin()
	if err != nil {
		return nil, fmt.Errorf("sq.Db.Begin(). %w", err)
	}

	return tx, nil
}

func (sq SqliteDB) AddBlob(tx *sql.Tx, data blossom.DBBlobData) error {
	stmt, err := tx.Prepare("INSERT INTO blobs (sha256, size, path, created_at, pubkey, content_type) values (?, ?, ?, ?, ?, ?)")

	if err != nil {
		return fmt.Errorf(`tx.Exec("INSERT INTO blobs (sha256, ). %w`, err)
	}
	_, err = stmt.Exec(data.Sha256, data.Data.Size, data.Path, data.CreatedAt, data.Pubkey, data.Data.Type)
	if err != nil {
		return fmt.Errorf(`stmt.Exec(data.Sha256, data.Data.Size, data.Path, data.CreatedAt, data.Pubkey, data.Data.Type). %w`, err)
	}
	return nil

}

func (sq SqliteDB) GetBlob(hash []byte) (blossom.DBBlobData, error) {
	blobData := blossom.DBBlobData{}
	tx, err := sq.Db.Begin()
	if err != nil {
		return blobData, fmt.Errorf("sq.Db.Begin(). %w", err)
	}

	stmt, err := tx.Prepare("SELECT sha256, size, path, created_at, pubkey, content_type FROM blobs WHERE sha256 = ?")
	if err != nil {
		return blobData, fmt.Errorf("sq.Db.Prepare(). %w", err)
	}
	defer stmt.Close()

	// Create a record to hold the result
	err = stmt.QueryRow(hash).Scan(&blobData.Sha256, &blobData.Data.Size, &blobData.Path, &blobData.CreatedAt, &blobData.Pubkey, &blobData.Data.Type)
	if err != nil {
		return blobData, fmt.Errorf("stmt.QueryRow(hash).Scan %w", err)
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return blobData, fmt.Errorf(`tx.Commit(). %w`, err)
	}

	return blobData, nil

}
func (sq SqliteDB) GetBlobLength(hash []byte) (uint64, error) {
	var length uint64 = 0

	tx, err := sq.Db.Begin()
	if err != nil {
		return length, fmt.Errorf("sq.Db.Begin(). %w", err)
	}
	// tx.St

	stmt, err := tx.Prepare("SELECT size FROM blobs WHERE sha256 = ?")
	if err != nil {
		return length, fmt.Errorf("sq.Db.Prepare(). %w", err)
	}
	defer stmt.Close()

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return length, fmt.Errorf(`tx.Commit(). %w`, err)
	}
	return length, nil
}
func (sq SqliteDB) AddLockedProofs(tx *sql.Tx, token cashu.Token, pubkey_version uint, redeemed bool, created_at uint64) error {

	stmt, err := tx.Prepare("INSERT INTO locked_proofs (amount, id, secret, C, witness, redeemed, created_at, pubkey_version, mint) values (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf(`tx.Prepare("INSERT INTO locked_proofs (amount, id, s. %w`, err)
	}
	defer stmt.Close()

	for _, proof := range token.Proofs() {
		_, err = stmt.Exec(proof.Amount, proof.Id, proof.Secret, proof.C, proof.Witness, redeemed, created_at, pubkey_version, token.Mint())
		if err != nil {
			return fmt.Errorf("stmt.Exec(): %w", err)
		}
	}
	return nil
}

func (sq SqliteDB) GetLockedProofsByPubkeyVersion(tx *sql.Tx, pubkey uint) (cashu.Proofs, error) {
	var proofs cashu.Proofs

	stmt, err := tx.Prepare("SELECT amount, id, secret, C, witness FROM locked_proofs WHERE pubkey_version = ?")
	if err != nil {
		return proofs, fmt.Errorf(`tx.Prepare("SELECT amount, id, secret, C, witness FROM locked_proofs. %w`, err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(pubkey)
	if err != nil {
		return proofs, fmt.Errorf(`stmt.Query(pubkey). %w`, err)
	}
	defer rows.Close()

	for rows.Next() {
		var p cashu.Proof
		err = rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness)
		if err != nil {
			return proofs, fmt.Errorf(`ows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness) %w`, err)
		}

		proofs = append(proofs, p)
	}

	return proofs, nil
}

func (sq SqliteDB) GetLockedProofsByRedeemed(tx *sql.Tx, redeemed bool) (map[string]cashu.Proofs, error) {
	proofs := make(map[string]cashu.Proofs)

	stmt, err := tx.Prepare("SELECT amount, id, secret, C, witness FROM locked_proofs WHERE redeemed = ?")
	if err != nil {
		return proofs, fmt.Errorf(`tx.Prepare("SELECT amount, id, secret, C. %w`, err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(redeemed)
	if err != nil {
		return proofs, fmt.Errorf(`stmt.Query(pubkey). %w`, err)
	}
	defer rows.Close()

	for rows.Next() {
		var p cashu.Proof
		var mint string
		err = rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness, &mint)
		if err != nil {
			return proofs, fmt.Errorf(`rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness, &mint) %w`, err)
		}

		proofs[mint] = append(proofs[mint], p)
	}

	return proofs, nil
}

func (sq SqliteDB) GetLockedProofsByC(tx *sql.Tx, Cs []string) (cashu.Proofs, error) {
	var proofs cashu.Proofs
	// Create the placeholders for the IN clause
	placeholders := make([]string, len(Cs))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"SELECT amount, id, secret, C, witness FROM locked_proofs WHERE C IN (%s)",
		strings.Join(placeholders, ","),
	)

	stmt, err := tx.Prepare(query)
	if err != nil {
		return proofs, fmt.Errorf(`tx.Prepare(query). %w`, err)
	}
	defer stmt.Close()

	args := make([]interface{}, len(Cs))
	for i, v := range Cs {
		args[i] = v
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		return proofs, fmt.Errorf(`stmt.Query(args...). %w`, err)
	}
	defer rows.Close()

	for rows.Next() {
		var p cashu.Proof
		err = rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness)
		if err != nil {
			return proofs, fmt.Errorf(`rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness) %w`, err)
		}

		proofs = append(proofs, p)
	}

	return proofs, nil
}

func (sq SqliteDB) ChangeRedeemState(tx *sql.Tx, Cs []string, redeem bool) error {
	var proofs cashu.Proofs

	// Create the placeholders for the IN clause
	placeholders := make([]string, len(Cs))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"UPDATE stored_proofs SET redeem = ? WHERE C IN (%s)",
		strings.Join(placeholders, ","),
	)

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf(`tx.Exec("INSERT INTO blobs (sha256, ). %w`, err)
	}
	defer stmt.Close()

	args := make([]interface{}, len(Cs))
	args[0] = redeem
	for i, v := range Cs {
		args[i+1] = v
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		return fmt.Errorf(`stmt.Query(args...). %w`, err)
	}
	defer rows.Close()

	for rows.Next() {
		var p cashu.Proof
		err = rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness)
		if err != nil {
			return fmt.Errorf(`rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness) %w`, err)
		}

		proofs = append(proofs, p)
	}

	return nil
}

func (sq SqliteDB) RotateNewPubkey(tx *sql.Tx, expiration int64) (CurrentPubkey, error) {

	var currentPubkey CurrentPubkey

	updateQuery := `
    UPDATE cashu_pubkey 
    SET active = false 
    WHERE active = true;`

	// Then insert new active row and return it
	insertAndSelectQuery := `
        INSERT INTO cashu_pubkey (created_at, active)
        VALUES ($1, true)
        RETURNING version, created_at;
    `

	_, err := tx.Exec(updateQuery)
	if err != nil {
		return currentPubkey, fmt.Errorf(`tx.Exec(updateQuery) %w`, err)
	}

	err = tx.QueryRow(insertAndSelectQuery, expiration).Scan(&currentPubkey.VersionNum, &currentPubkey.Expiration)
	if err != nil {
		return currentPubkey, fmt.Errorf(`tx.QueryRow(insertAndSelectQuery, now) %w`, err)
	}

	return currentPubkey, nil
}

func (sq SqliteDB) GetActivePubkey(tx *sql.Tx) (CurrentPubkey, error) {
	var currentPubkey CurrentPubkey

	stmt, err := tx.Prepare("SELECT version, created_at FROM cashu_pubkey WHERE active = true")
	if err != nil {
		return currentPubkey, fmt.Errorf("sq.Db.Prepare(). %w", err)
	}
	defer stmt.Close()

	// Create a record to hold the result
	err = stmt.QueryRow().Scan(&currentPubkey.VersionNum, &currentPubkey.Expiration)
	if err != nil {
		return currentPubkey, fmt.Errorf("stmt.QueryRow(hash).Scan %w", err)
	}

	return currentPubkey, nil
}
func (sq SqliteDB) GetTrustedMints(tx *sql.Tx) ([]string, error) {
	var mints []string

	rows, err := tx.Query("SELECT url FROM trusted_mints")

	// Create a record to hold the result
	if err != nil {
		return mints, fmt.Errorf("SELECT url FROM trusted_mints %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var url string = ""
		err = rows.Scan(&url)
		if err != nil {
			return mints, fmt.Errorf(`rows.Scan(&url) %w`, err)
		}

		mints = append(mints, url)
	}

	return mints, nil
}
func (sq SqliteDB) AddTrustedMint(tx *sql.Tx, url string) error {
	now := time.Now().Unix()
	stmt, err := tx.Prepare("INSERT INTO trusted_mints (url, created_at) values (?,?)")
	if err != nil {
		return fmt.Errorf("sq.Db.Prepare(). %w", err)
	}
	defer stmt.Close()

	// Create a record to hold the result
	_, err = stmt.Exec(url, now)
	if err != nil {
		return fmt.Errorf("stmt.Query() %w", err)
	}
	return nil
}
func (sq SqliteDB) SetKeysetCounter(tx *sql.Tx, counter KeysetCounter) error {
	stmt, err := tx.Prepare("INSERT INTO trusted_mints (keyset_id, counter) values (?,?)")
	if err != nil {
		return fmt.Errorf("sq.Db.Prepare(). %w", err)
	}
	defer stmt.Close()

	// Create a record to hold the result
	_, err = stmt.Exec(counter.KeysetId, counter.Counter)
	if err != nil {
		return fmt.Errorf("stmt.Query() %w", err)
	}
	return nil
}

func (sq SqliteDB) GetKeysetCounter(tx *sql.Tx, id string) (KeysetCounter, error) {
	var counter KeysetCounter

	stmt, err := tx.Prepare("SELECT keyset_id, counter WHERE keyset_id = ?")
	if err != nil {
		return counter, fmt.Errorf(`SELECT keyset_id, counter WHERE keyset_id = ?. %w`, err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(id)
	if err != nil {
		return counter, fmt.Errorf(`stmt.Query(pubkey). %w`, err)
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&counter.KeysetId, &counter.Counter)
		if err != nil {
			return counter, fmt.Errorf(`rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness, &mint) %w`, err)
		}
	}

	return counter, nil
}

func DatabaseSetup(ctx context.Context, databaseDir string, migrationDir string) (SqliteDB, error) {
	var sqlitedb SqliteDB

	db, err := sql.Open("sqlite3", databaseDir+"/"+"app.db"+"?cache=shared&mode=rwc&_journal_mode=WAL")
	if err != nil {
		return sqlitedb, fmt.Errorf(`sql.Open("sqlite3", string + "app.db" ). %w`, err)

	}

	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatalf("Error setting dialect: %v", err)
	}

	if err := goose.Up(db, migrationDir); err != nil {
		log.Fatalf("Error running migrations: %v", err)
	}
	db.SetMaxOpenConns(1)

	sqlitedb.Db = db

	return sqlitedb, nil
}
