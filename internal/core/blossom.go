package core

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"ratasker/external/blossom"
	"ratasker/external/xcashu"
	"ratasker/internal/cashu"
	"ratasker/internal/database"
	"ratasker/internal/io"
	"ratasker/internal/utils"
	"strconv"
	"time"
)

const (
	DOWNLOAD_COST_2MB = "DOWNLOAD_COST_2MB"
	UPLOAD_COST_2MB   = "UPLOAD_COST_2MB"
	OWNER_NPUB        = "OWNER_NPUB"
)

func WriteBlobAndCharge(c *gin.Context, wallet cashu.CashuWallet, db database.Database, fileHandler io.BlossomIO, cost uint64) error {
	quoteReq := c.GetHeader("content-length")
	buf := new(bytes.Buffer)

	_, err := buf.ReadFrom(c.Request.Body)
	if err != nil {
		log.Printf("buf.ReadFrom(c.Request.Body) %+v", err)
		c.JSON(500, "Somethig went wrong")
		return err
	}

	hash := sha256.Sum256(buf.Bytes())

	// check if hash already exists
	_, err = db.GetBlobLength(hash[:])
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("Chunk already exists %x", hash[:])
		type Error struct {
			Error string
		}
		c.JSON(201, Error{Error: "chuck exists"})
		return err
	}

	// Start DB transaction

	tx, err := db.BeginTransaction()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v\n", err)
	}

	// Ensure that the transaction is rolled back in case of a panic or error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			log.Fatalf("Panic occurred: %v\n", p)
		} else if err != nil {
			log.Println("Rolling back transaction due to error.")
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Fatalf("Failed to commit transaction: %v\n", err)
			}
			log.Println("Transaction committed successfully.")
		}
	}()

	// Check ecash amount correct
	contentLenght, err := strconv.ParseInt(quoteReq, 10, 64)
	if err != nil {
		c.JSON(400, "Malformed request")
		return err
	}

	mints, err := cashu.GetTrustedMintFromOsEnv()
	if err != nil {
		c.JSON(400, "Malformed request")
		return err
	}

	amountToPay := xcashu.QuoteAmountToPay(uint64(contentLenght), cost)
	paymentResponse := xcashu.PaymentQuoteResponse{
		Amount: amountToPay,
		Unit:   xcashu.Sat,
		Mints:  []string{mints},
		Pubkey: wallet.GetActivePubkey(),
	}

	jsonBytes, err := json.Marshal(paymentResponse)
	if err != nil {
		c.JSON(500, "Error request")
		return err
	}

	// In case you need to 402
	encodedPayReq := base64.URLEncoding.EncodeToString(jsonBytes)

	cashu_header := c.GetHeader(xcashu.Xcashu)

	token, err := xcashu.ParseTokenHeader(cashu_header, amountToPay)
	if err != nil {
		log.Printf(`xcashu.ParseTokenHeader(cashu_header, amountToPay) %+v`, err)
		c.JSON(402, encodedPayReq)
		return err
	}

	// Check Token is valid
	_, err = wallet.VerifyToken(token, tx, db)
	if err != nil {
		log.Printf(`wallet.VerifyToken(token, tx, db) %+v`, err)
		return err
	}

	err = wallet.StoreEcash(token, tx, db)
	if err != nil {
		log.Printf(`wallet.StoreEcash(proofs, tx, db) %+v`, err)
		return err
	}

	// check for upload payment
	hashHex := hex.EncodeToString(hash[:])

	blob := blossom.Blob{
		Data: buf.Bytes(),
		Size: uint64(buf.Len()),
		Type: c.ContentType(),
		Name: hashHex,
	}

	storedBlob := blossom.DBBlobData{
		Path:      fileHandler.GetStoragePath() + "/" + hashHex,
		Sha256:    hash[:],
		CreatedAt: uint64(time.Now().Unix()),
		Data:      blob,
		Pubkey:    "",
	}

	err = fileHandler.WriteBlob(hashHex, buf.Bytes())
	if err != nil {
		log.Printf(`fileHandler.WriteBlob(buf.Bytes()) %+v`, err)
		c.JSON(500, "Opss something went wrong")
		return err
	}
	log.Println("AFTER WRITING")

	err = db.AddBlob(tx, storedBlob)
	if err != nil {
		log.Printf(`db.AddBlob(storedBlob) %+v`, err)
		c.JSON(500, "Opss something went wrong")
		return err
	}

	blobDescriptor := blossom.BlobDescriptor{
		Url:      os.Getenv(utils.DOMAIN) + "/" + hashHex,
		Sha256:   hashHex,
		Size:     storedBlob.Data.Size,
		Uploaded: storedBlob.Pubkey,
		Type:     blob.Type,
	}
	c.JSON(200, blobDescriptor)

	return nil
}
