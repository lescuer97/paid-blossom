package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"ratasker/external/blossom"
	"ratasker/internal/database"
	"ratasker/internal/utils"
	"time"

	"github.com/elnosh/gonuts/cashu"
	w "github.com/elnosh/gonuts/wallet"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	DOCKER_ENV = "DOCKER"
)

func main() {
	ctx := context.Background()

	sqlite, err := database.DatabaseSetup(ctx, "migrations")
	defer sqlite.Db.Close()

	if err != nil {
		log.Panicf(`database.DatabaseSetup(ctx, "migrations"). %w`, err)
	}

	r := gin.Default()
	r.Use(cors.Default())

	string, err := utils.GetRastaskerHomeDirectory()
	if err != nil {
		log.Panicf(`utils.GetRastaskerHomeDirectory(). %w`, err)
	}

	pathToData := string + "/" + "data"

	err = utils.MakeSureFilePathExists(pathToData, "")
	if err != nil {
		log.Panicf(`utils.MakeSureFilePathExists(pathToData, ""). %w`, err)
	}

	pathToCashu := string + "/" + "cashu"

	err = utils.MakeSureFilePathExists(pathToCashu, "")
	if err != nil {
		log.Panicf(`utils.MakeSureFilePathExists(pathToData, ""). %w`, err)
	}

	// Setup wallet
	config := w.Config{
		WalletPath:     pathToCashu,
		CurrentMintURL: "http://localhost:8080",
	}
	wallet, err := w.LoadWallet(config)
	if err != nil {
		log.Panicf(`w.LoadWallet(config). %wa`, err)
	}

	r.GET("/:sha", func(c *gin.Context) {
		sha := c.Param("sha")

		// check for 50 sats payment
		cashu_header := c.GetHeader("cashu")

		if cashu_header == "" {
			c.JSON(402, "payment required")
			return

		}

		token, err := cashu.DecodeToken(cashu_header)

		if err != nil {
			c.JSON(402, "payment required")
			return
		}

		balance, err := wallet.Receive(*token, false)
		if err != nil {
			c.JSON(402, "payment required")
			return
		}

		log.Printf("\n Balance: %v \n", balance)

		// try to get blob
		hash, err := hex.DecodeString(sha)
		if err != nil {
			log.Panicf(`hex.DecodeString(sha) %w`, err)
		}

		blob, err := sqlite.GetBlob(hash)
		if err != nil {
			log.Panicf(`sqlite.GetBlob(hash) %w`, err)
		}

		fileBytes, err := os.ReadFile(blob.Path)
		if err != nil {
			log.Panicf(`os.ReadFile(blob.Path) %w`, err)
		}

		// check if sha256 is the same

		fileHash := sha256.Sum256(fileBytes)
		if sha != hex.EncodeToString(fileHash[:]) {
			log.Panic("HASHes are different")
		}

		c.Writer.Write(fileBytes)
	})

	r.PUT("/upload", func(c *gin.Context) {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(c.Request.Body)
		if err != nil {
			log.Panic(`buf.ReadFrom(c.Request.Body) %w`, err)
		}

		hash := sha256.Sum256(buf.Bytes())
		hashHex := hex.EncodeToString(hash[:])

		blob := blossom.Blob{
			Data: buf.Bytes(),
			Size: uint64(buf.Len()),
			Name: hex.EncodeToString(hash[:]),
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

	})

	log.Println("ratasker started in port 8070")
	r.Run(":8070")
}
