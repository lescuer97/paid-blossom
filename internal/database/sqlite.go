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

func (sq SqliteDB) AddBlob(data blossom.DBBlobData) error {
	tx, err := sq.Db.Begin()
	if err != nil {
		return fmt.Errorf("sq.Db.Begin(). %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()
	_, err = tx.Exec("INSERT INTO blobs (sha256, size, path, created_at, pubkey, content_type) values (?, ?, ?, ?, ?, ?)",
		data.Sha256, data.Data.Size, data.Path, data.CreatedAt, data.Pubkey, data.Data.Type,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf(`tx.Exec("INSERT INTO blobs (sha256, ). %w`, err)
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf(`tx.Commit(). %w`, err)
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

	stmt, err := tx.Prepare("SELECT size FROM blobs WHERE sha256 = ?")
	if err != nil {
		return length, fmt.Errorf("sq.Db.Prepare(). %w", err)
	}
	defer stmt.Close()

	// Create a record to hold the result
	err = stmt.QueryRow(hash).Scan(&length)
	if err != nil {
		return length, fmt.Errorf("stmt.QueryRow(hash).Scan %w", err)
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return length, fmt.Errorf(`tx.Commit(). %w`, err)
	}

	return length, nil
}
func (sq SqliteDB) AddProofs(data cashu.Proofs, pubkey_version uint, redeemed bool, created_at uint64) error {
	tx, err := sq.Db.Begin()
	if err != nil {
		return fmt.Errorf("sq.Db.Begin(). %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	stmt, err := tx.Prepare("INSERT INTO stored_proofs (amount, id, secret, C, witness, redeemed, created_at, pubkey_version) values (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf(`tx.Exec("INSERT INTO blobs (sha256, ). %w`, err)
	}
	defer stmt.Close()

	for _, proof := range data {
		_, err = stmt.Exec(proof.Amount, proof.Id, proof.Secret, proof.C, proof.Witness, redeemed, created_at, pubkey_version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("stmt.Exec(): %w", err)
		}

	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf(`tx.Commit(). %w`, err)
	}

	return nil
}

func (sq SqliteDB) GetProofsByPubkeyVersion(pubkey uint) (cashu.Proofs, error) {
	var proofs cashu.Proofs
	tx, err := sq.Db.Begin()
	if err != nil {
		return proofs, fmt.Errorf("sq.Db.Begin(). %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	stmt, err := tx.Prepare("SELECT amount, id, secret, C, witness FROM stored_proofs WHERE pubkey_version = ?")
	if err != nil {
		tx.Rollback()
		return proofs, fmt.Errorf(`tx.Exec("INSERT INTO blobs (sha256, ). %w`, err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(pubkey)
	if err != nil {
		tx.Rollback()
		return proofs, fmt.Errorf(`stmt.Query(pubkey). %w`, err)
	}
	defer rows.Close()

	for rows.Next() {
		var p cashu.Proof
		err = rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness)
		if err != nil {
			tx.Rollback()
			return proofs, fmt.Errorf(`ows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness) %w`, err)
		}

		proofs = append(proofs, p)
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return proofs, fmt.Errorf(`tx.Commit(). %w`, err)
	}

	return proofs, nil
}

func (sq SqliteDB) GetProofsByC(Cs []string) (cashu.Proofs, error) {
	var proofs cashu.Proofs
	tx, err := sq.Db.Begin()
	if err != nil {
		return proofs, fmt.Errorf("sq.Db.Begin(). %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()
	// Create the placeholders for the IN clause
	placeholders := make([]string, len(Cs))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"SELECT amount, id, secret, C, witness FROM stored_proofs WHERE C IN (%s)",
		strings.Join(placeholders, ","),
	)

	stmt, err := tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		return proofs, fmt.Errorf(`tx.Exec("INSERT INTO blobs (sha256, ). %w`, err)
	}
	defer stmt.Close()

	args := make([]interface{}, len(Cs))
	for i, v := range Cs {
		args[i] = v
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		tx.Rollback()
		return proofs, fmt.Errorf(`stmt.Query(args...). %w`, err)
	}
	defer rows.Close()

	for rows.Next() {
		var p cashu.Proof
		err = rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness)
		if err != nil {
			tx.Rollback()
			return proofs, fmt.Errorf(`rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness) %w`, err)
		}

		proofs = append(proofs, p)
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return proofs, fmt.Errorf(`tx.Commit(). %w`, err)
	}

	return proofs, nil
}

func (sq SqliteDB) ChangeRedeemState(Cs []string, redeem bool) error {
	var proofs cashu.Proofs
	tx, err := sq.Db.Begin()
	if err != nil {
		return fmt.Errorf("sq.Db.Begin(). %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()
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
		tx.Rollback()
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
		tx.Rollback()
		return fmt.Errorf(`stmt.Query(args...). %w`, err)
	}
	defer rows.Close()

	for rows.Next() {
		var p cashu.Proof
		err = rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf(`rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness) %w`, err)
		}

		proofs = append(proofs, p)
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf(`tx.Commit(). %w`, err)
	}

	return nil
}

func (sq SqliteDB) RotateNewPubkey() (uint, error) {
	tx, err := sq.Db.Begin()
	if err != nil {
		return 0, fmt.Errorf("sq.Db.Begin(). %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	updateQuery := `
    UPDATE cashu_pubkey 
    SET active = false 
    WHERE active = true;`
	// Then insert new active row and return it
	insertAndSelectQuery := `
        INSERT INTO cashu_pubkey (created_at, active)
        VALUES ($1, true)
        RETURNING version;
    `

	_, err = tx.Exec(updateQuery)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf(`tx.Exec(updateQuery) %w`, err)
	}

	now := time.Now().Unix()

	var version uint
	err = tx.QueryRow(insertAndSelectQuery, now).Scan(&version)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf(`tx.QueryRow(insertAndSelectQuery, now) %w`, err)
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf(`tx.Commit(). %w`, err)
	}

	return version, nil
}
func (sq SqliteDB) GetActivePubkey() (uint, error) {
	var version uint = 0
	tx, err := sq.Db.Begin()
	if err != nil {
		return 0, fmt.Errorf("sq.Db.Begin(). %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	stmt, err := tx.Prepare("SELECT version FROM cashu_pubkey WHERE active = true")
	if err != nil {
		return version, fmt.Errorf("sq.Db.Prepare(). %w", err)
	}
	defer stmt.Close()

	// Create a record to hold the result
	err = stmt.QueryRow().Scan(&version)
	if err != nil {
		tx.Rollback()
		return version, fmt.Errorf("stmt.QueryRow(hash).Scan %w", err)
	}
	tx.Commit()
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf(`tx.Commit(). %w`, err)
	}

	return version, nil
}
func (sq SqliteDB) GetTrustedMints() ([]string, error) {
	var mints []string
	tx, err := sq.Db.Begin()
	if err != nil {
		return mints, fmt.Errorf("sq.Db.Begin(). %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	stmt, err := tx.Prepare("SELECT url FROM trusted_mints")
	if err != nil {
		return mints, fmt.Errorf("sq.Db.Prepare(). %w", err)
	}
	defer stmt.Close()

	// Create a record to hold the result
	rows, err := stmt.Query()
	if err != nil {
		return mints, fmt.Errorf("stmt.Query() %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var url string
		err = rows.Scan(&url)
		if err != nil {
			tx.Rollback()
			return mints, fmt.Errorf(`rows.Scan(&p.Amount, &p.Id, &p.Secret, &p.C, &p.Witness) %w`, err)
		}

		mints = append(mints, url)
	}
	tx.Commit()

	return mints, nil
}
func (sq SqliteDB) AddTrustedMint(url string) error {
	tx, err := sq.Db.Begin()
	if err != nil {
		return fmt.Errorf("sq.Db.Begin(). %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

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

	tx.Commit()

	return nil
}

func DatabaseSetup(ctx context.Context, databaseDir string, migrationDir string) (SqliteDB, error) {
	var sqlitedb SqliteDB

	db, err := sql.Open("sqlite3", databaseDir+"/"+"app.db")
	if err != nil {
		return sqlitedb, fmt.Errorf(`sql.Open("sqlite3", string + "app.db" ). %w`, err)

	}

	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatalf("Error setting dialect: %v", err)
	}

	if err := goose.Up(db, migrationDir); err != nil {
		log.Fatalf("Error running migrations: %v", err)
	}

	sqlitedb.Db = db

	return sqlitedb, nil
}
