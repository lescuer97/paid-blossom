package routes

import (
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"ratasker/external/blossom"
	n "ratasker/external/nostr"
	"ratasker/external/xcashu"
	"ratasker/internal/cashu"
	"ratasker/internal/core"
	"ratasker/internal/database"
	"ratasker/internal/io"
	"strconv"

	"github.com/gin-gonic/gin"
)

func UploadRoutes(r *gin.Engine, wallet cashu.CashuWallet, db database.Database, fileHandler io.BlossomIO, cost uint64) {
	r.HEAD("/upload", func(c *gin.Context) {
		sha256Header := c.GetHeader(blossom.XSHA256)
		log.Println("sha256Header: ", sha256Header)
		hash, err := hex.DecodeString(sha256Header)
		if err != nil {
			c.JSON(400, "No X-SHA-256 Header available")
			return
		}

		_, err = db.GetBlobLength(hash)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("Chunk already exists %x. %+v", hash[:], err)
			c.JSON(201, n.NotifMessage{Message: "chuck exists"})
			return

		}

		quoteReq := c.GetHeader(blossom.XContentLength)
		contentLenght, err := strconv.ParseInt(quoteReq, 10, 64)
		if err != nil {
			c.JSON(400, "No X-Content-Length Header available")
			return
		}

		tx, err := db.BeginTransaction()
		if err != nil {
			c.JSON(400, "Malformed request")
			return
		}

		// Ensure that the transaction is rolled back in case of a panic or error
		defer func() {
			if p := recover(); p != nil {
				tx.Rollback()
				log.Fatalf("Panic occurred: %v\n", p)
			} else if err != nil {
				log.Println("Rolling back transaction due to error. err: ", err)
				tx.Rollback()
			} else {
				err = tx.Commit()
				if err != nil {
					log.Fatalf("Failed to commit transaction: %v\n", err)
				}
			}
		}()

		mints, err := cashu.GetTrustedMintFromOsEnv()
		if err != nil {
			c.JSON(400, "Malformed request")
			return
		}

		amount := xcashu.QuoteAmountToPay(uint64(contentLenght), cost)
		paymentResponse := xcashu.PaymentQuoteResponse{
			Amount: amount,
			Unit:   xcashu.Sat,
			Mints:  []string{mints},
			Pubkey: wallet.GetActivePubkey(),
		}
		jsonBytes, err := json.Marshal(paymentResponse)
		if err != nil {
			c.JSON(500, "Error request")
			return
		}
		encodedPayReq := base64.URLEncoding.EncodeToString(jsonBytes)
		c.Header(xcashu.Xcashu, encodedPayReq)
		c.Status(402)
		return
	})

	r.PUT("/upload", func(c *gin.Context) {
		err := core.WriteBlobAndCharge(c, wallet, db, fileHandler, cost)

		if err != nil {
			log.Printf("core.WriteBlobAndCharge(). %+v", err)

			c.JSON(400, "Opps!")
		}

	})
}
