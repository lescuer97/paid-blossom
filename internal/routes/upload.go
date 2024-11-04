package routes

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"ratasker/external/blossom"
	n "ratasker/external/nostr"
	"ratasker/external/xcashu"
	"ratasker/internal/database"
	"ratasker/internal/io"
	"ratasker/internal/utils"
	"strconv"
	"time"

	w "github.com/elnosh/gonuts/wallet"
	"github.com/gin-gonic/gin"
)

const SatPerMegaByteUpload = 1

func UploadRoutes(r *gin.Engine, wallet *w.Wallet, db database.Database, fileHandler io.BlossomIO) {
	r.HEAD("/upload", func(c *gin.Context) {
		sha256Header := c.GetHeader(blossom.XSHA256)
		hash, err := hex.DecodeString(sha256Header)
		if err != nil {
			c.JSON(400, "No X-SHA-256 Header available")
			return
		}

		_, err = db.GetBlobLength(hash)
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("Chunk already exists %x", hash[:])
			c.JSON(201, n.NotifMessage{Message: "chuck exists"})
			return

		}

		quoteReq := c.GetHeader(blossom.XContentLength)
		contentLenght, err := strconv.ParseInt(quoteReq, 10, 64)
		if err != nil {
			c.JSON(400, "No X-Content-Length Header available")
			return
		}

		amount := xcashu.QuoteAmountToPay(uint64(contentLenght), SatPerMegaByteUpload)
		paymentResponse := xcashu.PaymentQuoteResponse{
			Amount: amount,
			Unit:   xcashu.Sat,
			Mints:  []string{wallet.CurrentMint()},
			Pubkey: hex.EncodeToString(wallet.GetReceivePubkey().SerializeCompressed()),
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
		quoteReq := c.GetHeader("content-length")

		buf := new(bytes.Buffer)

		_, err := buf.ReadFrom(c.Request.Body)
		if err != nil {
			log.Printf("buf.ReadFrom(c.Request.Body) %+v", err)
			c.JSON(500, "Somethig went wrong")
			return
		}

		hash := sha256.Sum256(buf.Bytes())

		// check if hash already exists
		_, err = db.GetBlobLength(hash[:])
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("Chunk already exists %x", hash[:])
			type Error struct {
				Error string
			}
			c.JSON(201, Error{Error: "chuck exists"})
			return

		}

		// Check ecash amount correct
		contentLenght, err := strconv.ParseInt(quoteReq, 10, 64)
		if err != nil {
			c.JSON(400, "Malformed request")
			return
		}

		amountToPay := xcashu.QuoteAmountToPay(uint64(contentLenght), SatPerMegaByteUpload)
		paymentResponse := xcashu.PaymentQuoteResponse{
			Amount: amountToPay,
			Unit:   xcashu.Sat,
			Mints:  []string{wallet.CurrentMint()},
			Pubkey: hex.EncodeToString(wallet.GetReceivePubkey().SerializeCompressed()),
		}

		jsonBytes, err := json.Marshal(paymentResponse)
		if err != nil {
			c.JSON(500, "Error request")
			return
		}

		// In case you need to 402
		encodedPayReq := base64.URLEncoding.EncodeToString(jsonBytes)

		cashu_header := c.GetHeader(xcashu.Xcashu)

		err = xcashu.VerifyTokenIsValid(cashu_header, amountToPay, wallet)
		if err != nil {
			log.Printf(`xcashu.VerifyTokenIsValid(cashu_header, amountToPay,wallet ) %+v`, err)
			c.JSON(402, encodedPayReq)
			return
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

		err = fileHandler.WriteBlob(buf.Bytes())
		if err != nil {
			log.Printf(`fileHandler.WriteBlob(buf.Bytes()) %+v`, err)
			c.JSON(500, "Opss something went wrong")
			return
		}

		err = db.AddBlob(storedBlob)
		if err != nil {
			log.Printf(`db.AddBlob(storedBlob) %+v`, err)
			c.JSON(500, "Opss something went wrong")
			return
		}

		blobDescriptor := blossom.BlobDescriptor{
			Url:      os.Getenv(utils.DOMAIN) + "/" + hashHex,
			Sha256:   hashHex,
			Size:     storedBlob.Data.Size,
			Uploaded: storedBlob.Pubkey,
			Type:     blob.Type,
		}
		c.JSON(200, blobDescriptor)

	})
}
