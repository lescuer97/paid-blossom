package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"ratasker/external/nostr"
	"ratasker/internal/cashu"
	"ratasker/internal/core"
	"ratasker/internal/database"
	"ratasker/internal/io"
	"ratasker/internal/routes"
	"ratasker/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/nbd-wtf/go-nostr/nip19"
)

var (
	DOCKER_ENV = "DOCKER"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		c.Next()
	}
}

func main() {
	ctx := context.Background()

	_ = godotenv.Load()

	homeDir, err := utils.GetRastaskerHomeDirectory()
	if err != nil {
		log.Panicf(`utils.GetRastaskerHomeDirectory(). %+v`, err)
	}

	log.Println("Current home dir: ", homeDir)

	sqlite, err := database.DatabaseSetup(ctx, homeDir, database.EmbedMigrations)
	defer sqlite.Db.Close()

	if err != nil {
		log.Panicf(`database.DatabaseSetup(ctx, "migrations"). %+v`, err)
	}

	r := gin.Default()

	fileHandler, err := io.MakeFileSystemHandler()
	if err != nil {
		log.Panicf(`io.MakeFileSystemHandler(). %+v`, err)
	}

	domain := os.Getenv(utils.DOMAIN)
	if domain == "" {
		log.Panicf("\n Domain needs to be set\n")
	}
	seed := os.Getenv(core.SEED)
	if seed == "" {
		log.Panicf("\n No seed phrase set \n")
	}

	// try to load new wallet for test
	wallet, err := cashu.NewDBLocalWallet(seed, sqlite)
	if err != nil {
		log.Panicf(`cashu.NewDBLocalWallet(os.Getenv("SEED"), sqlite) %+va`, err)
	}

	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true, // Allow all origins
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		// AllowHeaders: []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Cashu", "X-Content-Length"},
		AllowHeaders:     []string{"Authorization", "*"},
		ExposeHeaders:    []string{"Content-Length", "*"},
		AllowCredentials: true,
	}))

	// get cost env variables
	uploadCostStr := os.Getenv(core.UPLOAD_COST_2MB)
	if uploadCostStr == "" {
		uploadCostStr = "0"
	}
	downloadCostStr := os.Getenv(core.DOWNLOAD_COST_2MB)
	if downloadCostStr == "" {
		downloadCostStr = "0"
	}

	owner_npub := os.Getenv(core.OWNER_NPUB)
	if owner_npub == "" {
		log.Panicf("no pubkey to send sats")
	}

	prefix, pubkey, err := nip19.Decode(owner_npub)
	if err != nil {
		log.Panicf("npub is incorrect. %+v", pubkey)
	}
	if prefix != "npub" {
		log.Panicf("no npub in the OWNER_NPUB variable. %+v", owner_npub)
	}

	uploadCost, err := strconv.ParseUint(uploadCostStr, 10, 32)
	if err != nil {
		log.Panicf(`Could not convert upload cost %+v`, err)
	}
	downloadCost, err := strconv.ParseUint(downloadCostStr, 10, 32)
	if err != nil {
		log.Panicf(`Could not convert upload cost %+v`, err)
	}

	routes.UploadRoutes(r, &wallet, sqlite, fileHandler, uploadCost)
	routes.RootRoutes(r, &wallet, sqlite, fileHandler, downloadCost)

	// rotate keys when expiration happens
	go func() {
		for {
			// Check if expiration of pubkey already happened
			now := time.Now().Add(1 * time.Minute).Unix()
			if now > int64(wallet.PubkeyVersion.Expiration) {
				func() {
					log.Println("begining key rotation")
					// rotate keys up
					tx, err := sqlite.BeginTransaction()
					if err != nil {
						log.Panicf("Could not get a lock on the db. %+v", err)
					}
					beforeRotation := wallet.PubkeyVersion
					// Ensure that the transaction is rolled back in case of a panic or error
					defer func() {
						if p := recover(); p != nil {
							log.Printf("\n Rolling back  because of failure %+v\n", p)
							wallet.PubkeyVersion = beforeRotation
							tx.Rollback()
						} else if err != nil {
							log.Println("Rolling back  because of error")
							wallet.PubkeyVersion = beforeRotation
							tx.Rollback()
						} else {
							err = tx.Commit()
							if err != nil {
								log.Printf("\n Failed to commit transaction: %v\n", err)
							}
							fmt.Println("Key rotation finished successfully")
						}
					}()

					// move locked proofs to valid swap
					err = core.RotateLockedProofs(&wallet, sqlite, tx)
					if err != nil {
						log.Panicf("core.RotateLockedProofs(&wallet, sqlite, tx). %+v", err)
					}

					err = wallet.RotatePubkey(tx, sqlite)
					if err != nil {
						log.Panicf("wallet.RotatePubkey(tx, sqlite). %+v", err)
					}

					log.Println("Finished key rotation")
				}()
				err := core.SpendSwappedProofs(&wallet, sqlite)
				if err != nil {
					log.Printf("core.SpendSwappedProofs(&wallet, sqlite). %+v ", err)
				}

			}

			time.Sleep(10 * time.Second)
		}

	}()

	log.Println("ratasker started in port 8070")
	r.Run("0.0.0.0:8070")
}

func NostrAutMiddleware() gin.HandlerFunc {
	authorizedKeys := os.Getenv("AUTHORIZED_KEYS")
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		event, err := nostr.ParseNostrHeader(authHeader)
		if err != nil {
			c.JSON(401, nostr.NotifMessage{Message: "Missing auth event"})
			return
		}

		if authorizedKeys != "" {
			if strings.Contains(authHeader, event.PubKey) {
				c.JSON(401, nostr.NotifMessage{Message: "unauthorized"})

			}
		}

		err = nostr.ValidateAuthEvent(event)
		if err != nil {
			c.JSON(401, nostr.NotifMessage{Message: "Invalid nostr event"})
			return
		}

		c.Set(utils.NOSTRAUTH, event)
		c.Next()
	}
}
