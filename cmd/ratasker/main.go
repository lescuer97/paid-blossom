package main

import (
	"context"
	"log"
	"os"
	"ratasker/external/nostr"
	"ratasker/internal/database"
	"ratasker/internal/io"
	"ratasker/internal/routes"
	"ratasker/internal/utils"
	"strings"

	"github.com/joho/godotenv"

	w "github.com/elnosh/gonuts/wallet"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Something happened while loading the env file")
	}
	homeDir, err := utils.GetRastaskerHomeDirectory()
	if err != nil {
		log.Panicf(`utils.GetRastaskerHomeDirectory(). %+v`, err)
	}

	sqlite, err := database.DatabaseSetup(ctx, homeDir, "migrations")
	defer sqlite.Db.Close()

	if err != nil {
		log.Panicf(`database.DatabaseSetup(ctx, "migrations"). %+v`, err)
	}

	r := gin.Default()

	pathToCashu := homeDir + "/" + "cashu"

	err = utils.MakeSureFilePathExists(pathToCashu, "")
	if err != nil {
		log.Panicf(`utils.MakeSureFilePathExists(pathToData, ""). %+v`, err)
	}

	fileHandler, err := io.MakeFileSystemHandler()
	if err != nil {
		log.Panicf(`io.MakeFileSystemHandler(). %+v`, err)
	}

	// Setup wallet
	config := w.Config{
		WalletPath:     pathToCashu,
		CurrentMintURL: os.Getenv("TRUSTED_MINT"),
	}

	wallet, err := w.LoadWallet(config)
	if err != nil {
		log.Panicf(`w.LoadWallet(config). %+va`, err)
	}

	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true, // Allow all origins
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		// AllowHeaders: []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Cashu", "X-Content-Length"},
		AllowHeaders:     []string{"Authorization", "*"},
		ExposeHeaders:    []string{"Content-Length", "*"},
		AllowCredentials: true,
	}))

	routes.RootRoutes(r, wallet, sqlite, fileHandler)
	routes.UploadRoutes(r, wallet, sqlite, fileHandler)

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
