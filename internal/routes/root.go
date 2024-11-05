package routes

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"ratasker/external/xcashu"
	"ratasker/internal/cashu"
	"ratasker/internal/database"
	"ratasker/internal/io"

	"github.com/gin-gonic/gin"
)

const SatPerMegaByteDownload = 1

func RootRoutes(r *gin.Engine, wallet cashu.CashuWallet, db database.Database, fileHandler io.BlossomIO) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, nil)
	})

	r.GET("/:sha", func(c *gin.Context) {
		sha := c.Param("sha")

		log.Println("got hash: ", sha)
		// try to get blob
		hash, err := hex.DecodeString(sha)
		if err != nil {
			log.Printf(`hex.DecodeString(sha) %+v`, err)
			c.JSON(500, "Opps! Server error")
			return
		}

		blob, err := db.GetBlob(hash)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(404, nil)
			}
			log.Printf(`sqlite.GetBlob(hash) %+v`, err)
			c.JSON(500, "Opps! Server error")
			return
		}

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
		mints, err := db.GetTrustedMints(tx)
		if err != nil {
			c.JSON(400, "Malformed request")
			return
		}


		amountToPay := xcashu.QuoteAmountToPay(uint64(blob.Data.Size), SatPerMegaByteDownload)
		paymentResponse := xcashu.PaymentQuoteResponse{
			Amount: amountToPay,
			Unit:   xcashu.Sat,
			Mints:  mints,
			Pubkey: wallet.GetActivePubkey(),
		}

		jsonBytes, err := json.Marshal(paymentResponse)
		if err != nil {
			c.JSON(500, "Error request")
			return
		}

		// In case you need to 402
		encodedPayReq := base64.URLEncoding.EncodeToString(jsonBytes)

		// check for 50 sats payment
		cashu_header := c.GetHeader(xcashu.Xcashu)
		if cashu_header == "" {
			log.Println("cashu header not available")
			c.Header(xcashu.Xcashu, encodedPayReq)
			c.JSON(402, "payment required")
			return
		}

		_, err = xcashu.ParseTokenHeader(cashu_header, amountToPay)
		if err != nil {
			log.Printf(`xcashu.VerifyTokenIsValid(cashu_header, amountToPay,wallet ) %+v`, err)
			c.Header(xcashu.Xcashu, encodedPayReq)
			c.JSON(402, "payment required")
			return
		}

		fileBytes, err := fileHandler.GetBlob(blob.Path)

		if err != nil {
			log.Printf(`fileHandler.GetBlob(blob.Path) %+v`, err)
			c.JSON(500, "Opps! Server error")
			return
		}

		// check if sha256 is the same
		fileHash := sha256.Sum256(fileBytes)
		if sha != hex.EncodeToString(fileHash[:]) {
			log.Printf("HASHes are different")
			c.JSON(500, "Opps! Server error")
			return
		}

		_, err = c.Writer.Write(fileBytes)
		if err != nil {
			log.Printf(`c.Writer.Write(fileBytes) %+v`, err)
			c.JSON(500, "Opps! Server error")
			return
		}
		c.Header("Content-Type", blob.Data.Type)
	})

	r.HEAD("/:sha", func(c *gin.Context) {
		sha := c.Param("sha")
		hash, err := hex.DecodeString(sha)
		if err != nil {
			log.Printf(`hex.DecodeString(sha) %+v`, err)
			c.JSON(500, "Opps! Server error")
			return
		}

		length, err := db.GetBlobLength(hash)
		if err != nil {
			log.Printf(`hex.DecodeString(sha) %+v`, err)
			c.JSON(500, "Opps! Server error")
			return
		}
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
		mints, err := db.GetTrustedMints(tx)
		if err != nil {
			c.JSON(400, "Malformed request")
			return
		}

		amount := xcashu.QuoteAmountToPay(length, SatPerMegaByteDownload)
		paymentResponse := xcashu.PaymentQuoteResponse{
			Amount: amount,
			Unit:   xcashu.Sat,
			Mints:  mints,
			Pubkey: wallet.GetActivePubkey(),
		}

		jsonBytes, err := json.Marshal(paymentResponse)
		if err != nil {
			c.JSON(500, "Error request")
			return
		}

		encodedPayReq := base64.URLEncoding.EncodeToString(jsonBytes)

		c.Header(xcashu.Xcashu, encodedPayReq)
		c.JSON(402, nil)
		return
	})

}
