package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"ratasker/external/blossom"
	"ratasker/internal/utils"

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
			panic(p) // re-throw panic after rollback
		}
	}()
	_, err = tx.Exec("INSERT INTO blobs (sha256, size, path, created_at, pubkey, content_type) values (?, ?, ?, ?, ?, ?)")
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

	stmt, err := sq.Db.Prepare("SELECT sha256, size, path, created_at, pubkey, content_type FROM blobs WHERE sha256 = ?")
	if err != nil {
		return blobData, fmt.Errorf("sq.Db.Prepare(). %w", err)
	}
	defer stmt.Close()

	// Create a record to hold the result
	err = stmt.QueryRow(hash).Scan(&blobData.Sha256, &blobData.Data.Size, &blobData.Path, &blobData.CreatedAt, &blobData.Pubkey, &blobData.Data.Type)
	if err != nil {
		return blobData, fmt.Errorf("stmt.QueryRow(hash).Scan %w", err)
	}

	return blobData, nil

}
func (sq SqliteDB) GetBlobLength(hash []byte) (uint64, error) {
	var length uint64 = 0

	stmt, err := sq.Db.Prepare("SELECT size FROM blobs WHERE sha256 = ?")
	if err != nil {
		return length, fmt.Errorf("sq.Db.Prepare(). %w", err)
	}
	defer stmt.Close()

	// Create a record to hold the result
	err = stmt.QueryRow(hash).Scan(&length)
	if err != nil {
		return length, fmt.Errorf("stmt.QueryRow(hash).Scan %w", err)
	}

	return length, nil

}

func DatabaseSetup(ctx context.Context, migrationDir string) (SqliteDB, error) {
	var sqlitedb SqliteDB

	string, err := utils.GetRastaskerHomeDirectory()

	if err != nil {
		return sqlitedb, fmt.Errorf("utils.GetRastaskerHomeDirectory(). %w", err)

	}

	db, err := sql.Open("sqlite3", string+"/"+"app.db")
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
