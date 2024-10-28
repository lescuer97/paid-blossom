package routes

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/elnosh/gonuts/cashu"
	w "github.com/elnosh/gonuts/wallet"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"ratasker/external/xcashu"
	"ratasker/internal/database"
)

const SatPerMegaByteDownload = 2

func RootRoutes(r *gin.Engine, wallet *w.Wallet, sqlite database.Database) {
	r.GET("/:sha", func(c *gin.Context) {
		sha := c.Param("sha")

		log.Println("got hash: ", sha)
		// try to get blob
		hash, err := hex.DecodeString(sha)
		if err != nil {
			log.Printf(`hex.DecodeString(sha) %w`, err)
			c.JSON(500, "Opps! Server error")
			return
		}

		blob, err := sqlite.GetBlob(hash)
		if err != nil {
			log.Printf(`sqlite.GetBlob(hash) %w`, err)
			c.JSON(500, "Opps! Server error")
			return
		}

		amountToPay := xcashu.QuoteAmountToPay(uint64(blob.Data.Size), SatPerMegaByteDownload)

		// check for 50 sats payment
		cashu_header := c.GetHeader(xcashu.Xcashu)

		if cashu_header == "" {
			c.JSON(402, nil)
			return
		}

		token, err := cashu.DecodeToken(cashu_header)

		if err != nil {
			c.JSON(402, nil)
			return
		}

		if token.Amount() < amountToPay {
			c.JSON(402, "Too few sats")
			return
		}

		// TODO - Check if is the correct mint
		// TODO - Check if it is locked to the pubkey of the wallet

		_, err = wallet.Receive(token, false)
		if err != nil {
			c.JSON(402, "payment required")
			return
		}

		fileBytes, err := os.ReadFile(blob.Path)
		if err != nil {
			log.Printf(`os.ReadFile(blob.Path) %w`, err)
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

		c.Writer.Write(fileBytes)
	})

	r.HEAD("/:sha", func(c *gin.Context) {
		sha := c.Param("sha")
		hash, err := hex.DecodeString(sha)
		if err != nil {
			log.Printf(`hex.DecodeString(sha) %w`, err)
			c.JSON(500, "Opps! Server error")
			return
		}

		length, err := sqlite.GetBlobLength(hash)
		if err != nil {
			log.Printf(`hex.DecodeString(sha) %w`, err)
			c.JSON(500, "Opps! Server error")
			return
		}

		amount := xcashu.QuoteAmountToPay(length, SatPerMegaByteDownload)
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
		return
	})

}
