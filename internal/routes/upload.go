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
	"ratasker/external/xcashu"
	"ratasker/internal/database"
	"strconv"
	"time"

	w "github.com/elnosh/gonuts/wallet"
	"github.com/gin-gonic/gin"
)

const SatPerMegaByteUpload = 1

func UploadRoutes(r *gin.Engine, wallet *w.Wallet, sqlite database.Database, pathToData string) {
	r.HEAD("/upload", func(c *gin.Context) {
		quoteReq := c.GetHeader(xcashu.XContentLength)
		log.Println("QuoteReq: ", quoteReq)

		contentLenght, err := strconv.ParseInt(quoteReq, 10, 64)
		if err != nil {
			c.JSON(400, "Malformed request")
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
			log.Panic(`buf.ReadFrom(c.Request.Body) %w`, err)
		}

		hash := sha256.Sum256(buf.Bytes())

		// check if hash already exists
		_, err = sqlite.GetBlobLength(hash[:])
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("Chunk already exists %x", hash[:])
			c.JSON(204, "Chunk already exists")
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
			log.Printf(`xcashu.VerifyTokenIsValid(cashu_header, amountToPay,wallet ) %w`, err)
			c.JSON(402, encodedPayReq)
			return
		}

		// check for upload payment

		hashHex := hex.EncodeToString(hash[:])
		blob := blossom.Blob{
			Data: buf.Bytes(),
			Size: uint64(buf.Len()),
			Name: hashHex,
		}

		storedBlob := blossom.DBBlobData{
			Path:      pathToData + "/" + hashHex,
			Sha256:    hash[:],
			CreatedAt: uint64(time.Now().Unix()),
			Data:      blob,
		}

		err = os.WriteFile(storedBlob.Path, buf.Bytes(), 0764)
		if err != nil {
			log.Panic(`os.WriteFile(pathToData %w`, err)
		}

		err = sqlite.AddBlob(storedBlob)
		if err != nil {
			log.Panic(`sqlite.AddBlob()`, err)
		}
		c.JSON(200, "yeyyy")

	})
}
